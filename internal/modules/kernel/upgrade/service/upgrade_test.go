package service

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckDetectsLatestRelease(t *testing.T) {
	client := fakeHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/repos/muidea/skill-hub/releases/latest" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		return textResponse(200, `{"tag_name":"v0.9.0"}`), nil
	})

	result, err := NewWithHTTPClient(client).Check(context.Background(), Options{
		CurrentVersion: "0.8.1",
		APIBaseURL:     "https://example.test",
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !result.UpdateAvailable || result.TargetVersion != "0.9.0" || result.Status != "update_available" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestUpgradeDownloadsVerifiesAndReplacesBinary(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "skill-hub-linux-amd64.tar.gz")
	writeReleaseArchive(t, archivePath, "0.9.0")
	archiveBytes, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	checksum := fmt.Sprintf("%x  skill-hub-linux-amd64.tar.gz\n", sha256.Sum256(archiveBytes))

	client := fakeHTTPClient(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.URL.Path == "/repos/muidea/skill-hub/releases/latest":
			return textResponse(200, `{"tag_name":"v0.9.0"}`), nil
		case strings.HasSuffix(r.URL.Path, "/skill-hub-linux-amd64.tar.gz"):
			return bytesResponse(200, archiveBytes), nil
		case strings.HasSuffix(r.URL.Path, "/skill-hub-linux-amd64.sha256"):
			return textResponse(200, checksum), nil
		default:
			return textResponse(404, "not found"), nil
		}
	})

	installPath := filepath.Join(tempDir, "installed", "skill-hub")
	if err := os.MkdirAll(filepath.Dir(installPath), 0755); err != nil {
		t.Fatalf("mkdir install dir: %v", err)
	}
	if err := os.WriteFile(installPath, []byte("old"), 0755); err != nil {
		t.Fatalf("write old binary: %v", err)
	}

	result, err := NewWithHTTPClient(client).Upgrade(context.Background(), Options{
		CurrentVersion:  "0.8.1",
		InstallPath:     installPath,
		GOOS:            "linux",
		GOARCH:          "amd64",
		APIBaseURL:      "https://example.test",
		DownloadBaseURL: "https://example.test",
		SkipAgentSkills: true,
		NoRestartServe:  true,
	})
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if result.Status != "upgraded" || result.ArchiveName != "skill-hub-linux-amd64.tar.gz" {
		t.Fatalf("unexpected result: %+v", result)
	}
	content, err := os.ReadFile(installPath)
	if err != nil {
		t.Fatalf("read installed binary: %v", err)
	}
	if !strings.Contains(string(content), "skill-hub version 0.9.0") {
		t.Fatalf("installed binary was not replaced: %q", string(content))
	}
}

func TestUpgradeDryRunDoesNotReplaceBinary(t *testing.T) {
	client := fakeHTTPClient(func(r *http.Request) (*http.Response, error) {
		return textResponse(200, `{"tag_name":"v0.9.0"}`), nil
	})

	tempDir := t.TempDir()
	installPath := filepath.Join(tempDir, "skill-hub")
	if err := os.WriteFile(installPath, []byte("old"), 0755); err != nil {
		t.Fatalf("write old binary: %v", err)
	}

	result, err := NewWithHTTPClient(client).Upgrade(context.Background(), Options{
		CurrentVersion: "0.8.1",
		InstallPath:    installPath,
		GOOS:           "linux",
		GOARCH:         "amd64",
		APIBaseURL:     "https://example.test",
		DryRun:         true,
	})
	if err != nil {
		t.Fatalf("upgrade dry-run: %v", err)
	}
	if result.Status != "planned" {
		t.Fatalf("unexpected status: %+v", result)
	}
	content, err := os.ReadFile(installPath)
	if err != nil {
		t.Fatalf("read install path: %v", err)
	}
	if string(content) != "old" {
		t.Fatalf("dry-run replaced binary: %q", string(content))
	}
}

func TestVerifySHA256FileChecksArchiveContent(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "asset.tar.gz")
	if err := os.WriteFile(archivePath, []byte("archive"), 0644); err != nil {
		t.Fatalf("write archive: %v", err)
	}
	sum := sha256.Sum256([]byte("archive"))
	checksumPath := filepath.Join(tempDir, "asset.sha256")
	if err := os.WriteFile(checksumPath, []byte(fmt.Sprintf("%x  asset.tar.gz\n", sum)), 0644); err != nil {
		t.Fatalf("write checksum: %v", err)
	}
	if err := verifySHA256File(archivePath, checksumPath); err != nil {
		t.Fatalf("verify archive: %v", err)
	}
}

func writeReleaseArchive(t *testing.T, archivePath string, version string) {
	t.Helper()
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	binary := fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo \"skill-hub version %s (commit: test, built: now)\"; exit 0; fi\n", version)
	if err := tarWriter.WriteHeader(&tar.Header{
		Name: "skill-hub",
		Mode: 0755,
		Size: int64(len(binary)),
	}); err != nil {
		t.Fatalf("write binary header: %v", err)
	}
	if _, err := tarWriter.Write([]byte(binary)); err != nil {
		t.Fatalf("write binary: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func fakeHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func textResponse(status int, body string) *http.Response {
	return bytesResponse(status, []byte(body))
}

func bytesResponse(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
	}
}

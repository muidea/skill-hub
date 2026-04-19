package cli

import (
	"context"
	"strings"
	"testing"
	"time"

	serverservice "github.com/muidea/skill-hub/internal/modules/kernel/server/service"
)

func TestRunServeStopsOnKeyboardInput(t *testing.T) {
	prevInput := serveInputReader
	prevRunServer := serveRunServer
	prevHost := serveHost
	prevPort := servePort
	prevSecretKey := serveSecretKey
	prevOpenBrowser := serveOpenBrowser
	t.Cleanup(func() {
		serveInputReader = prevInput
		serveRunServer = prevRunServer
		serveHost = prevHost
		servePort = prevPort
		serveSecretKey = prevSecretKey
		serveOpenBrowser = prevOpenBrowser
	})

	serveInputReader = strings.NewReader("q\n")
	serveHost = "127.0.0.1"
	servePort = 5525
	serveSecretKey = "write-secret"
	serveOpenBrowser = false

	stopped := make(chan struct{})
	serveRunServer = func(ctx context.Context, cfg serverservice.Config) error {
		if cfg.SecretKey != "write-secret" {
			t.Fatalf("expected secret key to be passed to server, got %q", cfg.SecretKey)
		}
		<-ctx.Done()
		close(stopped)
		return nil
	}

	done := make(chan error, 1)
	go func() {
		done <- runServe()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runServe() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runServe() did not stop after keyboard input")
	}

	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("server runner did not observe context cancellation")
	}
}

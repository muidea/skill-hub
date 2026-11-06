package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/muidea/skill-hub/pkg/spec"
	"github.com/spf13/cobra"
)

func TestPullDiagnosticBridgeOutput(t *testing.T) {
	d := &spec.PullDiagnostic{Kind: "blocked_dirty", RepositoryPath: "/remote/repository", OverlapKnown: true, Behind: 2,
		Files: []spec.PullFileChange{{Path: "skills/demo/SKILL.md", IndexStatus: "M", WorktreeStatus: " ", OverlapsRemote: true,
			StagedDiff: "-base\n+my-local-content\n", RemoteDiff: "-base\n+their-remote-content\n"}}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/health" {
			fmt.Fprint(w, `{"code":"OK"}`)
			return
		}
		if r.URL.Path != "/api/v1/skill-repository/sync" {
			t.Errorf("unexpected bridge request: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": "GIT_WORKTREE_DIRTY", "message": "拉取已暂停", "details": map[string]any{"pull": d}})
	}))
	defer server.Close()
	env := []string{"SKILL_HUB_DISABLE_SERVICE_BRIDGE=0", "SKILL_HUB_SERVICE_URL=" + server.URL}
	result, stderr, err := runCLIProcess(t, t.TempDir(), t.TempDir(), env, "pull", "--json")
	if err == nil || result["status"] != "failed" || result["behind"] != float64(2) {
		t.Fatalf("bridge failure lost summary: %v, %s, %v", result, stderr, err)
	}
	failure := result["command_error"].(map[string]any)
	if failure["code"] != "GIT_WORKTREE_DIRTY" || failure["details"].(map[string]any)["pull"] == nil || strings.Contains(stderr, "Usage:") {
		t.Fatalf("bridge failure lost diagnostics: %v, %s", result, stderr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCLIJSONProcess$", "--", "pull")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "SKILL_HUB_JSON_TEST_PROCESS=1", "SKILL_HUB_HOME="+t.TempDir())
	cmd.Env = append(cmd.Env, env...)
	reader, writer, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatal(pipeErr)
	}
	defer reader.Close()
	defer writer.Close()
	cmd.Stdin = reader
	var stdout, diagnostics bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &diagnostics
	if err := cmd.Run(); err == nil || ctx.Err() != nil {
		t.Fatalf("expected noninteractive bridge failure: %v, %v", err, ctx.Err())
	}
	for _, want := range []string{"/remote/repository", "skills/demo/SKILL.md", "my-local-content", "their-remote-content", "潜在冲突"} {
		if !strings.Contains(diagnostics.String(), want) {
			t.Fatalf("missing %q from bridge text output: %s", want, diagnostics.String())
		}
	}
	if strings.Contains(diagnostics.String(), "Usage:") || strings.Contains(diagnostics.String(), "Error:") {
		t.Fatalf("duplicate error/usage obscures diagnostic: %s", diagnostics.String())
	}
}

func TestRenderPullDiagnosticKinds(t *testing.T) {
	for _, tc := range []struct {
		name, kind, want string
		stash            bool
	}{
		{"dirty", "blocked_dirty", "未发现同路径修改", false},
		{"conflict", "merge_conflict", "--force 不会解决已有冲突", false},
		{"divergent", "divergent", "--force 不会解决提交分叉", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			previous := activeOutput
			defer func() { activeOutput = previous }()
			var output bytes.Buffer
			cmd := &cobra.Command{Use: "pull"}
			cmd.SetErr(&output)
			activeOutput = &commandOutput{command: cmd}
			d := &spec.PullDiagnostic{Kind: tc.kind, RepositoryPath: "/repo with 'quotes'", OverlapKnown: true, Files: []spec.PullFileChange{}}
			if tc.stash {
				d.StashRef = strings.Repeat("a", 40)
			}
			renderPullDiagnostic(d)
			if !strings.Contains(output.String(), tc.want) {
				t.Fatalf("missing %q: %s", tc.want, output.String())
			}
			if tc.stash && (strings.Contains(output.String(), "工作区内容保持不变") || !strings.Contains(output.String(), d.StashRef)) {
				t.Fatalf("stash effects misstated: %s", output.String())
			}
		})
	}
}

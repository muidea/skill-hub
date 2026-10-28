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
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// Run the real command tree in a fresh process, with an open but unwritten
// stdin. A regression to interactive reads must time out instead of being
// masked by EOF or by state retained between Cobra executions.
func TestCLIJSONProcess(t *testing.T) {
	if os.Getenv("SKILL_HUB_JSON_TEST_PROCESS") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"skill-hub"}, os.Args[i+1:]...)
			break
		}
	}
	if err := Execute(); err != nil {
		if !ErrorReported(err) {
			fmt.Fprintln(os.Stderr, FormatError(err))
		}
		os.Exit(1)
	}
	os.Exit(0)
}

func runCLIProcess(t *testing.T, home, project string, env []string, args ...string) (map[string]any, string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestCLIJSONProcess$", "--"}, args...)...)
	cmd.Dir = project
	cmd.Env = append(os.Environ(), "SKILL_HUB_JSON_TEST_PROCESS=1", "SKILL_HUB_HOME="+home, "SKILL_HUB_DISABLE_SERVICE_BRIDGE=1")
	cmd.Env = append(cmd.Env, env...)
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	cmd.Stdin = reader
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("CLI waited for input: %v\nstdout=%s\nstderr=%s", args, stdout.String(), stderr.String())
	}
	var result map[string]any
	if decodeErr := json.Unmarshal(stdout.Bytes(), &result); decodeErr != nil {
		t.Fatalf("whole stdout must be one JSON object: %v\nargs=%v\nstdout=%q\nstderr=%s", decodeErr, args, stdout.String(), stderr.String())
	}
	return result, stderr.String(), err
}

func requireCLIJSON(t *testing.T, home, project string, args ...string) map[string]any {
	t.Helper()
	result, stderr, err := runCLIProcess(t, home, project, nil, args...)
	if err != nil {
		t.Fatalf("CLI %v failed: %v\nresult=%v\nstderr=%s", args, err, result, stderr)
	}
	return result
}

func writeJSONTestSkill(t *testing.T, project, id string) string {
	t.Helper()
	dir := filepath.Join(project, ".agents", "skills", id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "SKILL.md")
	content := fmt.Sprintf("---\nname: %s\ndescription: Test JSON CLI integration.\nmetadata:\n  version: \"1.0.0\"\n  author: tester\n---\n# Test\n", id)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func jsonTestWorkspace(t *testing.T) (string, string) {
	t.Helper()
	home, project := t.TempDir(), t.TempDir()
	requireCLIJSON(t, home, project, "init", "--json")
	return home, project
}

func TestCLIJSONFirstRegister(t *testing.T) {
	home, project := jsonTestWorkspace(t)
	path := writeJSONTestSkill(t, project, "manual-skill")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	first, stderr, err := runCLIProcess(t, home, project, nil, "register", "manual-skill", "--json")
	if err != nil || first["skill_id"] != "manual-skill" || first["registered"] != true {
		t.Fatalf("unexpected first register: result=%v stderr=%s err=%v", first, stderr, err)
	}
	if !strings.Contains(stderr, "已创建项目工作区") || strings.Contains(stderr, "[Y/n]") {
		t.Fatalf("progress must go to stderr without a prompt: %q", stderr)
	}
	second := requireCLIJSON(t, home, project, "--json", "register", "manual-skill")
	if second["registered"] != false {
		t.Fatalf("second register must refresh existing state: %v", second)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("register changed SKILL.md: %v", err)
	}
	status := requireCLIJSON(t, home, project, "status", "--json")
	if status["skill_count"] != float64(1) {
		t.Fatalf("registered skill absent from status: %v", status)
	}
}

func TestCLIJSONEmptyProjectRegistrationReceipt(t *testing.T) {
	home, project := jsonTestWorkspace(t)
	first := requireCLIJSON(t, home, project, "project", "register", "--request-id", "lost-response", "--json")
	if first["status"] != "succeeded" || first["finished"] != true || first["executed"] != true || first["project_path"] != project {
		t.Fatalf("unexpected receipt: %v", first)
	}
	effects := first["result"].(map[string]any)
	if effects["project_created"] != true || effects["skill_count"] != float64(0) {
		t.Fatalf("unexpected original effects: %v", first)
	}
	status := requireCLIJSON(t, home, project, "status", "--json")
	if status["skill_count"] != float64(0) {
		t.Fatalf("empty project not registered: %v", status)
	}
	// Change current state, then query from a different directory. The response
	// must still describe the original empty registration, not today's status.
	writeJSONTestSkill(t, project, "later-skill")
	requireCLIJSON(t, home, project, "register", "later-skill", "--json")
	for _, args := range [][]string{
		{"project", "register", "--request-id", "lost-response", "--json"},
		{"project", "register", project, "--request-id", "lost-response", "--json"},
		{"project", "register", "--query", "--request-id", "lost-response", "--json"},
	} {
		got := requireCLIJSON(t, home, project, args...)
		if !reflect.DeepEqual(first, got) {
			t.Fatalf("original receipt changed: got=%v, want=%v", got, first)
		}
	}
	got := requireCLIJSON(t, home, t.TempDir(), "project", "register", "--query", "--request-id", "lost-response", "--json")
	if !reflect.DeepEqual(first, got) {
		t.Fatalf("query used cwd instead of request ID: %v", got)
	}
	other := t.TempDir()
	conflict, _, err := runCLIProcess(t, home, other, nil, "project", "register", "--request-id", "lost-response", "--json")
	if err == nil || conflict["command_error"].(map[string]any)["code"] != "REQUEST_CONFLICT" {
		t.Fatalf("expected request conflict: %v, %v", conflict, err)
	}
	if _, err := os.Stat(filepath.Join(other, ".agents")); !os.IsNotExist(err) {
		t.Fatal("conflict created another workspace")
	}
	// Receipt queries must work even when current state/config become unreadable.
	if err := os.WriteFile(filepath.Join(home, "state.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(home, "config.yaml")); err != nil {
		t.Fatal(err)
	}
	got = requireCLIJSON(t, home, other, "project", "register", "--query", "--request-id", "lost-response", "--json")
	if !reflect.DeepEqual(first, got) {
		t.Fatalf("receipt query depended on current state: %v", got)
	}
}

func TestCLIJSONEmptyProjectRegistrationErrors(t *testing.T) {
	home, project := jsonTestWorkspace(t)
	for _, tc := range []struct {
		args []string
		code string
	}{
		{[]string{"project", "register", "--json"}, "INVALID_INPUT"},
		{[]string{"project", "register", "--request-id", "../invalid", "--json"}, "INVALID_INPUT"},
		{[]string{"project", "register", "--query", "--request-id", "missing", "--json"}, "REQUEST_NOT_FOUND"},
		{[]string{"project", "register", ".", "--query", "--request-id", "missing", "--json"}, "INVALID_INPUT"},
	} {
		got, _, err := runCLIProcess(t, home, project, nil, tc.args...)
		if err == nil || got["command_error"].(map[string]any)["code"] != tc.code {
			t.Fatalf("%v = %v, %v", tc.args, got, err)
		}
	}
	if _, err := os.Stat(filepath.Join(project, ".agents")); !os.IsNotExist(err) {
		t.Fatal("invalid request created workspace")
	}
	failed, _, err := runCLIProcess(t, home, project, nil, "project", "register", filepath.Join(project, "missing"), "--request-id", "failed-request", "--json")
	if err == nil || failed["status"] != "failed" || failed["finished"] != true {
		t.Fatalf("failure not recorded: %v, %v", failed, err)
	}
	query := requireCLIJSON(t, home, project, "project", "register", "--query", "--request-id", "failed-request", "--json")
	if !reflect.DeepEqual(failed, query) {
		t.Fatalf("failed query lost original error: %v, want %v", query, failed)
	}
}

func TestCLIJSONEmptyProjectRegistrationSurvivesLostStdout(t *testing.T) {
	home, project := jsonTestWorkspace(t)
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = reader.Close() // No consumer can receive the registration response.
	defer writer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCLIJSONProcess$", "--", "project", "register", "--request-id", "broken-stdout", "--json")
	cmd.Dir = project
	cmd.Env = append(os.Environ(), "SKILL_HUB_JSON_TEST_PROCESS=1", "SKILL_HUB_HOME="+home, "SKILL_HUB_DISABLE_SERVICE_BRIDGE=1")
	cmd.Stdout = writer
	if err := cmd.Run(); err == nil || ctx.Err() != nil {
		t.Fatalf("expected lost stdout, got err=%v, timeout=%v", err, ctx.Err())
	}
	receipt := requireCLIJSON(t, home, project, "project", "register", "--query", "--request-id", "broken-stdout", "--json")
	if receipt["status"] != "succeeded" || receipt["result"].(map[string]any)["project_created"] != true {
		t.Fatalf("lost response lost receipt: %v", receipt)
	}
	// Neither a replay nor query may recreate a project removed after that request.
	if err := os.Remove(filepath.Join(home, "state.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(project); err != nil {
		t.Fatal(err)
	}
	replayed := requireCLIJSON(t, home, t.TempDir(), "project", "register", project, "--request-id", "broken-stdout", "--json")
	if !reflect.DeepEqual(receipt, replayed) {
		t.Fatalf("lost response was not replayed: %v", replayed)
	}
	if _, err := os.Stat(project); !os.IsNotExist(err) {
		t.Fatal("replay recreated the removed project")
	}
	if _, err := os.Stat(filepath.Join(home, "state.json")); !os.IsNotExist(err) {
		t.Fatal("replay registered project again")
	}
}

func TestCLIJSONEmptyProjectRegistrationConcurrent(t *testing.T) {
	for _, sameID := range []bool{true, false} {
		t.Run(fmt.Sprintf("same-id=%t", sameID), func(t *testing.T) {
			home, project := jsonTestWorkspace(t)
			ids := make([]string, 6)
			var requests sync.WaitGroup
			for i := range ids {
				id := "concurrent-request"
				if !sameID {
					id = fmt.Sprintf("concurrent-request-%d", i)
				}
				ids[i] = id
				requests.Go(func() {
					result, _, err := runCLIProcess(t, home, project, nil, "project", "register", "--request-id", id, "--json")
					if err != nil && result["command_error"].(map[string]any)["code"] != "REQUEST_IN_PROGRESS" {
						t.Errorf("concurrent CLI request: %v, %v", result, err)
					}
				})
			}
			requests.Wait()
			if sameID {
				ids = ids[:1]
			}
			created := 0
			for _, id := range ids {
				receipt := requireCLIJSON(t, home, project, "project", "register", "--query", "--request-id", id, "--json")
				if receipt["status"] != "succeeded" {
					t.Fatalf("request did not finish: %v", receipt)
				}
				if receipt["result"].(map[string]any)["project_created"] == true {
					created++
				}
			}
			if created != 1 {
				t.Fatalf("expected exactly one original project creation, got %d", created)
			}
		})
	}
}

func TestCLIJSONErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		code string
	}{
		{"missing argument", []string{"register", "--json"}, "INVALID_INPUT"},
		{"unknown flag", []string{"register", "test", "--bogus", "--json"}, "INVALID_INPUT"},
		{"unknown command", []string{"does-not-exist", "--json"}, "INVALID_INPUT"},
		{"uninitialized", []string{"register", "test", "--json"}, "CONFIG_NOT_FOUND"},
		{"invalid bool", []string{"register", "test", "--json=invalid"}, "INVALID_INPUT"},
		{"missing required flag", []string{"sync-copies", "--json"}, "INVALID_INPUT"},
		{"completion", []string{"__complete", "register", "--json"}, "INVALID_INPUT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, stderr, err := runCLIProcess(t, t.TempDir(), t.TempDir(), nil, tc.args...)
			if err == nil {
				t.Fatal("expected nonzero exit status")
			}
			failure, ok := result["command_error"].(map[string]any)
			if !ok || failure["code"] != tc.code {
				t.Fatalf("unexpected error: %v", result)
			}
			if strings.Contains(stderr, "Usage:") || strings.Contains(stderr, "Error:") {
				t.Fatalf("error rendered a second time: %q", stderr)
			}
		})
	}
}

func TestCLIJSONReadOnlyAndDryRunDoNotCreateWorkspace(t *testing.T) {
	home, project := jsonTestWorkspace(t)
	statePath := filepath.Join(home, "state.json")
	before, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	result, _, err := runCLIProcess(t, home, project, nil, "status", "--json")
	if err == nil || result["command_error"].(map[string]any)["code"] != "PROJECT_NOT_FOUND" {
		t.Fatalf("expected missing project: %v, %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(project, ".agents")); !os.IsNotExist(err) {
		t.Fatalf("status created workspace: %v", err)
	}
	writeJSONTestSkill(t, project, "preview-skill")
	preview := requireCLIJSON(t, home, project, "import", ".agents/skills", "--dry-run", "--json")
	if preview["discovered"] != float64(1) {
		t.Fatalf("unexpected import preview: %v", preview)
	}
	after, err := os.ReadFile(statePath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("read-only/dry-run invocation changed state: %v", err)
	}
}

func TestCLIJSONImportPartialFailure(t *testing.T) {
	home, project := jsonTestWorkspace(t)
	writeJSONTestSkill(t, project, "valid-skill")
	invalid := writeJSONTestSkill(t, project, "invalid-skill")
	if err := os.WriteFile(invalid, []byte("# Invalid skill\n"), 0644); err != nil {
		t.Fatal(err)
	}
	result, _, err := runCLIProcess(t, home, project, nil, "import", ".agents/skills", "--json")
	if err == nil || result["registered"] != float64(1) || result["failed"] != float64(1) || result["command_error"] == nil {
		t.Fatalf("partial results must survive the error: %v, %v", result, err)
	}
}

func TestCLIJSONCommandCoverage(t *testing.T) {
	home, project := jsonTestWorkspace(t)
	requireCLIJSON(t, home, project, "create", "sample", "--description", "Sample skill", "--json")
	for _, args := range [][]string{
		{"init", "--json"},
		{"list", "--json"},
		{"repo", "--json", "list"},
		{"git", "status", "--json"},
		{"git", "remote", "--json"},
		{"validate", "sample", "--json"},
		{"validate", "--pattern", "absent-*", "--json"},
		{"apply", "--pattern", "absent-*", "--dry-run", "--json"},
		{"remove", "--pattern", "absent-*", "--json"},
		{"feedback", "--pattern", "absent-*", "--dry-run", "--json"},
		{"prune", "--json"},
		{"--version", "--json"},
		{"register", "--help", "--json"},
		{"help", "register", "--json"},
		{"repo", "--json"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			requireCLIJSON(t, home, project, args...)
		})
	}
	result, _, err := runCLIProcess(t, home, project, nil, "remove", "sample", "--json")
	if err == nil || result["command_error"].(map[string]any)["code"] != "USER_INPUT_ERROR" {
		t.Fatalf("remove must require explicit confirmation: %v %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(project, ".agents", "skills", "sample", "SKILL.md")); err != nil {
		t.Fatalf("unconfirmed remove changed skill: %v", err)
	}
	requireCLIJSON(t, home, project, "remove", "sample", "--yes", "--json")
}

func TestCLIJSONApplyPartialFailure(t *testing.T) {
	home, project := jsonTestWorkspace(t)
	writeJSONTestSkill(t, project, "unarchived")
	requireCLIJSON(t, home, project, "register", "unarchived", "--json")
	result, _, err := runCLIProcess(t, home, project, nil, "apply", "--json")
	if err == nil || result["command_error"] == nil {
		t.Fatalf("item errors must produce a nonzero exit: %v, %v", result, err)
	}
	items := result["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["status"] != "error" {
		t.Fatalf("failed apply item lost: %v", result)
	}
}

func TestCLIJSONAuditArtifact(t *testing.T) {
	home, project := jsonTestWorkspace(t)
	writeJSONTestSkill(t, project, "sample")
	requireCLIJSON(t, home, project, "register", "sample", "--json")
	path := filepath.Join(project, "audit.json")
	result, stderr, _ := runCLIProcess(t, home, project, nil, "audit", "--format", "json", "--output", path)
	if result["output"] != path || result["report"] == nil || !strings.Contains(stderr, "审计报告已写入") {
		t.Fatalf("artifact response=%v stderr=%q", result, stderr)
	}
	data, err := os.ReadFile(path)
	if err != nil || !json.Valid(data) {
		t.Fatalf("invalid audit artifact: %s, %v", data, err)
	}
}

func TestRequestsJSON(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"--json", "register", "sample"}, true},
		{[]string{"register", "sample", "--json=false"}, false},
		{[]string{"register", "sample", "--json", "--json=false"}, false},
		{[]string{"register", "sample", "--", "--json"}, false},
		{[]string{"create", "sample", "--description", "--json"}, false},
		{[]string{"create", "sample", "--description=--json", "--json"}, true},
		{[]string{"git", "commit", "-m", "--json"}, false},
		{[]string{"git", "commit", "-m--json"}, false},
		{[]string{"git", "commit", "-mmessage", "--json"}, true},
		{[]string{"audit", "--format", "JSON"}, true},
		{[]string{"audit", "--format=json", "--format=markdown"}, false},
		{[]string{"register", "sample", "--bogus", "--json"}, true},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			selected, _, _ := rootCmd.Find(tc.args)
			if got := requestsJSON(selected, tc.args); got != tc.want {
				t.Fatalf("requestsJSON(%v)=%v, want %v", tc.args, got, tc.want)
			}
		})
	}
}

func TestCLIJSONServiceBridge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/health":
			fmt.Fprint(w, `{"code":"OK"}`)
		case "/api/v1/project-skills/register":
			fmt.Fprint(w, `{"code":"OK","data":{"item":{"skill_id":"remote-skill","registered":true}}}`)
		default:
			t.Errorf("unexpected bridge request: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	home, project := t.TempDir(), t.TempDir()
	result, _, err := runCLIProcess(t, home, project, []string{"SKILL_HUB_DISABLE_SERVICE_BRIDGE=0", "SKILL_HUB_SERVICE_URL=" + server.URL}, "register", "remote-skill", "--json")
	if err != nil || result["skill_id"] != "remote-skill" {
		t.Fatalf("bridge result: %v, %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(home, "state.json")); !os.IsNotExist(err) {
		t.Fatalf("bridge initialized local state: %v", err)
	}
}

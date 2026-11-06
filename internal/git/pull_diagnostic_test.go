package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
)

func newPullDiagnosticFixture(t *testing.T) (*SkillRepository, string, string) {
	t.Helper()
	root := t.TempDir()
	remote, seed, local := filepath.Join(root, "remote.git"), filepath.Join(root, "seed"), filepath.Join(root, "local")
	runGitCommand(t, root, "init", "--bare", "--initial-branch=main", remote)
	runGitCommand(t, root, "init", "--initial-branch=main", seed)
	runGitCommand(t, seed, "config", "user.email", "test@example.com")
	runGitCommand(t, seed, "config", "user.name", "Test")
	writeSystemGitCommit(t, seed, "shared.txt", "base\n", "initial")
	writeSystemGitCommit(t, seed, "old name.txt", "rename base\n", "rename base")
	writeSystemGitCommit(t, seed, "binary.dat", "base\x00binary", "binary base")
	runGitCommand(t, seed, "remote", "add", "origin", remote)
	runGitCommand(t, seed, "push", "-u", "origin", "main")
	runGitCommand(t, root, "clone", remote, local)
	runGitCommand(t, local, "config", "user.email", "test@example.com")
	runGitCommand(t, local, "config", "user.name", "Test")
	repo, err := NewRepository(local)
	if err != nil {
		t.Fatal(err)
	}
	return &SkillRepository{repo: repo}, seed, local
}

func pullTestWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func requirePullDiagnostic(t *testing.T, err error, code errors.ErrorCode) *spec.PullDiagnostic {
	t.Helper()
	if !errors.IsCode(err, code) {
		t.Fatalf("error = %v, want %s", err, code)
	}
	d, ok := errors.Details(err)["pull"].(*spec.PullDiagnostic)
	if !ok || d == nil {
		t.Fatalf("missing typed diagnostic: %v", err)
	}
	return d
}

func TestPullDiagnosticDirtyChangesAndRemoteOverlap(t *testing.T) {
	sr, seed, local := newPullDiagnosticFixture(t)
	writeSystemGitCommit(t, seed, "shared.txt", "remote-content\n", "remote update")
	runGitCommand(t, seed, "push")
	pullTestWrite(t, filepath.Join(local, "shared.txt"), "staged-content\n")
	runGitCommand(t, local, "add", "shared.txt")
	pullTestWrite(t, filepath.Join(local, "shared.txt"), "unstaged-content\n")
	weird := "untracked [file]\nwith space.txt"
	pullTestWrite(t, filepath.Join(local, weird), "untracked")
	before := runGitCommand(t, local, "status", "--porcelain=v1", "-z")
	head := runGitCommand(t, local, "rev-parse", "HEAD")
	t.Setenv("GIT_EXTERNAL_DIFF", "false") // Diagnostics must bypass external diff helpers.
	d := requirePullDiagnostic(t, sr.Sync(), errors.ErrGitDirty)
	if d.Kind != "blocked_dirty" || d.Behind != 1 || !d.OverlapKnown || len(d.Files) != 2 || len(d.InspectionErrors) != 0 {
		t.Fatalf("unexpected diagnostic: %+v", d)
	}
	for _, file := range d.Files {
		if file.Path == "shared.txt" {
			if !file.OverlapsRemote || !strings.Contains(file.StagedDiff, "staged-content") || !strings.Contains(file.UnstagedDiff, "unstaged-content") || !strings.Contains(file.RemoteDiff, "remote-content") {
				t.Fatalf("missing staged/worktree/remote content: %+v", file)
			}
		} else if file.Path != weird || file.IndexStatus != "?" || file.OverlapsRemote {
			t.Fatalf("incorrect NUL-safe untracked status: %+v", file)
		}
	}
	if runGitCommand(t, local, "status", "--porcelain=v1", "-z") != before || runGitCommand(t, local, "rev-parse", "HEAD") != head {
		t.Fatal("diagnosis changed local state")
	}
}

func TestPullDiagnosticNonOverlapAndRename(t *testing.T) {
	for _, rename := range []bool{false, true} {
		t.Run(map[bool]string{false: "unrelated", true: "rename"}[rename], func(t *testing.T) {
			sr, seed, local := newPullDiagnosticFixture(t)
			remotePath := "shared.txt"
			if rename {
				remotePath = "old name.txt"
				runGitCommand(t, local, "mv", "old name.txt", "renamed [file].txt")
			} else {
				pullTestWrite(t, filepath.Join(local, "old name.txt"), "local unrelated\n")
			}
			writeSystemGitCommit(t, seed, remotePath, "remote-change\n", "remote")
			runGitCommand(t, seed, "push")
			d := requirePullDiagnostic(t, sr.Sync(), errors.ErrGitDirty)
			if len(d.Files) != 1 || d.Files[0].OverlapsRemote != rename {
				t.Fatalf("incorrect overlap: %+v", d)
			}
			if rename && (d.Files[0].OriginalPath != "old name.txt" || !strings.Contains(d.Files[0].RemoteDiff, "remote-change")) {
				t.Fatalf("rename source not diagnosed: %+v", d.Files[0])
			}
		})
	}
}

func TestPullDiagnosticExistingConflict(t *testing.T) {
	for _, binary := range []bool{false, true} {
		t.Run(map[bool]string{false: "text", true: "binary"}[binary], func(t *testing.T) {
			sr, seed, local := newPullDiagnosticFixture(t)
			name, ours, theirs := "shared.txt", "ours\n", "theirs\n"
			if binary {
				name, ours, theirs = "binary.dat", "ours\x00binary", "theirs\x00binary"
			}
			writeSystemGitCommit(t, local, name, ours, "ours")
			writeSystemGitCommit(t, seed, name, theirs, "theirs")
			runGitCommand(t, seed, "push")
			runGitCommand(t, local, "fetch")
			cmd := exec.Command("git", "-C", local, "merge", "origin/main")
			if output, err := cmd.CombinedOutput(); err == nil {
				t.Fatalf("expected real conflict: %s", output)
			}
			before := runGitCommand(t, local, "status", "--porcelain=v1", "-z")
			d := requirePullDiagnostic(t, sr.SyncWithOptions(SyncOptions{Force: true}), errors.ErrGitConflict)
			if d.Kind != "merge_conflict" || len(d.Files) != 1 || d.Files[0].ConflictType != "UU" {
				t.Fatalf("incorrect conflict: %+v", d)
			}
			if binary {
				if !d.Files[0].Binary {
					t.Fatalf("binary conflict not identified: %+v", d.Files[0])
				}
			} else if !strings.Contains(d.Files[0].ConflictDiff, "<<<<<<<") || !strings.Contains(d.Files[0].ConflictDiff, "ours") || !strings.Contains(d.Files[0].ConflictDiff, "theirs") {
				t.Fatalf("missing actual conflict blocks: %+v", d.Files[0])
			}
			if before != runGitCommand(t, local, "status", "--porcelain=v1", "-z") || runGitCommand(t, local, "stash", "list") != "" {
				t.Fatal("diagnosis/--force modified an unresolved conflict")
			}
		})
	}
}

func TestPullDiagnosticModifyDeleteConflict(t *testing.T) {
	for _, localDeletes := range []bool{false, true} {
		t.Run(map[bool]string{false: "remote_deleted", true: "local_deleted"}[localDeletes], func(t *testing.T) {
			sr, seed, local := newPullDiagnosticFixture(t)
			deleted, modified, code := seed, local, "UD"
			if localDeletes {
				deleted, modified, code = local, seed, "DU"
			}
			runGitCommand(t, deleted, "rm", "shared.txt")
			runGitCommand(t, deleted, "commit", "-m", "delete shared")
			writeSystemGitCommit(t, modified, "shared.txt", "modified-side-content\n", "modify shared")
			runGitCommand(t, seed, "push")
			runGitCommand(t, local, "fetch")
			if output, err := exec.Command("git", "-C", local, "merge", "origin/main").CombinedOutput(); err == nil {
				t.Fatalf("expected modify/delete conflict: %s", output)
			}
			before := runGitCommand(t, local, "status", "--porcelain=v1", "-z")
			d := requirePullDiagnostic(t, sr.Sync(), errors.ErrGitConflict)
			if len(d.Files) != 1 || d.Files[0].ConflictType != code || !strings.Contains(d.Files[0].ConflictDiff, "modified-side-content") {
				t.Fatalf("missing modify/delete conflict content: %+v", d.Files)
			}
			if len(d.InspectionErrors) != 0 || runGitCommand(t, local, "status", "--porcelain=v1", "-z") != before {
				t.Fatalf("modify/delete diagnostic incomplete or changed state: %+v", d)
			}
		})
	}
}

func TestPullDiagnosticDivergenceAndRetainedStash(t *testing.T) {
	sr, seed, local := newPullDiagnosticFixture(t)
	writeSystemGitCommit(t, local, "old name.txt", "local commit\n", "local-only-commit")
	writeSystemGitCommit(t, seed, "shared.txt", "remote commit\n", "remote-only-commit")
	runGitCommand(t, seed, "push")
	pullTestWrite(t, filepath.Join(local, "old name.txt"), "uncommitted\n")
	d := requirePullDiagnostic(t, sr.SyncWithOptions(SyncOptions{Force: true}), errors.ErrGitDiverged)
	if d.Ahead != 1 || d.Behind != 1 || !strings.Contains(d.LocalCommits, "local-only-commit") || !strings.Contains(d.RemoteCommits, "remote-only-commit") {
		t.Fatalf("missing divergent commits: %+v", d)
	}
	if d.StashRef == "" || !strings.Contains(runGitCommand(t, local, "stash", "show", "-p", d.StashRef), "uncommitted") {
		t.Fatalf("retained stash not identified: %+v", d)
	}
}

func TestPullDiagnosticPreviewLimitAndNoUpdate(t *testing.T) {
	sr, seed, local := newPullDiagnosticFixture(t)
	pullTestWrite(t, filepath.Join(local, "shared.txt"), "local preserved\n")
	if err := sr.SyncWithOptions(SyncOptions{Force: true}); err != nil || sr.StashRef() != "" {
		t.Fatalf("no-update behavior changed: %v, %s", err, sr.StashRef())
	}
	writeSystemGitCommit(t, seed, "shared.txt", "remote\n", "remote")
	runGitCommand(t, seed, "push")
	pullTestWrite(t, filepath.Join(local, "shared.txt"), strings.Repeat("local large content\n", 20000))
	d := requirePullDiagnostic(t, sr.Sync(), errors.ErrGitDirty)
	if len(d.Files) != 1 || !d.Files[0].Truncated || len(d.Files[0].UnstagedDiff) > pullPreviewLimit {
		t.Fatalf("diff output was not bounded: %+v", d.Files[0])
	}
}

func TestPullForceRetainsIdentifiedStash(t *testing.T) {
	sr, seed, local := newPullDiagnosticFixture(t)
	writeSystemGitCommit(t, seed, "shared.txt", "remote\n", "remote")
	runGitCommand(t, seed, "push")
	pullTestWrite(t, filepath.Join(local, "shared.txt"), "my-local-edit\n")
	if err := sr.SyncWithOptions(SyncOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	if sr.StashRef() == "" || !strings.Contains(runGitCommand(t, local, "stash", "show", "-p", sr.StashRef()), "my-local-edit") {
		t.Fatal("stash contents or identity missing")
	}
	data, _ := os.ReadFile(filepath.Join(local, "shared.txt"))
	if string(data) != "remote\n" {
		t.Fatalf("--force unexpectedly restored stash: %q", data)
	}
}

func TestPullDiagnosticDoesNotHideTransportFailure(t *testing.T) {
	sr, seed, local := newPullDiagnosticFixture(t)
	writeSystemGitCommit(t, local, "old name.txt", "local commit\n", "local")
	writeSystemGitCommit(t, seed, "shared.txt", "remote commit\n", "remote")
	runGitCommand(t, seed, "push")
	runGitCommand(t, local, "fetch")
	runGitCommand(t, local, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing.git"))
	err := sr.Sync()
	if err == nil || errors.IsCode(err, errors.ErrGitDiverged) || errors.Details(err)["pull"] != nil {
		t.Fatalf("stale divergent refs hid the transport failure: %v", err)
	}
}

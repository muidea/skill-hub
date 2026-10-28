package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/fs"
	"github.com/muidea/skill-hub/pkg/spec"
)

func registrationManager(t *testing.T) *StateManager {
	t.Helper()
	return &StateManager{statePath: filepath.Join(t.TempDir(), "hub", "state.json"), fs: &fs.RealFileSystem{}}
}

func TestProjectRegistrationOriginalReceiptSurvivesProjectChanges(t *testing.T) {
	m, project := registrationManager(t), t.TempDir()
	first, err := m.RegisterEmptyProject(project, "original-request")
	if err != nil || first.Status != spec.RegistrationSucceeded || !first.Result.ProjectCreated || first.Result.SkillCount != 0 {
		t.Fatalf("first registration = %#v, %v", first, err)
	}
	if len(first.Result.CreatedDirectories) != 2 || first.Executed == nil || !*first.Executed || !*first.Finished {
		t.Fatalf("missing original effects: %+v", first)
	}
	state, err := m.LoadProjectState(project)
	if err != nil {
		t.Fatal(err)
	}
	state.Skills["later-skill"] = spec.SkillVars{SkillID: "later-skill", Version: "2.0.0"}
	if err := m.SaveProjectState(state); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(m.statePath)
	for _, query := range []bool{false, true} {
		var got *spec.ProjectRegistrationReceipt
		if query {
			got, err = m.QueryProjectRegistration("original-request")
		} else {
			got, err = m.RegisterEmptyProject(project, "original-request")
		}
		if err != nil || !reflect.DeepEqual(first, got) {
			t.Fatalf("query=%t did not replay original receipt: got=%+v err=%v", query, got, err)
		}
	}
	after, _ := os.ReadFile(m.statePath)
	if string(before) != string(after) {
		t.Fatal("replay/query changed project state")
	}
	// A new request for the same project must preserve its existing skills.
	next, err := m.RegisterEmptyProject(project, "new-request")
	if err != nil || next.Result.ProjectCreated || next.Result.SkillCount != 1 || len(next.Result.CreatedDirectories) != 0 {
		t.Fatalf("existing-project request = %+v, %v", next, err)
	}
	if err := os.Remove(m.statePath); err != nil {
		t.Fatal(err)
	}
	got, err := m.QueryProjectRegistration("original-request")
	if err != nil || !reflect.DeepEqual(first, got) {
		t.Fatalf("query depends on live project state: %+v, %v", got, err)
	}
	if _, err := os.Stat(m.statePath); !os.IsNotExist(err) {
		t.Fatalf("query recreated state: %v", err)
	}
}

func TestProjectRegistrationRequestBinding(t *testing.T) {
	m, project := registrationManager(t), t.TempDir()
	if _, err := m.RegisterEmptyProject(project, "bound-request"); err != nil {
		t.Fatal(err)
	}
	other := t.TempDir()
	if _, err := m.RegisterEmptyProject(other, "bound-request"); !errors.IsCode(err, errors.ErrRequestConflict) {
		t.Fatalf("expected ID conflict, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(other, ".agents")); !os.IsNotExist(err) {
		t.Fatal("conflicting request created workspace")
	}
}

func TestProjectRegistrationQueryMissingDoesNotCreateFiles(t *testing.T) {
	m := registrationManager(t)
	for _, id := range []string{"missing", "../escape", "", "a/b"} {
		_, err := m.QueryProjectRegistration(id)
		want := errors.ErrInvalidInput
		if id == "missing" {
			want = errors.ErrRequestNotFound
		}
		if !errors.IsCode(err, want) {
			t.Fatalf("id %q error=%v, want %s", id, err, want)
		}
	}
	if _, err := os.Stat(filepath.Dir(m.statePath)); !os.IsNotExist(err) {
		t.Fatalf("query created hub files: %v", err)
	}
}

func TestProjectRegistrationFailureIsReplayed(t *testing.T) {
	m, project := registrationManager(t), t.TempDir()
	if err := os.WriteFile(filepath.Join(project, ".agents"), []byte("blocked"), 0644); err != nil {
		t.Fatal(err)
	}
	first, err := m.RegisterEmptyProject(project, "failed-request")
	if err != nil || first.Status != spec.RegistrationFailed || first.CommandError == nil || !*first.Finished {
		t.Fatalf("failed operation receipt=%+v, err=%v", first, err)
	}
	if err := os.Remove(filepath.Join(project, ".agents")); err != nil {
		t.Fatal(err)
	}
	got, err := m.RegisterEmptyProject(project, "failed-request")
	if err != nil || !reflect.DeepEqual(first, got) {
		t.Fatalf("failure was not replayed: %+v, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(project, ".agents")); !os.IsNotExist(err) {
		t.Fatal("retry executed a previously failed request")
	}
}

func TestProjectRegistrationRunningAndInterrupted(t *testing.T) {
	m, project := registrationManager(t), t.TempDir()
	path, _ := m.registrationPath("interrupted")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	lease, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if ok, err := tryLockRegistration(lease); err != nil || !ok {
		t.Fatalf("acquire lease: %v, %v", ok, err)
	}
	finished := false
	original := &spec.ProjectRegistrationReceipt{RequestID: "interrupted", ProjectPath: project, Status: spec.RegistrationRunning, Finished: &finished}
	if err := saveRegistration(path, original); err != nil {
		t.Fatal(err)
	}
	for _, retry := range []bool{false, true} {
		var got *spec.ProjectRegistrationReceipt
		if retry {
			got, err = m.RegisterEmptyProject(project, "interrupted")
		} else {
			got, err = m.QueryProjectRegistration("interrupted")
		}
		if err != nil || got.Status != spec.RegistrationRunning || got.Finished == nil || *got.Finished {
			t.Fatalf("active request = %+v, %v", got, err)
		}
	}
	_ = lease.Close() // Simulate an execution process dying before its final receipt.
	for _, retry := range []bool{false, true} {
		var got *spec.ProjectRegistrationReceipt
		if retry {
			got, err = m.RegisterEmptyProject(project, "interrupted")
		} else {
			got, err = m.QueryProjectRegistration("interrupted")
		}
		if err != nil || got.Status != spec.RegistrationUnknown || got.Executed != nil || got.Finished != nil || got.Result != nil {
			t.Fatalf("interrupted request must be unknown: %+v, %v", got, err)
		}
	}
	if _, err := os.Stat(filepath.Join(project, ".agents")); !os.IsNotExist(err) {
		t.Fatal("interrupted request was executed again")
	}
}

func TestProjectRegistrationConcurrentRequests(t *testing.T) {
	for _, sameID := range []bool{true, false} {
		t.Run(map[bool]string{true: "same ID", false: "different IDs"}[sameID], func(t *testing.T) {
			m, project := registrationManager(t), t.TempDir()
			var wg sync.WaitGroup
			for i := range 12 {
				wg.Go(func() {
					id := "concurrent"
					if !sameID {
						id += string(rune('a' + i))
					}
					if _, err := m.RegisterEmptyProject(project, id); err != nil && !errors.IsCode(err, errors.ErrRequestInProgress) {
						t.Errorf("concurrent registration: %v", err)
					}
				})
			}
			wg.Wait()
			projects, err := m.LoadAllProjectStates()
			if err != nil || len(projects) != 1 || len(projects[project].Skills) != 0 {
				t.Fatalf("concurrent requests damaged project state: %+v, %v", projects, err)
			}
			if sameID {
				got, err := m.QueryProjectRegistration("concurrent")
				if err != nil || got.Status != spec.RegistrationSucceeded || !got.Result.ProjectCreated {
					data, _ := json.Marshal(got)
					t.Fatalf("original receipt lost: %s, %v", data, err)
				}
			}
		})
	}
}

func TestProjectRegistrationCorruptReceiptDoesNotRepeat(t *testing.T) {
	m, project := registrationManager(t), t.TempDir()
	if _, err := m.RegisterEmptyProject(project, "damaged-receipt"); err != nil {
		t.Fatal(err)
	}
	path, _ := m.registrationPath("damaged-receipt")
	if err := os.WriteFile(path, []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(project, ".agents")); err != nil {
		t.Fatal(err)
	}
	if _, err := m.QueryProjectRegistration("damaged-receipt"); err == nil {
		t.Fatal("corrupt receipt was accepted")
	}
	if _, err := m.RegisterEmptyProject(project, "damaged-receipt"); err == nil {
		t.Fatal("corrupt receipt allowed execution")
	}
	// Well-formed JSON with an incomplete terminal result is also invalid.
	if err := saveRegistration(path, &spec.ProjectRegistrationReceipt{
		RequestID: "damaged-receipt", ProjectPath: project, Status: spec.RegistrationSucceeded,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.RegisterEmptyProject(project, "damaged-receipt"); err == nil {
		t.Fatal("incomplete terminal receipt allowed execution")
	}
	if _, err := os.Stat(filepath.Join(project, ".agents")); !os.IsNotExist(err) {
		t.Fatal("corrupt receipt allowed workspace recreation")
	}
}

func TestProjectRegistrationAfterInterruptedStateCommitDoesNotRepeat(t *testing.T) {
	m, project := registrationManager(t), t.TempDir()
	// Simulate the crash window after project commit but before the final receipt.
	if _, err := m.registerProject(project); err != nil {
		t.Fatal(err)
	}
	path, _ := m.registrationPath("committed-without-reply")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := saveRegistration(path, &spec.ProjectRegistrationReceipt{
		RequestID: "committed-without-reply", ProjectPath: project, Status: spec.RegistrationRunning,
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(project, ".agents")); err != nil {
		t.Fatal(err)
	}
	receipt, err := m.RegisterEmptyProject(project, "committed-without-reply")
	if err != nil || receipt.Status != spec.RegistrationUnknown || receipt.Result != nil {
		t.Fatalf("interrupted commit must not be inferred from live status: %+v, %v", receipt, err)
	}
	if _, err := os.Stat(filepath.Join(project, ".agents")); !os.IsNotExist(err) {
		t.Fatal("interrupted operation was repeated")
	}
}

func TestProjectRegistrationExplicitNestedPath(t *testing.T) {
	m, parent := registrationManager(t), t.TempDir()
	if _, err := m.RegisterEmptyProject(parent, "parent-request"); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(parent, "nested")
	if err := os.Mkdir(child, 0755); err != nil {
		t.Fatal(err)
	}
	receipt, err := m.RegisterEmptyProject(child, "child-request")
	if err != nil || receipt.ProjectPath != child || !receipt.Result.ProjectCreated {
		t.Fatalf("nested registration inherited parent: %+v, %v", receipt, err)
	}
	projects, err := m.LoadAllProjectStates()
	if err != nil || len(projects) != 2 {
		t.Fatalf("expected distinct registered projects: %+v, %v", projects, err)
	}
}

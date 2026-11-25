package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/muidea/skill-hub/pkg/errors"
)

func TestPushRuntimeErrorDoesNotPrintUsage(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", home)
	t.Setenv("SKILL_HUB_DISABLE_SERVICE_BRIDGE", "1")

	var stdout, stderr bytes.Buffer
	oldOut := rootCmd.OutOrStdout()
	oldErr := rootCmd.ErrOrStderr()
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	defer func() {
		rootCmd.SetOut(oldOut)
		rootCmd.SetErr(oldErr)
		pushForce = false
		pushDryRun = false
		pushJSON = false
		pushMessage = ""
	}()

	err := executeCommand(rootCmd, []string{"push", "--force"})
	if err == nil {
		t.Fatal("expected error for uninitialized SKILL_HUB_HOME")
	}
	if strings.Contains(stdout.String(), "Usage:") || strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("runtime push error should not print Cobra usage\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
}

func TestPushInvalidFlagReturnsInvalidInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	oldOut := rootCmd.OutOrStdout()
	oldErr := rootCmd.ErrOrStderr()
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	defer func() {
		rootCmd.SetOut(oldOut)
		rootCmd.SetErr(oldErr)
		pushForce = false
		pushDryRun = false
		pushJSON = false
		pushMessage = ""
	}()

	err := executeCommand(rootCmd, []string{"push", "--no-such-flag"})
	if !errors.IsCode(err, errors.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

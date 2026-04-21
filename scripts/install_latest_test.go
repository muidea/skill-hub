package scripts_test

import (
	"os"
	"strings"
	"testing"
)

func TestInstallLatestHandlesRunningServeBinary(t *testing.T) {
	contentBytes, err := os.ReadFile("install-latest.sh")
	if err != nil {
		t.Fatalf("read install script: %v", err)
	}

	content := string(contentBytes)
	installBlock := sectionBetween(t, content, "自动安装到 ~/.local/bin/...", "# 验证安装和使用说明")
	verifyBlock := sectionBetween(t, content, "# 验证安装和使用说明", "# 清理提示和总结")

	if strings.Contains(installBlock, "cp \"$ACTUAL_BINARY\" ~/.local/bin/") {
		t.Fatalf("install should not copy directly over the active binary")
	}
	if !strings.Contains(installBlock, "mv -f \"$install_tmp\" \"$install_path\"") {
		t.Fatalf("install should atomically replace the target binary with mv -f")
	}
	if !strings.Contains(installBlock, "exit 1") {
		t.Fatalf("install failure should stop instead of falling through to verification")
	}
	if !strings.Contains(installBlock, "serve") {
		t.Fatalf("install failure guidance should mention running serve processes")
	}

	if strings.Contains(verifyBlock, "command -v \"$ACTUAL_BINARY\"") {
		t.Fatalf("verification should not use PATH lookup because it can find an old binary")
	}
	if strings.Contains(verifyBlock, "\"$ACTUAL_BINARY\" --version") {
		t.Fatalf("verification should execute the installed target path, not the extracted filename")
	}
	if !strings.Contains(verifyBlock, "\"$install_path\" --version") {
		t.Fatalf("verification should execute the installed target path")
	}
	if !strings.Contains(verifyBlock, "installed_version") || !strings.Contains(verifyBlock, "expected_version") {
		t.Fatalf("verification should compare installed and expected versions")
	}
}

func TestInstallLatestRestartsRegisteredServeInstances(t *testing.T) {
	contentBytes, err := os.ReadFile("install-latest.sh")
	if err != nil {
		t.Fatalf("read install script: %v", err)
	}

	content := string(contentBytes)
	if !strings.Contains(content, "update_registered_serve_instances()") {
		t.Fatalf("installer should define registered serve updater")
	}

	updateBlock := sectionBetween(t, content, "update_registered_serve_instances()", "# 主函数")
	for _, expected := range []string{
		"serve status",
		"awk -F '\\t' '$2==\"running\"{print $1}'",
		"serve stop \"$service_name\"",
		"serve start \"$service_name\"",
		"UPDATED_SERVE_COUNT",
	} {
		if !strings.Contains(updateBlock, expected) {
			t.Fatalf("serve updater should contain %q", expected)
		}
	}

	postVerifyBlock := sectionBetween(t, content, "✅ 安装验证成功！", "# 清理提示和总结")
	if !strings.Contains(postVerifyBlock, "update_registered_serve_instances \"$install_path\"") {
		t.Fatalf("installer should update registered serve instances after verifying the installed binary")
	}
	if !strings.Contains(content, "serve更新") {
		t.Fatalf("install summary should report serve update result")
	}
}

func sectionBetween(t *testing.T, content, start, end string) string {
	t.Helper()

	startIndex := strings.Index(content, start)
	if startIndex < 0 {
		t.Fatalf("missing section start %q", start)
	}

	endIndex := strings.Index(content[startIndex:], end)
	if endIndex < 0 {
		t.Fatalf("missing section end %q", end)
	}

	return content[startIndex : startIndex+endIndex]
}

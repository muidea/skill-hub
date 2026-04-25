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

func TestInstallLatestInstallsBundledAgentSkills(t *testing.T) {
	contentBytes, err := os.ReadFile("install-latest.sh")
	if err != nil {
		t.Fatalf("read install script: %v", err)
	}

	content := string(contentBytes)
	if !strings.Contains(content, "install_agent_skills()") {
		t.Fatalf("installer should define bundled agent skills installer")
	}

	installBlock := sectionBetween(t, content, "install_agent_skills()", "# 主函数")
	for _, expected := range []string{
		"SKILL_HUB_INSTALL_AGENT_SKILLS",
		"SKILL_HUB_INSTALL_CODEX_SKILLS",
		"SKILL_HUB_INSTALL_OPENCODE_SKILLS",
		"SKILL_HUB_INSTALL_CLAUDE_SKILLS",
		"SKILL_HUB_AGENT_SKILLS_DIR",
		"CODEX_SKILLS_DIR",
		"OPENCODE_SKILLS_DIR",
		"CLAUDE_SKILLS_DIR",
		`${XDG_DATA_HOME:-$HOME/.local/share}/skill-hub/agent-skills`,
		`${CODEX_HOME:-$HOME/.codex}`,
		`$codex_home/skills`,
		`${OPENCODE_HOME:-$HOME/.config/opencode}`,
		`$opencode_home/skills`,
		`$HOME/.claude/skills`,
		"skill-hub-*",
		"SKILL.md",
	} {
		if !strings.Contains(installBlock, expected) {
			t.Fatalf("agent skills installer should contain %q", expected)
		}
	}

	postInstallBlock := sectionBetween(t, content, "安装 Shell 补全", "# 验证安装和使用说明")
	if !strings.Contains(postInstallBlock, `install_agent_skills "agent-skills"`) {
		t.Fatalf("installer should install bundled agent skills after installing the binary")
	}
	if !strings.Contains(installBlock, "command -v codex") || !strings.Contains(installBlock, "command -v opencode") {
		t.Fatalf("installer should detect installed agents before mirroring skills")
	}
	if !strings.Contains(content, "Agent Skills") {
		t.Fatalf("install summary should report agent skills installation")
	}
}

func TestInstallLatestVerifiesReleaseArchiveChecksum(t *testing.T) {
	contentBytes, err := os.ReadFile("install-latest.sh")
	if err != nil {
		t.Fatalf("read install script: %v", err)
	}

	content := string(contentBytes)
	downloadBlock := sectionBetween(t, content, "# 下载校验文件", "# 解压文件")
	if !strings.Contains(downloadBlock, "verify_file \"$ARCHIVE_NAME\" \"$CHECKSUM_NAME\"") {
		t.Fatalf("installer should verify the release archive checksum before extraction")
	}

	binaryBlock := sectionBetween(t, content, "# 查找解压出的二进制文件", "# 显示内容")
	if strings.Contains(binaryBlock, "verify_file \"$ACTUAL_BINARY\" \"$CHECKSUM_NAME\"") {
		t.Fatalf("installer should not verify extracted binary with archive checksum")
	}
}

func TestReleasePackagesIncludeAgentSkills(t *testing.T) {
	contentBytes, err := os.ReadFile("../Makefile")
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}

	content := string(contentBytes)
	if got := strings.Count(content, "cp -R agent-skills"); got != 6 {
		t.Fatalf("release packages should copy agent-skills for all six platforms, got %d", got)
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

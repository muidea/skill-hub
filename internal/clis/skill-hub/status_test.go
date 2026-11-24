package cli

import (
	"strings"
	"testing"

	globalservice "github.com/muidea/skill-hub/internal/modules/application/global/service"
	projectstatusservice "github.com/muidea/skill-hub/internal/modules/application/project_status/service"
	"github.com/muidea/skill-hub/pkg/spec"
)

func TestDetermineSkillStatus(t *testing.T) {
	tests := []struct {
		name       string
		localVer   string
		localHash  string
		repoVer    string
		repoHash   string
		wantStatus string
	}{
		{
			name:       "outdated_when_repo_version_higher_and_content_differs",
			localVer:   "1.0.0",
			localHash:  "hash-local",
			repoVer:    "1.1.0",
			repoHash:   "hash-repo",
			wantStatus: spec.SkillStatusOutdated,
		},
		{
			name:       "modified_when_local_version_not_lower_and_content_differs",
			localVer:   "1.1.0",
			localHash:  "hash-local",
			repoVer:    "1.0.0",
			repoHash:   "hash-repo",
			wantStatus: spec.SkillStatusModified,
		},
		{
			name:       "outdated_when_hash_equal_but_local_version_lower",
			localVer:   "1.0.0",
			localHash:  "same-hash",
			repoVer:    "1.1.0",
			repoHash:   "same-hash",
			wantStatus: spec.SkillStatusOutdated,
		},
		{
			name:       "synced_when_hash_and_version_equal",
			localVer:   "1.0.0",
			localHash:  "same-hash",
			repoVer:    "1.0.0",
			repoHash:   "same-hash",
			wantStatus: spec.SkillStatusSynced,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineSkillStatus(tt.localVer, tt.localHash, tt.repoVer, tt.repoHash)
			if got != tt.wantStatus {
				t.Fatalf("determineSkillStatus() = %s, want %s", got, tt.wantStatus)
			}
		})
	}
}

func TestDescribeChangeDirection(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		wantSubstr string
	}{
		{
			name:       "modified_direction_message",
			status:     spec.SkillStatusModified,
			wantSubstr: "本地在其基础上发生了修改",
		},
		{
			name:       "outdated_direction_message",
			status:     spec.SkillStatusOutdated,
			wantSubstr: "仓库中的技能内容比本地版本更新",
		},
		{
			name:       "synced_direction_message",
			status:     spec.SkillStatusSynced,
			wantSubstr: "本地与仓库版本一致",
		},
		{
			name:       "empty_for_unknown_status",
			status:     "UNKNOWN",
			wantSubstr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := describeChangeDirection(tt.status, "1.0.0", "1.0.0")
			if tt.wantSubstr == "" {
				if got != "" {
					t.Fatalf("describeChangeDirection() = %q, want empty string", got)
				}
				return
			}
			if !strings.Contains(got, tt.wantSubstr) {
				t.Fatalf("describeChangeDirection() = %q, want substring %q", got, tt.wantSubstr)
			}
		})
	}
}

func TestFilterStatusSummaryByIDs(t *testing.T) {
	summary := &projectstatusservice.ProjectStatusSummary{
		ProjectPath: "/proj",
		SkillCount:  3,
		Items: []projectstatusservice.SkillStatusItem{
			{SkillID: "magic-helper", Status: spec.SkillStatusSynced},
			{SkillID: "magic-pack", Status: spec.SkillStatusSynced},
			{SkillID: "git-expert", Status: spec.SkillStatusModified},
		},
	}
	ids := map[string]struct{}{
		"magic-helper": {},
		"git-expert":   {},
		"absent":       {},
	}

	got := filterStatusSummaryByIDs(summary, ids)
	if got == nil {
		t.Fatalf("expected non-nil summary")
	}
	if got.SkillCount != 2 {
		t.Errorf("SkillCount = %d, want 2", got.SkillCount)
	}
	if len(got.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(got.Items))
	}
	gotIDs := map[string]bool{}
	for _, it := range got.Items {
		gotIDs[it.SkillID] = true
	}
	if !gotIDs["magic-helper"] || !gotIDs["git-expert"] {
		t.Errorf("filtered items mismatch: %v", gotIDs)
	}
	if gotIDs["magic-pack"] {
		t.Errorf("magic-pack should be filtered out")
	}

	if got := filterStatusSummaryByIDs(nil, ids); got != nil {
		t.Errorf("nil summary should return nil")
	}
}

func TestCompilePatterns(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		wantErr  bool
	}{
		{name: "empty", patterns: nil, wantErr: false},
		{name: "valid_literal", patterns: []string{"magic-skill"}, wantErr: false},
		{name: "valid_wildcard", patterns: []string{"magic*"}, wantErr: false},
		{name: "valid_double_star", patterns: []string{"**"}, wantErr: false},
		{name: "lone_star_rejected", patterns: []string{"*"}, wantErr: true},
		{name: "malformed_bracket", patterns: []string{"[abc"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := compilePatterns(tt.patterns)
			if (err != nil) != tt.wantErr {
				t.Errorf("compilePatterns(%v) err = %v, wantErr %v", tt.patterns, err, tt.wantErr)
			}
		})
	}
}

func TestFilterGlobalStatusSummaryByIDs(t *testing.T) {
	summary := &globalservice.StatusSummary{
		Scope:      "global",
		GlobalPath: "/home/.skill-hub/global/skills",
		SkillCount: 3,
		Items: []globalservice.StatusItem{
			{SkillID: "magic-helper", Agent: "codex", Status: globalservice.StatusOK},
			{SkillID: "magic-pack", Agent: "codex", Status: globalservice.StatusNotApplied},
			{SkillID: "git-expert", Agent: "codex", Status: globalservice.StatusStale},
		},
	}
	ids := map[string]struct{}{
		"magic-helper": {},
		"git-expert":   {},
		"absent":       {},
	}

	got := filterGlobalStatusSummaryByIDs(summary, ids)
	if got == nil {
		t.Fatalf("expected non-nil summary")
	}
	if got.Scope != "global" || got.GlobalPath != summary.GlobalPath {
		t.Errorf("summary metadata not preserved: scope=%q path=%q", got.Scope, got.GlobalPath)
	}
	if got.SkillCount != 2 {
		t.Errorf("SkillCount = %d, want 2", got.SkillCount)
	}
	if len(got.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(got.Items))
	}
	gotIDs := map[string]bool{}
	for _, it := range got.Items {
		gotIDs[it.SkillID] = true
	}
	if !gotIDs["magic-helper"] || !gotIDs["git-expert"] {
		t.Errorf("filtered items mismatch: %v", gotIDs)
	}
	if gotIDs["magic-pack"] {
		t.Errorf("magic-pack should be filtered out")
	}

	if got := filterGlobalStatusSummaryByIDs(nil, ids); got != nil {
		t.Errorf("nil summary should return nil")
	}
}

func TestFilterGlobalStatusSummaryByIDsCountsDistinctSkills(t *testing.T) {
	summary := &globalservice.StatusSummary{
		Items: []globalservice.StatusItem{
			{SkillID: "magic-helper", Agent: "claude", Status: globalservice.StatusOK},
			{SkillID: "magic-helper", Agent: "codex", Status: globalservice.StatusOK},
		},
	}

	got := filterGlobalStatusSummaryByIDs(summary, map[string]struct{}{"magic-helper": {}})
	if got.SkillCount != 1 {
		t.Errorf("SkillCount = %d, want 1", got.SkillCount)
	}
}

func TestRenderGlobalStatusSummaryShowsVersion(t *testing.T) {
	summary := &globalservice.StatusSummary{
		Scope:      "global",
		GlobalPath: "/home/.skill-hub/global/skills",
		SkillCount: 1,
		Items: []globalservice.StatusItem{
			{SkillID: "demo-skill", Version: "1.2.3", Agent: "codex", Status: globalservice.StatusOK},
		},
	}

	output := captureStdout(t, func() {
		renderGlobalStatusSummary(summary)
	})
	if !strings.Contains(output, "版本") || !strings.Contains(output, "1.2.3") {
		t.Fatalf("global status output must include version, got %q", output)
	}
}

func TestRenderGlobalStatusSummaryAggregatesAgentsBySkill(t *testing.T) {
	summary := &globalservice.StatusSummary{
		GlobalPath: "/home/.skill-hub/global/skills",
		SkillCount: 1,
		Items: []globalservice.StatusItem{
			{SkillID: "demo-skill", Version: "1.2.3", Agent: "opencode", Status: globalservice.StatusOK},
			{SkillID: "demo-skill", Version: "1.2.3", Agent: "codex", Status: globalservice.StatusOK},
			{SkillID: "demo-skill", Version: "1.2.3", Agent: "claude", Status: globalservice.StatusOK},
			{SkillID: "demo-skill", Version: "1.2.3", Agent: "agy", Status: globalservice.StatusOK},
		},
	}

	output := captureStdout(t, func() {
		renderGlobalStatusSummary(summary)
	})
	if got := strings.Count(output, "demo-skill"); got != 1 {
		t.Fatalf("demo-skill appeared %d times, want one aggregated row: %q", got, output)
	}
	if !strings.Contains(output, "agy,claude,codex,opencode") {
		t.Fatalf("output must include the aggregated agent list, got %q", output)
	}
}

func TestRenderGlobalStatusSummaryShowsAgentDetailsForMixedStatus(t *testing.T) {
	summary := &globalservice.StatusSummary{
		GlobalPath: "/home/.skill-hub/global/skills",
		Items: []globalservice.StatusItem{
			{SkillID: "demo-skill", Version: "1.2.3", Agent: "codex", Status: globalservice.StatusOK},
			{SkillID: "demo-skill", Version: "1.2.3", Agent: "claude", Status: globalservice.StatusStale, Message: "来源已更新"},
		},
	}

	output := captureStdout(t, func() {
		renderGlobalStatusSummary(summary)
	})
	for _, want := range []string{"mixed", "claude: 🔄 outdated: 来源已更新", "codex: ✅ synced"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output must include %q, got %q", want, output)
		}
	}
}

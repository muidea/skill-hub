package cli

import (
	"strings"
	"testing"

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

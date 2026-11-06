package spec

// PullDiagnostic describes the repository snapshot that blocked a pull.
type PullDiagnostic struct {
	Kind             string           `json:"kind"`
	RepositoryPath   string           `json:"repository_path"`
	LocalCommit      string           `json:"local_commit,omitempty"`
	RemoteCommit     string           `json:"remote_commit,omitempty"`
	Ahead            int              `json:"ahead"`
	Behind           int              `json:"behind"`
	OverlapKnown     bool             `json:"overlap_known"`
	Files            []PullFileChange `json:"files"`
	LocalCommits     string           `json:"local_commits,omitempty"`
	RemoteCommits    string           `json:"remote_commits,omitempty"`
	StashRef         string           `json:"stash_ref,omitempty"`
	OmittedFiles     int              `json:"omitted_files,omitempty"`
	InspectionErrors []string         `json:"inspection_errors,omitempty"`
}

// PullFileChange separates index/worktree changes from potential remote overlap.
type PullFileChange struct {
	Path           string `json:"path"`
	OriginalPath   string `json:"original_path,omitempty"`
	IndexStatus    string `json:"index_status"`
	WorktreeStatus string `json:"worktree_status"`
	ConflictType   string `json:"conflict_type,omitempty"`
	OverlapsRemote bool   `json:"overlaps_remote"`
	StagedDiff     string `json:"staged_diff,omitempty"`
	UnstagedDiff   string `json:"unstaged_diff,omitempty"`
	RemoteDiff     string `json:"remote_diff,omitempty"`
	ConflictDiff   string `json:"conflict_diff,omitempty"`
	Binary         bool   `json:"binary"`
	Truncated      bool   `json:"truncated"`
}

// PullResult supplements the existing skill count with any retained stash.
type PullResult struct {
	SkillCount int    `json:"skill_count"`
	StashRef   string `json:"stash_ref,omitempty"`
}

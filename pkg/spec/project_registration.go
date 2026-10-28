package spec

import "github.com/muidea/skill-hub/pkg/errors"

// ProjectRegistrationReceipt records one CLI request independently of current
// project state. Nil Executed/Finished mean the outcome cannot be established.
type ProjectRegistrationReceipt struct {
	RequestID    string                     `json:"request_id"`
	ProjectPath  string                     `json:"project_path,omitempty"`
	Status       string                     `json:"status"`
	Executed     *bool                      `json:"executed"`
	Finished     *bool                      `json:"finished"`
	StartedAt    string                     `json:"started_at,omitempty"`
	CompletedAt  string                     `json:"completed_at,omitempty"`
	Result       *ProjectRegistrationResult `json:"result,omitempty"`
	CommandError *ProjectRegistrationError  `json:"command_error,omitempty"`
}

// ProjectRegistrationResult describes effects observed during the original
// request, rather than a fresh inspection of the project.
type ProjectRegistrationResult struct {
	Registered         bool     `json:"registered"`
	ProjectCreated     bool     `json:"project_created"`
	CreatedDirectories []string `json:"created_directories"`
	SkillCount         int      `json:"skill_count"`
}

// ProjectRegistrationError preserves the original command failure in a receipt.
type ProjectRegistrationError struct {
	Code    errors.ErrorCode `json:"code"`
	Message string           `json:"message"`
}

const (
	RegistrationRunning   = "running"
	RegistrationSucceeded = "succeeded"
	RegistrationFailed    = "failed"
	RegistrationUnknown   = "unknown"
)

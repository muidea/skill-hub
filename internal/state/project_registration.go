package state

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
)

var registrationRequestID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

func (m *StateManager) registrationPath(requestID string) (string, error) {
	if !registrationRequestID.MatchString(requestID) {
		return "", errors.NewWithCode("project.register", errors.ErrInvalidInput, "request-id 须为 1–128 个字母、数字、点、下划线、冒号或连字符，且以字母或数字开头")
	}
	return filepath.Join(filepath.Dir(m.statePath), "project-registrations", fmt.Sprintf("%x.json", sha256.Sum256([]byte(requestID)))), nil
}

// RegisterEmptyProject claims the request before making changes. A completed
// request always replays its stored receipt, even if the project later changes.
// An interrupted request is never executed again under the same ID.
func (m *StateManager) RegisterEmptyProject(projectPath, requestID string) (*spec.ProjectRegistrationReceipt, error) {
	path, err := m.registrationPath(requestID)
	if err != nil {
		return nil, err
	}
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	lease, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	defer lease.Close()
	locked, err := tryLockRegistration(lease)
	if err != nil {
		return nil, err
	}
	if !locked {
		receipt, err := m.QueryProjectRegistration(requestID)
		if errors.IsCode(err, errors.ErrRequestNotFound) {
			return nil, errors.NewWithCode("project.register", errors.ErrRequestInProgress, "原请求正在建立回执，请稍后按原 request-id 查询")
		}
		return registrationForPath(receipt, absPath, err)
	}
	if receipt, err := readRegistration(path, requestID); !os.IsNotExist(err) {
		if err == nil && receipt.Status == spec.RegistrationRunning {
			markRegistrationUnknown(receipt)
		}
		return registrationForPath(receipt, absPath, err)
	}

	unfinished := false
	receipt := &spec.ProjectRegistrationReceipt{
		RequestID: requestID, ProjectPath: absPath, Status: spec.RegistrationRunning,
		Finished: &unfinished, StartedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := saveRegistration(path, receipt); err != nil {
		return nil, err // No project changes are allowed before this durable claim.
	}
	result, operationErr := m.registerProject(absPath)
	finished := true
	receipt.Executed, receipt.Finished = &finished, &finished
	receipt.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	receipt.Result = result
	receipt.Status = spec.RegistrationSucceeded
	if operationErr != nil {
		receipt.Status = spec.RegistrationFailed
		receipt.CommandError = &spec.ProjectRegistrationError{Code: errors.Code(operationErr), Message: errors.Message(operationErr)}
	}
	if err := saveRegistration(path, receipt); err != nil {
		// The claim remains durable, so a retry cannot repeat these effects.
		// Do not report a terminal result that was not persisted.
		markRegistrationUnknown(receipt)
		return receipt, errors.WrapWithCode(err, "project.register", errors.ErrRequestOutcomeUnknown, "无法保存最终回执；请按原 request-id 查询，勿更换 ID 重试")
	}
	return receipt, nil
}

func registrationForPath(receipt *spec.ProjectRegistrationReceipt, projectPath string, err error) (*spec.ProjectRegistrationReceipt, error) {
	if err != nil {
		return nil, err
	}
	if receipt.ProjectPath != projectPath {
		return nil, errors.NewWithCode("project.register", errors.ErrRequestConflict, "request-id 已绑定其他项目路径，未执行新的登记")
	}
	return receipt, nil
}

// QueryProjectRegistration only reads the original receipt and its execution
// lease. It neither loads project state nor creates files or retries work.
func (m *StateManager) QueryProjectRegistration(requestID string) (*spec.ProjectRegistrationReceipt, error) {
	path, err := m.registrationPath(requestID)
	if err != nil {
		return nil, err
	}
	receipt, err := readRegistration(path, requestID)
	if os.IsNotExist(err) {
		return nil, errors.NewWithCode("project.register.query", errors.ErrRequestNotFound, "未找到此 request-id 的持久化回执；这不证明请求从未执行")
	}
	if err != nil || receipt.Status != spec.RegistrationRunning {
		return receipt, err
	}
	lease, err := os.OpenFile(path+".lock", os.O_RDWR, 0)
	if os.IsNotExist(err) {
		markRegistrationUnknown(receipt)
		return receipt, nil
	}
	if err != nil {
		return nil, err
	}
	defer lease.Close()
	locked, err := tryLockRegistration(lease)
	if err != nil {
		return nil, err
	}
	if locked {
		// Re-read under the lease: execution may have finished after our first read.
		receipt, err = readRegistration(path, requestID)
		if err == nil && receipt.Status == spec.RegistrationRunning {
			markRegistrationUnknown(receipt)
		}
	}
	return receipt, err
}

func markRegistrationUnknown(receipt *spec.ProjectRegistrationReceipt) {
	receipt.Status = spec.RegistrationUnknown
	receipt.Executed, receipt.Finished = nil, nil
	receipt.CompletedAt, receipt.Result = "", nil
	receipt.CommandError = &spec.ProjectRegistrationError{
		Code: errors.ErrRequestOutcomeUnknown, Message: "原执行进程已中断或最终回执未保存，无法确定原请求的完整效果；相同 request-id 不会重新执行",
	}
}

func readRegistration(path, requestID string) (*spec.ProjectRegistrationReceipt, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var receipt spec.ProjectRegistrationReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return nil, fmt.Errorf("读取登记回执失败（不会重新执行请求）: %w", err)
	}
	if receipt.RequestID != requestID || receipt.ProjectPath == "" ||
		(receipt.Status != spec.RegistrationRunning && receipt.Status != spec.RegistrationSucceeded && receipt.Status != spec.RegistrationFailed) {
		return nil, fmt.Errorf("登记回执内容无效（不会重新执行请求）")
	}
	if receipt.Status != spec.RegistrationRunning &&
		(receipt.Executed == nil || !*receipt.Executed || receipt.Finished == nil || !*receipt.Finished ||
			receipt.Result == nil || receipt.CompletedAt == "" ||
			(receipt.Status == spec.RegistrationFailed && receipt.CommandError == nil) ||
			(receipt.Status == spec.RegistrationSucceeded && receipt.CommandError != nil)) {
		return nil, fmt.Errorf("登记终态回执不完整（不会重新执行请求）")
	}
	return &receipt, nil
}

func saveRegistration(path string, receipt *spec.ProjectRegistrationReceipt) error {
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomically(path, data, 0600)
}

func (m *StateManager) registerProject(projectPath string) (*spec.ProjectRegistrationResult, error) {
	result := &spec.ProjectRegistrationResult{CreatedDirectories: []string{}}
	err := m.withWriteLock(func() error {
		info, err := os.Stat(projectPath)
		if err != nil || !info.IsDir() {
			return errors.NewWithCode("project.register", errors.ErrProjectInvalid, "项目路径必须是已有目录")
		}
		projects, err := m.LoadAllProjectStates()
		if err != nil {
			return err
		}
		project, exists := projects[projectPath] // Explicit registration never inherits a parent project.
		result.Registered, result.SkillCount = exists, len(project.Skills)
		for _, dir := range []string{filepath.Join(projectPath, ".agents"), filepath.Join(projectPath, ".agents", "skills")} {
			if err := os.Mkdir(dir, 0755); err != nil {
				if !os.IsExist(err) {
					return err
				}
				info, err := os.Stat(dir)
				if err != nil || !info.IsDir() {
					return errors.NewWithCode("project.register", errors.ErrFileOperation, "工作区路径不是目录: "+dir)
				}
			} else {
				result.CreatedDirectories = append(result.CreatedDirectories, dir)
			}
		}
		if exists {
			return nil // Do not reset skills, timestamps or legacy fields.
		}
		projects[projectPath] = spec.ProjectState{ProjectPath: projectPath, Skills: map[string]spec.SkillVars{}}
		if err := m.saveAllProjectStates(projects); err != nil {
			return err
		}
		result.Registered, result.ProjectCreated = true, true
		return nil
	})
	return result, err
}

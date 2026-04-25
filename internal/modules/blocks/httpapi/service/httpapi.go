package service

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/muidea/skill-hub/internal/config"
	gitpkg "github.com/muidea/skill-hub/internal/git"
	httpapibiz "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/biz"
	globalservice "github.com/muidea/skill-hub/internal/modules/kernel/global/service"
	projectapplymodule "github.com/muidea/skill-hub/internal/modules/kernel/project_apply"
	projectfeedbackmodule "github.com/muidea/skill-hub/internal/modules/kernel/project_feedback"
	projectinventorymodule "github.com/muidea/skill-hub/internal/modules/kernel/project_inventory"
	projectstatusmodule "github.com/muidea/skill-hub/internal/modules/kernel/project_status"
	projectusemodule "github.com/muidea/skill-hub/internal/modules/kernel/project_use"
	runtimemodule "github.com/muidea/skill-hub/internal/modules/kernel/runtime"
	apperrors "github.com/muidea/skill-hub/pkg/errors"
)

type HTTPAPI struct {
	runtimeSvc   *runtimemodule.Runtime
	inventorySvc *projectinventorymodule.ProjectInventory
	statusSvc    *projectstatusmodule.ProjectStatus
	useSvc       *projectusemodule.ProjectUse
	applySvc     *projectapplymodule.ProjectApply
	feedbackSvc  *projectfeedbackmodule.ProjectFeedback
	globalSvc    *globalservice.Global
}

func New() *HTTPAPI {
	return &HTTPAPI{
		runtimeSvc:   runtimemodule.New(),
		inventorySvc: projectinventorymodule.New(),
		statusSvc:    projectstatusmodule.New(),
		useSvc:       projectusemodule.New(),
		applySvc:     projectapplymodule.New(),
		feedbackSvc:  projectfeedbackmodule.New(),
		globalSvc:    globalservice.New(),
	}
}

func (h *HTTPAPI) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", h.handleHealth)
	mux.HandleFunc("/api/v1/repos", h.handleRepos)
	mux.HandleFunc("/api/v1/repos/", h.handleRepoActions)
	mux.HandleFunc("/api/v1/skill-repository/status", h.handleSkillRepositoryStatus)
	mux.HandleFunc("/api/v1/skill-repository/sync-check", h.handleCheckSkillRepository)
	mux.HandleFunc("/api/v1/skill-repository/sync", h.handleSyncSkillRepository)
	mux.HandleFunc("/api/v1/skill-repository/push-preview", h.handlePushSkillRepositoryPreview)
	mux.HandleFunc("/api/v1/skill-repository/push", h.handlePushSkillRepository)
	mux.HandleFunc("/api/v1/search", h.handleSearch)
	mux.HandleFunc("/api/v1/skills", h.handleSkills)
	mux.HandleFunc("/api/v1/skills/", h.handleSkillActions)
	mux.HandleFunc("/api/v1/projects", h.handleProjects)
	mux.HandleFunc("/api/v1/projects/", h.handleProjectActions)
	mux.HandleFunc("/api/v1/project-status", h.handleProjectStatus)
	mux.HandleFunc("/api/v1/project-skills/use", h.handleUseSkill)
	mux.HandleFunc("/api/v1/project-skills/register", h.handleRegisterSkill)
	mux.HandleFunc("/api/v1/project-skills/import", h.handleImportSkills)
	mux.HandleFunc("/api/v1/project-skills/dedupe", h.handleDedupeSkills)
	mux.HandleFunc("/api/v1/project-skills/sync-copies", h.handleSyncCopies)
	mux.HandleFunc("/api/v1/project-skills/lint-paths", h.handleLintPaths)
	mux.HandleFunc("/api/v1/project-skills/validate", h.handleValidateProjectSkills)
	mux.HandleFunc("/api/v1/project-skills/audit", h.handleAuditProjectSkills)
	mux.HandleFunc("/api/v1/project-apply", h.handleApplyProject)
	mux.HandleFunc("/api/v1/project-feedback/preview", h.handlePreviewFeedback)
	mux.HandleFunc("/api/v1/project-feedback/apply", h.handleApplyFeedback)
	mux.HandleFunc("/api/v1/global-status", h.handleGlobalStatus)
	mux.HandleFunc("/api/v1/global-skills/use", h.handleUseGlobalSkill)
	mux.HandleFunc("/api/v1/global-skills/", h.handleGlobalSkillActions)
	mux.HandleFunc("/api/v1/global-apply", h.handleApplyGlobal)
	return mux
}

func (h *HTTPAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.HealthData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.HealthData{Status: "ok"},
	})
}

func (h *HTTPAPI) handleRepos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		repos, err := h.runtimeSvc.Service().ListRepositories(true)
		if err != nil {
			writeWrappedError(w, err)
			return
		}
		defaultRepo, _ := h.runtimeSvc.Service().DefaultRepository()
		defaultRepoName := ""
		if defaultRepo != nil {
			defaultRepoName = defaultRepo.Name
		}
		writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.RepoListData]{
			Code: httpapibiz.CodeOK,
			Data: httpapibiz.RepoListData{
				DefaultRepo: defaultRepoName,
				Items:       repos,
			},
		})
	case http.MethodPost:
		var req httpapibiz.AddRepoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_JSON", "请求体格式无效")
			return
		}
		repoCfg := config.RepositoryConfig{
			Name:        req.Name,
			URL:         req.URL,
			Branch:      req.Branch,
			Type:        req.Type,
			Description: req.Description,
			Enabled:     true,
		}
		if err := h.runtimeSvc.Service().AddRepository(repoCfg); err != nil {
			writeWrappedError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, httpapibiz.Response[map[string]string]{
			Code: httpapibiz.CodeOK,
			Data: map[string]string{"status": "ok"},
		})
	default:
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
	}
}

func (h *HTTPAPI) handleRepoActions(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/repos/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "接口不存在")
		return
	}

	parts := strings.Split(path, "/")
	name := parts[0]

	if len(parts) == 1 && r.Method == http.MethodDelete {
		if err := h.runtimeSvc.Service().RemoveRepository(name); err != nil {
			writeWrappedError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, httpapibiz.Response[map[string]string]{Code: httpapibiz.CodeOK, Data: map[string]string{"status": "ok"}})
		return
	}

	if len(parts) != 2 || r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	var err error
	switch parts[1] {
	case "sync":
		err = h.runtimeSvc.Service().SyncRepository(name)
	case "enable":
		err = h.runtimeSvc.Service().EnableRepository(name)
	case "disable":
		err = h.runtimeSvc.Service().DisableRepository(name)
	case "set-default":
		err = h.runtimeSvc.Service().SetDefaultRepository(name)
	default:
		writeError(w, http.StatusNotFound, "NOT_FOUND", "接口不存在")
		return
	}
	if err != nil {
		writeWrappedError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, httpapibiz.Response[map[string]string]{Code: httpapibiz.CodeOK, Data: map[string]string{"status": "ok"}})
}

func (h *HTTPAPI) handleSkillRepositoryStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	status, err := h.runtimeSvc.Service().SkillRepositoryStatus()
	if err != nil {
		writeWrappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.SkillRepositoryStatusData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.SkillRepositoryStatusData{Status: status},
	})
}

func (h *HTTPAPI) handleCheckSkillRepository(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	result, err := h.runtimeSvc.Service().CheckSkillRepositoryUpdates()
	if err != nil {
		writeWrappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.SkillRepositoryCheckData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.SkillRepositoryCheckData{
			Status:       result.Status,
			Message:      result.Message,
			RemoteURL:    result.RemoteURL,
			LocalCommit:  result.LocalCommit,
			RemoteCommit: result.RemoteCommit,
			HasUpdates:   result.HasUpdates,
			Ahead:        result.Ahead,
			Behind:       result.Behind,
		},
	})
}

func (h *HTTPAPI) handleSyncSkillRepository(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	if err := h.runtimeSvc.Service().SyncSkillRepositoryAndRefresh(); err != nil {
		writeWrappedError(w, err)
		return
	}
	skillCount := 0
	if defaultRepo, err := h.runtimeSvc.Service().DefaultRepository(); err == nil && defaultRepo != nil {
		if skills, err := h.runtimeSvc.Service().ListSkillMetadata([]string{defaultRepo.Name}); err == nil {
			skillCount = len(skills)
		}
	}
	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.SyncSkillRepositoryData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.SyncSkillRepositoryData{Status: "synced", SkillCount: skillCount},
	})
}

func (h *HTTPAPI) handlePushSkillRepositoryPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	preview, err := h.pushSkillRepositoryPreview()
	if err != nil {
		writeWrappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.PushSkillRepositoryPreviewData]{
		Code: httpapibiz.CodeOK,
		Data: preview,
	})
}

func (h *HTTPAPI) handlePushSkillRepository(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	var req httpapibiz.PushSkillRepositoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "请求体格式无效")
		return
	}
	if !req.Confirm {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "推送默认仓库需要 confirm=true")
		return
	}
	preview, err := h.pushSkillRepositoryPreview()
	if err != nil {
		writeWrappedError(w, err)
		return
	}
	if !preview.HasChanges {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "没有要推送的更改")
		return
	}
	if len(req.ExpectedChangedFiles) > 0 && !sameStringSet(req.ExpectedChangedFiles, preview.ChangedFiles) {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "待推送文件已变化，请重新预览后再推送")
		return
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = preview.SuggestedMessage
	}
	if err := h.runtimeSvc.Service().PushSkillRepositoryChanges(message); err != nil {
		writeWrappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.PushSkillRepositoryData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.PushSkillRepositoryData{Status: "pushed", Message: message, ChangedFiles: preview.ChangedFiles},
	})
}

func (h *HTTPAPI) pushSkillRepositoryPreview() (httpapibiz.PushSkillRepositoryPreviewData, error) {
	status, err := h.runtimeSvc.Service().SkillRepositoryStatus()
	if err != nil {
		return httpapibiz.PushSkillRepositoryPreviewData{}, err
	}
	changedFiles := skillRepositoryChangedFiles(status)
	preview := httpapibiz.PushSkillRepositoryPreviewData{
		HasChanges:       len(changedFiles) > 0,
		ChangedFiles:     changedFiles,
		SuggestedMessage: gitpkg.SuggestedCommitMessage(changedFiles),
		RawStatus:        status,
	}
	if defaultRepo, err := h.runtimeSvc.Service().DefaultRepository(); err == nil && defaultRepo != nil {
		preview.DefaultRepo = defaultRepo.Name
		preview.RemoteURL = defaultRepo.URL
	}
	return preview, nil
}

func (h *HTTPAPI) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))
	if keyword == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "缺少 keyword 参数")
		return
	}

	limit := 20
	if limitParam := strings.TrimSpace(r.URL.Query().Get("limit")); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	items, err := h.runtimeSvc.Service().SearchRemoteSkills(keyword, limit)
	if err != nil {
		writeWrappedError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.RemoteSearchData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.RemoteSearchData{Items: items},
	})
}

func (h *HTTPAPI) handleSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	var repoNames []string
	if repoParam := strings.TrimSpace(r.URL.Query().Get("repo")); repoParam != "" {
		for _, item := range strings.Split(repoParam, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				repoNames = append(repoNames, item)
			}
		}
	}

	skills, err := h.runtimeSvc.Service().ListSkillMetadata(repoNames)
	if err != nil {
		writeWrappedError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.SkillListData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.SkillListData{
			Items: skills,
			Total: len(skills),
		},
	})
}

func (h *HTTPAPI) handleSkillActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/skills/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "接口不存在")
		return
	}

	if strings.HasSuffix(path, "/candidates") {
		skillID, err := url.PathUnescape(strings.TrimSuffix(path, "/candidates"))
		if err != nil || strings.TrimSpace(skillID) == "" {
			writeError(w, http.StatusBadRequest, "INVALID_INPUT", "缺少 skill_id")
			return
		}
		items, err := h.runtimeSvc.Service().FindSkill(skillID)
		if err != nil {
			writeWrappedError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.SkillCandidateListData]{
			Code: httpapibiz.CodeOK,
			Data: httpapibiz.SkillCandidateListData{Items: items},
		})
		return
	}

	skillID, err := url.PathUnescape(path)
	if err != nil || strings.TrimSpace(skillID) == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "缺少 skill_id")
		return
	}

	repoName := strings.TrimSpace(r.URL.Query().Get("repo"))
	if repoName == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "缺少 repo 参数")
		return
	}

	item, err := h.runtimeSvc.Service().LoadSkill(skillID, repoName)
	if err != nil {
		writeWrappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.SkillDetailData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.SkillDetailData{Item: item},
	})
}

func (h *HTTPAPI) handleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	projects, err := h.inventorySvc.Service().ListProjects()
	if err != nil {
		writeWrappedError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.ProjectListData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.ProjectListData{Items: projects},
	})
}

func (h *HTTPAPI) handleProjectActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/projects/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "接口不存在")
		return
	}

	parts := strings.Split(path, "/")
	id := parts[0]

	if len(parts) == 1 {
		project, err := h.inventorySvc.Service().GetProject(id)
		if err != nil {
			writeWrappedError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.ProjectDetailData]{
			Code: httpapibiz.CodeOK,
			Data: httpapibiz.ProjectDetailData{Item: project},
		})
		return
	}

	if len(parts) == 2 && parts[1] == "skills" {
		skills, err := h.inventorySvc.Service().ListProjectSkills(id)
		if err != nil {
			writeWrappedError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.ProjectSkillListData]{
			Code: httpapibiz.CodeOK,
			Data: httpapibiz.ProjectSkillListData{Items: skills},
		})
		return
	}

	writeError(w, http.StatusNotFound, "NOT_FOUND", "接口不存在")
}

func (h *HTTPAPI) handleProjectStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	projectPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if projectPath == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "缺少 path 参数")
		return
	}

	skillID := strings.TrimSpace(r.URL.Query().Get("skill_id"))
	status, err := h.statusSvc.Service().Inspect(projectPath, skillID)
	if err != nil {
		writeWrappedError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.ProjectStatusData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.ProjectStatusData{Item: status},
	})
}

func (h *HTTPAPI) handleUseSkill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	var req httpapibiz.UseSkillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "请求体格式无效")
		return
	}

	result, err := h.useSvc.Service().EnableSkill(req.ProjectPath, req.SkillID, req.Repository, req.Variables)
	if err != nil {
		writeWrappedError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.UseSkillData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.UseSkillData{Item: result},
	})
}

func (h *HTTPAPI) handleRegisterSkill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	var req httpapibiz.RegisterSkillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "请求体格式无效")
		return
	}
	if strings.TrimSpace(req.ProjectPath) == "" || strings.TrimSpace(req.SkillID) == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "缺少 project_path 或 skill_id")
		return
	}

	result, err := h.runtimeSvc.Service().RegisterProjectSkill(req.ProjectPath, req.SkillID, req.SkipValidate)
	if err != nil {
		writeWrappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.RegisterSkillData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.RegisterSkillData{Item: result},
	})
}

func (h *HTTPAPI) handleImportSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	var req httpapibiz.ImportSkillsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "请求体格式无效")
		return
	}
	if strings.TrimSpace(req.ProjectPath) == "" || strings.TrimSpace(req.SkillsDir) == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "缺少 project_path 或 skills_dir")
		return
	}

	result, err := h.runtimeSvc.Service().ImportProjectSkills(req.ProjectPath, req.SkillsDir, req.Options)
	if err != nil && result == nil {
		writeWrappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.ImportSkillsData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.ImportSkillsData{Item: result},
	})
}

func (h *HTTPAPI) handleDedupeSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	var req httpapibiz.DedupeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "请求体格式无效")
		return
	}
	if strings.TrimSpace(req.Scope) == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "缺少 scope")
		return
	}

	result, err := h.runtimeSvc.Service().DedupeProjectSkills(req.Scope, req.Options)
	if err != nil && result == nil {
		writeWrappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.DedupeData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.DedupeData{Item: result},
	})
}

func (h *HTTPAPI) handleSyncCopies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	var req httpapibiz.SyncCopiesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "请求体格式无效")
		return
	}
	if strings.TrimSpace(req.Options.Scope) == "" || strings.TrimSpace(req.Options.Canonical) == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "缺少 scope 或 canonical")
		return
	}

	result, err := h.runtimeSvc.Service().SyncProjectSkillCopies(req.Options)
	if err != nil && result == nil {
		writeWrappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.SyncCopiesData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.SyncCopiesData{Item: result},
	})
}

func (h *HTTPAPI) handleLintPaths(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	var req httpapibiz.PathLintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "请求体格式无效")
		return
	}
	if strings.TrimSpace(req.Options.Scope) == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "缺少 scope")
		return
	}

	result, err := h.runtimeSvc.Service().LintProjectSkillPaths(req.Options)
	if err != nil && result == nil {
		writeWrappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.PathLintData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.PathLintData{Item: result},
	})
}

func (h *HTTPAPI) handleValidateProjectSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	var req httpapibiz.ValidateProjectSkillsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "请求体格式无效")
		return
	}
	if strings.TrimSpace(req.Options.ProjectPath) == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "缺少 project_path")
		return
	}
	if !req.Options.All && strings.TrimSpace(req.Options.SkillID) == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "缺少 skill_id")
		return
	}

	result, err := h.runtimeSvc.Service().ValidateProjectSkills(req.Options)
	if err != nil && result == nil {
		writeWrappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.ValidateProjectSkillsData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.ValidateProjectSkillsData{Item: result},
	})
}

func (h *HTTPAPI) handleAuditProjectSkills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	var req httpapibiz.AuditProjectSkillsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "请求体格式无效")
		return
	}
	if strings.TrimSpace(req.Options.ProjectPath) == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "缺少 project_path")
		return
	}

	result, err := h.runtimeSvc.Service().AuditProjectSkills(req.Options)
	if err != nil && result == nil {
		writeWrappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.AuditProjectSkillsData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.AuditProjectSkillsData{Item: result},
	})
}

func (h *HTTPAPI) handleApplyProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	var req httpapibiz.ApplyProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "请求体格式无效")
		return
	}
	if strings.TrimSpace(req.ProjectPath) == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "缺少 project_path")
		return
	}

	result, err := h.applySvc.Service().Apply(req.ProjectPath, req.SkillID, req.DryRun, req.Force)
	if err != nil {
		writeWrappedError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.ApplyProjectData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.ApplyProjectData{Item: result},
	})
}

func (h *HTTPAPI) handleGlobalStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	skillID := strings.TrimSpace(r.URL.Query().Get("skill_id"))
	agents := parseAgentQuery(r.URL.Query())
	result, err := h.globalSvc.Inspect(skillID, agents)
	if err != nil {
		writeWrappedError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.GlobalStatusData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.GlobalStatusData{Item: result},
	})
}

func (h *HTTPAPI) handleUseGlobalSkill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	var req httpapibiz.UseGlobalSkillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "请求体格式无效")
		return
	}
	if strings.TrimSpace(req.SkillID) == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "缺少 skill_id")
		return
	}

	result, err := h.globalSvc.EnableSkill(req.SkillID, req.Repository, req.Agents, req.Variables)
	if err != nil {
		writeWrappedError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.UseGlobalSkillData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.UseGlobalSkillData{Item: result},
	})
}

func (h *HTTPAPI) handleApplyGlobal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	var req httpapibiz.ApplyGlobalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "请求体格式无效")
		return
	}

	result, err := h.globalSvc.Apply(req.SkillID, req.Agents, req.DryRun, req.Force)
	if err != nil {
		writeWrappedError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.ApplyGlobalData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.ApplyGlobalData{Item: result},
	})
}

func (h *HTTPAPI) handleGlobalSkillActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	prefix := "/api/v1/global-skills/"
	skillID, err := url.PathUnescape(strings.TrimPrefix(r.URL.Path, prefix))
	if err != nil || strings.TrimSpace(skillID) == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "缺少 skill_id")
		return
	}

	force := strings.EqualFold(r.URL.Query().Get("force"), "true") || r.URL.Query().Get("force") == "1"
	result, err := h.globalSvc.Remove(skillID, parseAgentQuery(r.URL.Query()), force)
	if err != nil {
		writeWrappedError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.RemoveGlobalSkillData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.RemoveGlobalSkillData{Item: result},
	})
}

func (h *HTTPAPI) handlePreviewFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	var req httpapibiz.FeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "请求体格式无效")
		return
	}
	if strings.TrimSpace(req.ProjectPath) == "" || strings.TrimSpace(req.SkillID) == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "缺少 project_path 或 skill_id")
		return
	}

	result, err := h.feedbackSvc.Service().Preview(req.ProjectPath, req.SkillID)
	if err != nil {
		writeWrappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.FeedbackPreviewData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.FeedbackPreviewData{Item: result},
	})
}

func (h *HTTPAPI) handleApplyFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "方法不支持")
		return
	}

	var req httpapibiz.FeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "请求体格式无效")
		return
	}
	if strings.TrimSpace(req.ProjectPath) == "" || strings.TrimSpace(req.SkillID) == "" {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "缺少 project_path 或 skill_id")
		return
	}

	result, err := h.feedbackSvc.Service().Apply(req.ProjectPath, req.SkillID)
	if err != nil {
		writeWrappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.FeedbackPreviewData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.FeedbackPreviewData{Item: result},
	})
}

func skillRepositoryChangedFiles(status string) []string {
	var files []string
	for _, line := range strings.Split(status, "\n") {
		line = strings.TrimRight(line, "\r")
		if !isSkillRepositoryChangeLine(line) {
			continue
		}
		if len(line) > 3 {
			files = append(files, strings.TrimSpace(line[3:]))
		} else {
			files = append(files, strings.TrimSpace(line))
		}
	}
	return files
}

func isSkillRepositoryChangeLine(line string) bool {
	for _, prefix := range []string{" M ", "?? ", " D ", "M  ", "A  ", "D  ", "R  ", "C  ", "AM ", "MM ", "AD ", "MD "} {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func parseAgentQuery(values url.Values) []string {
	var agents []string
	for _, raw := range values["agent"] {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				agents = append(agents, part)
			}
		}
	}
	return agents
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := map[string]int{}
	for _, item := range left {
		seen[item]++
	}
	for _, item := range right {
		seen[item]--
		if seen[item] < 0 {
			return false
		}
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"code":    code,
		"message": message,
	})
}

func writeWrappedError(w http.ResponseWriter, err error) {
	code := apperrors.Code(err)
	if code == "" {
		code = apperrors.ErrSystem
	}
	message := apperrors.Message(err)
	if message == "" {
		message = err.Error()
	}
	writeError(w, httpStatusForErrorCode(code), string(code), message)
}

func httpStatusForErrorCode(code apperrors.ErrorCode) int {
	switch code {
	case apperrors.ErrSkillNotFound, apperrors.ErrProjectNotFound, apperrors.ErrFileNotFound:
		return http.StatusNotFound
	case apperrors.ErrFilePermission:
		return http.StatusForbidden
	case apperrors.ErrNetwork, apperrors.ErrAPIRequest, apperrors.ErrGitRemote:
		return http.StatusBadGateway
	case apperrors.ErrNotImplemented:
		return http.StatusNotImplemented
	case apperrors.ErrSystem:
		return http.StatusInternalServerError
	default:
		return http.StatusBadRequest
	}
}

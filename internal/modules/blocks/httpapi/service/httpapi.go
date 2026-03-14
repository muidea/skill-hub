package service

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/muidea/skill-hub/internal/config"
	httpapibiz "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/biz"
	projectapplymodule "github.com/muidea/skill-hub/internal/modules/kernel/project_apply"
	projectfeedbackmodule "github.com/muidea/skill-hub/internal/modules/kernel/project_feedback"
	projectinventorymodule "github.com/muidea/skill-hub/internal/modules/kernel/project_inventory"
	projectstatusmodule "github.com/muidea/skill-hub/internal/modules/kernel/project_status"
	projectusemodule "github.com/muidea/skill-hub/internal/modules/kernel/project_use"
	runtimemodule "github.com/muidea/skill-hub/internal/modules/kernel/runtime"
	"github.com/muidea/skill-hub/pkg/spec"
)

type HTTPAPI struct {
	runtimeSvc   *runtimemodule.Runtime
	inventorySvc *projectinventorymodule.ProjectInventory
	statusSvc    *projectstatusmodule.ProjectStatus
	useSvc       *projectusemodule.ProjectUse
	applySvc     *projectapplymodule.ProjectApply
	feedbackSvc  *projectfeedbackmodule.ProjectFeedback
}

func New() *HTTPAPI {
	return &HTTPAPI{
		runtimeSvc:   runtimemodule.New(),
		inventorySvc: projectinventorymodule.New(),
		statusSvc:    projectstatusmodule.New(),
		useSvc:       projectusemodule.New(),
		applySvc:     projectapplymodule.New(),
		feedbackSvc:  projectfeedbackmodule.New(),
	}
}

func (h *HTTPAPI) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", h.handleHealth)
	mux.HandleFunc("/api/v1/repos", h.handleRepos)
	mux.HandleFunc("/api/v1/repos/", h.handleRepoActions)
	mux.HandleFunc("/api/v1/skills", h.handleSkills)
	mux.HandleFunc("/api/v1/skills/", h.handleSkillActions)
	mux.HandleFunc("/api/v1/projects", h.handleProjects)
	mux.HandleFunc("/api/v1/projects/", h.handleProjectActions)
	mux.HandleFunc("/api/v1/project-status", h.handleProjectStatus)
	mux.HandleFunc("/api/v1/project-skills/use", h.handleUseSkill)
	mux.HandleFunc("/api/v1/project-apply", h.handleApplyProject)
	mux.HandleFunc("/api/v1/project-feedback/preview", h.handlePreviewFeedback)
	mux.HandleFunc("/api/v1/project-feedback/apply", h.handleApplyFeedback)
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

	target := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("target")))
	if target != "" {
		skills = filterSkillsByTarget(skills, target)
	}

	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.SkillListData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.SkillListData{Items: skills},
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

	result, err := h.useSvc.Service().EnableSkill(req.ProjectPath, req.SkillID, req.Repository, req.Target, req.Variables)
	if err != nil {
		writeWrappedError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.UseSkillData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.UseSkillData{Item: result},
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

	result, err := h.applySvc.Service().Apply(req.ProjectPath, req.DryRun, req.Force)
	if err != nil {
		writeWrappedError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, httpapibiz.Response[httpapibiz.ApplyProjectData]{
		Code: httpapibiz.CodeOK,
		Data: httpapibiz.ApplyProjectData{Item: result},
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

func filterSkillsByTarget(skills []spec.SkillMetadata, target string) []spec.SkillMetadata {
	var filtered []spec.SkillMetadata
	for _, item := range skills {
		compat := strings.ToLower(item.Compatibility)
		switch target {
		case spec.TargetCursor:
			if strings.Contains(compat, "cursor") {
				filtered = append(filtered, item)
			}
		case spec.TargetClaude, spec.TargetClaudeCode:
			if strings.Contains(compat, "claude") || strings.Contains(compat, "claude_code") {
				filtered = append(filtered, item)
			}
		case spec.TargetOpenCode, "opencode":
			if strings.Contains(compat, "open_code") || strings.Contains(compat, "opencode") {
				filtered = append(filtered, item)
			}
		default:
			filtered = append(filtered, item)
		}
	}
	return filtered
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
	writeError(w, http.StatusBadRequest, "REQUEST_FAILED", err.Error())
}

package service

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/muidea/skill-hub/pkg/errors"
)

const (
	defaultRepoOwner = "muidea"
	defaultRepoName  = "skill-hub"
	defaultAPIBase   = "https://api.github.com"
	defaultDownloads = "https://github.com"
)

type Options struct {
	CurrentVersion  string `json:"current_version"`
	TargetVersion   string `json:"target_version,omitempty"`
	InstallPath     string `json:"install_path,omitempty"`
	GOOS            string `json:"goos,omitempty"`
	GOARCH          string `json:"goarch,omitempty"`
	APIBaseURL      string `json:"api_base_url,omitempty"`
	DownloadBaseURL string `json:"download_base_url,omitempty"`
	RepoOwner       string `json:"repo_owner,omitempty"`
	RepoName        string `json:"repo_name,omitempty"`
	CheckOnly       bool   `json:"check_only,omitempty"`
	DryRun          bool   `json:"dry_run,omitempty"`
	Force           bool   `json:"force,omitempty"`
	SkipAgentSkills bool   `json:"skip_agent_skills,omitempty"`
	NoRestartServe  bool   `json:"no_restart_serve,omitempty"`
}

type Result struct {
	CurrentVersion       string   `json:"current_version"`
	LatestVersion        string   `json:"latest_version,omitempty"`
	TargetVersion        string   `json:"target_version,omitempty"`
	Status               string   `json:"status"`
	UpdateAvailable      bool     `json:"update_available"`
	DryRun               bool     `json:"dry_run,omitempty"`
	CheckOnly            bool     `json:"check_only,omitempty"`
	InstallPath          string   `json:"install_path,omitempty"`
	ArchiveName          string   `json:"archive_name,omitempty"`
	ChecksumName         string   `json:"checksum_name,omitempty"`
	AgentSkillsInstalled int      `json:"agent_skills_installed,omitempty"`
	ServeRestarted       []string `json:"serve_restarted,omitempty"`
	Warnings             []string `json:"warnings,omitempty"`
}

type Service struct {
	client *http.Client
}

func New() *Service {
	return &Service{client: &http.Client{Timeout: 30 * time.Second}}
}

func NewWithHTTPClient(client *http.Client) *Service {
	if client == nil {
		return New()
	}
	return &Service{client: client}
}

func (s *Service) Check(ctx context.Context, opts Options) (*Result, error) {
	opts = normalizeOptions(opts)
	current := normalizeVersion(opts.CurrentVersion)
	result := &Result{
		CurrentVersion: current,
		DryRun:         opts.DryRun,
		CheckOnly:      opts.CheckOnly,
		InstallPath:    opts.InstallPath,
	}

	target := normalizeVersion(opts.TargetVersion)
	if target == "" {
		latest, err := s.fetchLatestVersion(ctx, opts)
		if err != nil {
			return nil, err
		}
		target = latest
		result.LatestVersion = latest
	} else {
		result.LatestVersion = target
	}
	result.TargetVersion = target

	if current == "" || current == "dev" {
		result.Status = "unknown_current_version"
		result.UpdateAvailable = target != ""
		if target != "" {
			result.Status = "update_available"
		}
		return result, nil
	}

	cmp, err := compareVersions(current, target)
	if err != nil {
		return nil, err
	}
	switch {
	case cmp < 0:
		result.Status = "update_available"
		result.UpdateAvailable = true
	case cmp == 0:
		result.Status = "up_to_date"
	case opts.Force:
		result.Status = "force_target"
		result.UpdateAvailable = true
	default:
		result.Status = "target_not_newer"
	}
	return result, nil
}

func (s *Service) Upgrade(ctx context.Context, opts Options) (*Result, error) {
	opts = normalizeOptions(opts)
	result, err := s.Check(ctx, opts)
	if err != nil {
		return nil, err
	}
	if !result.UpdateAvailable && !opts.Force {
		return result, nil
	}

	archiveName, checksumName, err := releaseAssetNames(opts.GOOS, opts.GOARCH)
	if err != nil {
		return nil, err
	}
	result.ArchiveName = archiveName
	result.ChecksumName = checksumName

	if opts.CheckOnly {
		result.Status = "check_complete"
		return result, nil
	}
	if opts.DryRun {
		result.Status = "planned"
		return result, nil
	}
	if opts.GOOS == "windows" {
		return result, errors.NewWithCode("upgrade", errors.ErrNotImplemented, "Windows 暂不支持运行中自动替换，请使用 install-latest.sh 或下载 Release 包手动升级")
	}

	tempDir, err := os.MkdirTemp("", "skill-hub-upgrade-*")
	if err != nil {
		return nil, errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "创建临时目录失败")
	}
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, archiveName)
	checksumPath := filepath.Join(tempDir, checksumName)
	tag := ensureVPrefix(result.TargetVersion)
	downloadBase := strings.TrimRight(opts.DownloadBaseURL, "/")
	releaseBase := fmt.Sprintf("%s/%s/%s/releases/download/%s", downloadBase, opts.RepoOwner, opts.RepoName, tag)

	if err := s.downloadFile(ctx, releaseBase+"/"+archiveName, archivePath); err != nil {
		return nil, err
	}
	if err := s.downloadFile(ctx, releaseBase+"/"+checksumName, checksumPath); err != nil {
		return nil, err
	}
	if err := verifySHA256File(archivePath, checksumPath); err != nil {
		return nil, err
	}

	extractDir := filepath.Join(tempDir, "extract")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return nil, errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "创建解压目录失败")
	}
	if err := extractTarGz(archivePath, extractDir); err != nil {
		return nil, err
	}

	binaryName := "skill-hub"
	if opts.GOOS == "windows" {
		binaryName = "skill-hub.exe"
	}
	newBinary, err := findExtractedBinary(extractDir, binaryName)
	if err != nil {
		return nil, err
	}
	if err := validateBinaryVersion(ctx, newBinary, result.TargetVersion); err != nil {
		return nil, err
	}
	if err := replaceBinary(newBinary, opts.InstallPath); err != nil {
		return nil, err
	}

	result.Status = "upgraded"
	if !opts.SkipAgentSkills {
		count, warnings := installBundledAgentSkills(filepath.Join(extractDir, "agent-skills"))
		result.AgentSkillsInstalled = count
		result.Warnings = append(result.Warnings, warnings...)
	}
	if !opts.NoRestartServe {
		restarted, warnings := restartRegisteredServeInstances(ctx, opts.InstallPath)
		result.ServeRestarted = restarted
		result.Warnings = append(result.Warnings, warnings...)
	}
	return result, nil
}

func normalizeOptions(opts Options) Options {
	if opts.RepoOwner == "" {
		opts.RepoOwner = defaultRepoOwner
	}
	if opts.RepoName == "" {
		opts.RepoName = defaultRepoName
	}
	if opts.APIBaseURL == "" {
		opts.APIBaseURL = defaultAPIBase
	}
	if opts.DownloadBaseURL == "" {
		opts.DownloadBaseURL = defaultDownloads
	}
	if opts.GOOS == "" {
		opts.GOOS = runtime.GOOS
	}
	if opts.GOARCH == "" {
		opts.GOARCH = runtime.GOARCH
	}
	if opts.InstallPath == "" {
		if executable, err := os.Executable(); err == nil {
			opts.InstallPath = executable
		}
	}
	return opts
}

func (s *Service) fetchLatestVersion(ctx context.Context, opts Options) (string, error) {
	apiBase := strings.TrimRight(opts.APIBaseURL, "/")
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", apiBase, opts.RepoOwner, opts.RepoName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", errors.WrapWithCode(err, "upgrade", errors.ErrAPIRequest, "创建版本检测请求失败")
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", errors.WrapWithCode(err, "upgrade", errors.ErrNetwork, "获取最新版本失败")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errors.NewWithCodef("upgrade", errors.ErrAPIRequest, "获取最新版本失败: HTTP %d", resp.StatusCode)
	}
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", errors.WrapWithCode(err, "upgrade", errors.ErrAPIRequest, "解析最新版本响应失败")
	}
	latest := normalizeVersion(payload.TagName)
	if latest == "" {
		return "", errors.NewWithCode("upgrade", errors.ErrAPIRequest, "最新版本响应缺少 tag_name")
	}
	return latest, nil
}

func (s *Service) downloadFile(ctx context.Context, url, output string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return errors.WrapWithCode(err, "upgrade", errors.ErrAPIRequest, "创建下载请求失败")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return errors.WrapWithCode(err, "upgrade", errors.ErrNetwork, "下载 Release 资产失败")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.NewWithCodef("upgrade", errors.ErrAPIRequest, "下载 Release 资产失败: HTTP %d: %s", resp.StatusCode, url)
	}

	file, err := os.Create(output)
	if err != nil {
		return errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "创建下载文件失败")
	}
	defer file.Close()
	if _, err := io.Copy(file, resp.Body); err != nil {
		return errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "写入下载文件失败")
	}
	return nil
}

func releaseAssetNames(goos, goarch string) (string, string, error) {
	switch goos {
	case "linux", "darwin", "windows":
	default:
		return "", "", errors.NewWithCodef("upgrade", errors.ErrNotImplemented, "不支持的系统: %s", goos)
	}
	switch goarch {
	case "amd64", "arm64":
	default:
		return "", "", errors.NewWithCodef("upgrade", errors.ErrNotImplemented, "不支持的架构: %s", goarch)
	}
	base := fmt.Sprintf("skill-hub-%s-%s", goos, goarch)
	return base + ".tar.gz", base + ".sha256", nil
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")
	if idx := strings.IndexByte(v, ' '); idx >= 0 {
		v = v[:idx]
	}
	return v
}

func ensureVPrefix(v string) string {
	v = normalizeVersion(v)
	if v == "" {
		return ""
	}
	return "v" + v
}

type parsedVersion struct {
	major int
	minor int
	patch int
	pre   string
}

func compareVersions(a, b string) (int, error) {
	av, err := parseVersion(a)
	if err != nil {
		return 0, err
	}
	bv, err := parseVersion(b)
	if err != nil {
		return 0, err
	}
	for _, pair := range [][2]int{{av.major, bv.major}, {av.minor, bv.minor}, {av.patch, bv.patch}} {
		if pair[0] < pair[1] {
			return -1, nil
		}
		if pair[0] > pair[1] {
			return 1, nil
		}
	}
	if av.pre == bv.pre {
		return 0, nil
	}
	if av.pre == "" {
		return 1, nil
	}
	if bv.pre == "" {
		return -1, nil
	}
	if av.pre < bv.pre {
		return -1, nil
	}
	return 1, nil
}

func parseVersion(v string) (parsedVersion, error) {
	v = normalizeVersion(v)
	var parsed parsedVersion
	if v == "" {
		return parsed, errors.NewWithCode("upgrade", errors.ErrInvalidInput, "版本号不能为空")
	}
	if plus := strings.IndexByte(v, '+'); plus >= 0 {
		v = v[:plus]
	}
	if dash := strings.IndexByte(v, '-'); dash >= 0 {
		parsed.pre = v[dash+1:]
		v = v[:dash]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return parsed, errors.NewWithCodef("upgrade", errors.ErrInvalidInput, "版本号格式无效: %s", v)
	}
	values := []*int{&parsed.major, &parsed.minor, &parsed.patch}
	for idx, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil || value < 0 {
			return parsed, errors.NewWithCodef("upgrade", errors.ErrInvalidInput, "版本号格式无效: %s", v)
		}
		*values[idx] = value
	}
	return parsed, nil
}

func verifySHA256File(filePath, checksumPath string) error {
	expected, err := readExpectedSHA256(checksumPath)
	if err != nil {
		return err
	}
	file, err := os.Open(filePath)
	if err != nil {
		return errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "打开待校验文件失败")
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "计算 sha256 失败")
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(expected, actual) {
		return errors.NewWithCodef("upgrade", errors.ErrValidation, "sha256 校验失败: expected %s, got %s", expected, actual)
	}
	return nil
}

func readExpectedSHA256(checksumPath string) (string, error) {
	content, err := os.ReadFile(checksumPath)
	if err != nil {
		return "", errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "读取 sha256 文件失败")
	}
	fields := strings.Fields(string(content))
	if len(fields) == 0 {
		return "", errors.NewWithCode("upgrade", errors.ErrValidation, "sha256 文件为空")
	}
	value := strings.TrimPrefix(strings.TrimSpace(fields[0]), "sha256:")
	if len(value) != 64 {
		return "", errors.NewWithCodef("upgrade", errors.ErrValidation, "sha256 格式无效: %s", value)
	}
	return value, nil
}

func extractTarGz(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "打开压缩包失败")
	}
	defer file.Close()
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return errors.WrapWithCode(err, "upgrade", errors.ErrValidation, "读取 gzip 压缩包失败")
	}
	defer gzReader.Close()
	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.WrapWithCode(err, "upgrade", errors.ErrValidation, "读取 tar 压缩包失败")
		}
		target, err := secureJoin(destDir, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "创建解压目录失败")
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "创建解压父目录失败")
			}
			if err := writeFileFromReader(target, tarReader, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}
}

func secureJoin(root, name string) (string, error) {
	cleanName := filepath.Clean(name)
	if cleanName == "." || strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
		return "", errors.NewWithCodef("upgrade", errors.ErrValidation, "压缩包包含非法路径: %s", name)
	}
	target := filepath.Join(root, cleanName)
	rel, err := filepath.Rel(root, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", errors.NewWithCodef("upgrade", errors.ErrValidation, "压缩包包含非法路径: %s", name)
	}
	return target, nil
}

func writeFileFromReader(path string, reader io.Reader, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "创建解压文件失败")
	}
	defer file.Close()
	if _, err := io.Copy(file, reader); err != nil {
		return errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "写入解压文件失败")
	}
	return nil
}

func findExtractedBinary(root, name string) (string, error) {
	candidates := []string{filepath.Join(root, name)}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != name {
			return nil
		}
		candidates = append(candidates, path)
		return nil
	})
	if err != nil {
		return "", errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "查找新版二进制失败")
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.NewWithCodef("upgrade", errors.ErrFileNotFound, "Release 包中未找到 %s", name)
}

func validateBinaryVersion(ctx context.Context, binaryPath, expectedVersion string) error {
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "设置新版二进制权限失败")
	}
	output, err := exec.CommandContext(ctx, binaryPath, "--version").CombinedOutput()
	if err != nil {
		return errors.WrapWithCode(err, "upgrade", errors.ErrValidation, "新版二进制版本验证失败")
	}
	installed := parseVersionOutput(string(output))
	if installed != normalizeVersion(expectedVersion) {
		return errors.NewWithCodef("upgrade", errors.ErrValidation, "新版二进制版本不匹配: expected %s, got %s", normalizeVersion(expectedVersion), installed)
	}
	return nil
}

func parseVersionOutput(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "skill-hub version ") {
			value := strings.TrimPrefix(line, "skill-hub version ")
			if idx := strings.IndexByte(value, ' '); idx >= 0 {
				value = value[:idx]
			}
			return normalizeVersion(value)
		}
	}
	return ""
}

func replaceBinary(source, target string) error {
	if target == "" {
		return errors.NewWithCode("upgrade", errors.ErrFileNotFound, "无法确定当前 skill-hub 安装路径")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "创建安装目录失败")
	}
	tmp := filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".new."+strconv.Itoa(os.Getpid()))
	_ = os.Remove(tmp)
	if err := copyFile(source, tmp, 0755); err != nil {
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "替换当前二进制失败")
	}
	return nil
}

func copyFile(source, target string, mode os.FileMode) error {
	src, err := os.Open(source)
	if err != nil {
		return errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "打开源文件失败")
	}
	defer src.Close()
	dst, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "创建目标文件失败")
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return errors.WrapWithCode(err, "upgrade", errors.ErrFileOperation, "复制文件失败")
	}
	return nil
}

func installBundledAgentSkills(sourceDir string) (int, []string) {
	if isDisabled(os.Getenv("SKILL_HUB_INSTALL_AGENT_SKILLS")) {
		return 0, nil
	}
	if info, err := os.Stat(sourceDir); err != nil || !info.IsDir() {
		return 0, []string{"Release 包中未包含 agent-skills，已跳过 agent skill 同步"}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return 0, []string{"无法确定 HOME，已跳过 agent skill 同步"}
	}
	targets := agentSkillTargets(home)
	installed := 0
	var warnings []string
	for _, target := range targets {
		count, err := copySkillHubSkills(sourceDir, target)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		installed += count
	}
	return installed, warnings
}

func agentSkillTargets(home string) []string {
	xdgData := os.Getenv("XDG_DATA_HOME")
	if xdgData == "" {
		xdgData = filepath.Join(home, ".local", "share")
	}
	primary := envOr("SKILL_HUB_AGENT_SKILLS_DIR", filepath.Join(xdgData, "skill-hub", "agent-skills"))
	targets := []string{primary}

	codexHome := envOr("CODEX_HOME", filepath.Join(home, ".codex"))
	codexDir := envOr("CODEX_SKILLS_DIR", filepath.Join(codexHome, "skills"))
	if shouldMirrorAgent("SKILL_HUB_INSTALL_CODEX_SKILLS", codexHome, "codex") {
		targets = append(targets, codexDir)
	}

	opencodeHome := envOr("OPENCODE_HOME", filepath.Join(home, ".config", "opencode"))
	opencodeDir := envOr("OPENCODE_SKILLS_DIR", filepath.Join(opencodeHome, "skills"))
	if shouldMirrorAgent("SKILL_HUB_INSTALL_OPENCODE_SKILLS", opencodeHome, "opencode") {
		targets = append(targets, opencodeDir)
	}

	claudeDir := envOr("CLAUDE_SKILLS_DIR", filepath.Join(home, ".claude", "skills"))
	if shouldMirrorAgent("SKILL_HUB_INSTALL_CLAUDE_SKILLS", claudeDir, "") {
		targets = append(targets, claudeDir)
	}
	return uniqueStrings(targets)
}

func shouldMirrorAgent(settingEnv, markerPath, commandName string) bool {
	setting := os.Getenv(settingEnv)
	if isDisabled(setting) {
		return false
	}
	if setting != "" && setting != "auto" {
		return true
	}
	if commandName != "" {
		if _, err := exec.LookPath(commandName); err == nil {
			return true
		}
	}
	if info, err := os.Stat(markerPath); err == nil && info.IsDir() {
		return true
	}
	return false
}

func isDisabled(value string) bool {
	switch value {
	case "0", "false", "False", "FALSE", "no", "No", "NO", "off", "Off", "OFF":
		return true
	default:
		return false
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func copySkillHubSkills(sourceDir, targetDir string) (int, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return 0, fmt.Errorf("无法创建 agent skills 目录 %s: %w", targetDir, err)
	}
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return 0, fmt.Errorf("读取 agent skills 目录失败 %s: %w", sourceDir, err)
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "skill-hub-") {
			continue
		}
		sourceSkill := filepath.Join(sourceDir, entry.Name())
		if _, err := os.Stat(filepath.Join(sourceSkill, "SKILL.md")); err != nil {
			continue
		}
		destSkill := filepath.Join(targetDir, entry.Name())
		tmpSkill := filepath.Join(targetDir, "."+entry.Name()+".new."+strconv.Itoa(os.Getpid()))
		_ = os.RemoveAll(tmpSkill)
		if err := copyDir(sourceSkill, tmpSkill); err != nil {
			_ = os.RemoveAll(tmpSkill)
			return count, err
		}
		if err := os.RemoveAll(destSkill); err != nil {
			_ = os.RemoveAll(tmpSkill)
			return count, err
		}
		if err := os.Rename(tmpSkill, destSkill); err != nil {
			_ = os.RemoveAll(tmpSkill)
			return count, err
		}
		count++
	}
	return count, nil
}

func copyDir(source, target string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(target, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(dest, info.Mode())
		}
		return copyFile(path, dest, info.Mode())
	})
}

func restartRegisteredServeInstances(ctx context.Context, installPath string) ([]string, []string) {
	output, err := exec.CommandContext(ctx, installPath, "serve", "status").CombinedOutput()
	if err != nil {
		return nil, []string{fmt.Sprintf("读取 serve 注册表失败，已跳过重启: %s", strings.TrimSpace(string(output)))}
	}
	var restarted []string
	var warnings []string
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 2 || fields[1] != "running" {
			continue
		}
		name := strings.TrimSpace(fields[0])
		if name == "" {
			continue
		}
		if out, err := exec.CommandContext(ctx, installPath, "serve", "stop", name).CombinedOutput(); err != nil {
			warnings = append(warnings, fmt.Sprintf("停止 serve 实例 %s 失败: %s", name, strings.TrimSpace(string(out))))
			continue
		}
		if out, err := exec.CommandContext(ctx, installPath, "serve", "start", name).CombinedOutput(); err != nil {
			warnings = append(warnings, fmt.Sprintf("启动 serve 实例 %s 失败: %s", name, strings.TrimSpace(string(out))))
			continue
		}
		restarted = append(restarted, name)
	}
	return restarted, warnings
}

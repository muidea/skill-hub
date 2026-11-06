package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
)

const pullPreviewLimit = 4096
const pullPreviewBudget = 64 * 1024
const pullFileLimit = 100

// All diagnostic commands are read-only, use literal paths and bypass external
// diff/textconv helpers. Never parse localized Git error messages for conflicts.
func (r *Repository) pullCommand(args ...string) *exec.Cmd {
	cmd := exec.Command("git", append([]string{"--literal-pathspecs", "-C", r.path, "-c", "color.ui=false", "-c", "core.quotePath=false"}, args...)...)
	cmd.Env = append(os.Environ(), "LC_ALL=C", "GIT_OPTIONAL_LOCKS=0")
	return cmd
}

func (r *Repository) pullChangedFiles() ([]spec.PullFileChange, error) {
	output, err := r.pullCommand("status", "--porcelain=v1", "-z", "--untracked-files=all").Output()
	if err != nil {
		return nil, fmt.Errorf("读取拉取诊断文件状态失败: %w", err)
	}
	parts := strings.Split(string(output), "\x00")
	files := []spec.PullFileChange{}
	for i := 0; i < len(parts); i++ {
		entry := parts[i]
		if len(entry) < 4 {
			continue
		}
		item := spec.PullFileChange{Path: entry[3:], IndexStatus: entry[:1], WorktreeStatus: entry[1:2]}
		if strings.ContainsAny(entry[:2], "RC") && i+1 < len(parts) {
			i++
			item.OriginalPath = parts[i]
		}
		switch entry[:2] {
		case "DD", "AU", "UD", "UA", "DU", "AA", "UU":
			item.ConflictType = entry[:2]
		}
		files = append(files, item)
	}
	return files, nil
}

func hasPullConflicts(files []spec.PullFileChange) bool {
	for _, file := range files {
		if file.ConflictType != "" {
			return true
		}
	}
	return false
}

type pullPreviewWriter struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (w *pullPreviewWriter) Len() int       { return w.buffer.Len() }
func (w *pullPreviewWriter) String() string { return w.buffer.String() }

func (w *pullPreviewWriter) Write(p []byte) (int, error) {
	n := len(p)
	remaining := w.limit - w.Len()
	if n > remaining {
		w.truncated = true
		p = p[:remaining]
	}
	_, _ = w.buffer.Write(p)
	return n, nil
}

func (r *Repository) pullPreview(limit int, args ...string) (string, bool, error) {
	cmd := r.pullCommand(args...)
	output := &pullPreviewWriter{limit: limit}
	stderr := &pullPreviewWriter{limit: pullPreviewLimit}
	cmd.Stdout, cmd.Stderr = output, stderr
	err := cmd.Run()
	data := []byte(strings.ToValidUTF8(output.String(), "�"))
	if len(data) > limit {
		data = data[:limit]
		output.truncated = true
	}
	for len(data) > 0 && !utf8.Valid(data) {
		data = data[:len(data)-1]
	}
	if err != nil {
		return "", output.truncated, fmt.Errorf("git %s: %s: %w", args[0], strings.TrimSpace(stderr.String()), err)
	}
	return string(data), output.truncated, nil
}

func (sr *SkillRepository) pullFailure(kind string, files []spec.PullFileChange, updates *RemoteUpdateStatus, stashRef string) error {
	d := sr.repo.pullDiagnostic(kind, files, updates)
	d.StashRef = stashRef
	return pullDiagnosticError(d)
}

func pullDiagnosticError(d *spec.PullDiagnostic) error {
	code, message := errors.ErrGitDirty, "拉取已暂停：默认仓库存在未提交改动"
	switch d.Kind {
	case "merge_conflict":
		code, message = errors.ErrGitConflict, "拉取已暂停：默认仓库存在未解决的合并冲突"
	case "divergent":
		code, message = errors.ErrGitDiverged, "拉取已暂停：本地与远端提交分叉，无法快进拉取"
	}
	return &errors.AppError{Op: "pull", Code: code, Message: message, Details: map[string]any{"pull": d}}
}

func (r *Repository) pullDiagnostic(kind string, files []spec.PullFileChange, updates *RemoteUpdateStatus) *spec.PullDiagnostic {
	d := &spec.PullDiagnostic{Kind: kind, RepositoryPath: r.path, Files: []spec.PullFileChange{}}
	note := func(err error) {
		if err != nil {
			d.InspectionErrors = append(d.InspectionErrors, err.Error())
		}
	}
	if updates != nil {
		d.LocalCommit, d.RemoteCommit, d.Ahead, d.Behind = updates.LocalCommit, updates.RemoteCommit, updates.Ahead, updates.Behind
	} else {
		local, _, err := r.pullPreview(128, "rev-parse", "--verify", "HEAD^{commit}")
		note(err)
		d.LocalCommit = strings.TrimSpace(local)
		remote, _, err := r.pullPreview(128, "rev-parse", "--verify", r.remoteName+"/main^{commit}")
		note(err)
		d.RemoteCommit = strings.TrimSpace(remote)
		if d.LocalCommit != "" && d.RemoteCommit != "" {
			counts, _, err := r.pullPreview(128, "rev-list", "--left-right", "--count", d.LocalCommit+"..."+d.RemoteCommit)
			note(err)
			fields := strings.Fields(counts)
			if len(fields) == 2 {
				d.Ahead, _ = strconv.Atoi(fields[0])
				d.Behind, _ = strconv.Atoi(fields[1])
			}
		}
	}
	base := ""
	remotePaths := []string{}
	if d.LocalCommit != "" && d.RemoteCommit != "" {
		value, _, err := r.pullPreview(128, "merge-base", d.LocalCommit, d.RemoteCommit)
		note(err)
		base = strings.TrimSpace(value)
		if err == nil {
			// --no-renames includes both old and new paths of remote renames.
			paths, err := r.pullCommand("diff", "--name-only", "-z", "--no-renames", base, d.RemoteCommit, "--").Output()
			note(err)
			if err == nil {
				d.OverlapKnown = true
				remotePaths = strings.Split(strings.TrimSuffix(string(paths), "\x00"), "\x00")
			}
		}
		d.LocalCommits, _, _ = r.pullPreview(pullPreviewLimit, "log", "--format=%h %s", "-5", d.RemoteCommit+".."+d.LocalCommit)
		d.RemoteCommits, _, _ = r.pullPreview(pullPreviewLimit, "log", "--format=%h %s", "-5", d.LocalCommit+".."+d.RemoteCommit)
	}
	budget := pullPreviewBudget
	diff := func(file *spec.PullFileChange, args ...string) string {
		limit := min(pullPreviewLimit, budget)
		if limit == 0 {
			file.Truncated = true
			return ""
		}
		value, truncated, err := r.pullPreview(limit, args...)
		note(err)
		budget -= len(value)
		file.Truncated = file.Truncated || truncated
		file.Binary = file.Binary || strings.Contains(value, "Binary files ") || strings.Contains(value, "GIT binary patch")
		return value
	}
	for i, file := range files {
		if i >= pullFileLimit {
			d.OmittedFiles++
			continue
		}
		paths := []string{file.Path}
		if file.OriginalPath != "" {
			paths = append(paths, file.OriginalPath)
		}
		for _, local := range paths {
			for _, remote := range remotePaths {
				if remote != "" && (local == remote || strings.HasPrefix(local, remote+"/") || strings.HasPrefix(remote, local+"/")) {
					file.OverlapsRemote = true
				}
			}
		}
		flags := []string{"diff", "--no-ext-diff", "--no-textconv", "--no-color"}
		if file.ConflictType != "" {
			mode := "--cc"
			if file.ConflictType == "UD" || file.ConflictType == "DU" {
				// Combined diffs omit modify/delete content because one side is
				// absent. Compare the surviving worktree file with stage 1.
				mode = "--base"
			}
			file.ConflictDiff = diff(&file, append(append([]string{}, flags...), append([]string{mode, "--"}, paths...)...)...)
		} else if file.IndexStatus != "?" {
			file.StagedDiff = diff(&file, append(append([]string{}, flags...), append([]string{"--cached", "--"}, paths...)...)...)
			file.UnstagedDiff = diff(&file, append(append([]string{}, flags...), append([]string{"--"}, paths...)...)...)
		}
		if file.OverlapsRemote && base != "" {
			file.RemoteDiff = diff(&file, append(append([]string{}, flags...), append([]string{base, d.RemoteCommit, "--"}, paths...)...)...)
		}
		d.Files = append(d.Files, file)
	}
	return d
}

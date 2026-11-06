package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/muidea/skill-hub/pkg/spec"
)

func renderPullDiagnostic(d *spec.PullDiagnostic) {
	var writer io.Writer = os.Stderr
	if activeOutput != nil {
		writer = activeOutput.command.ErrOrStderr()
	}
	message := "拉取已暂停：默认仓库存在未提交改动"
	switch d.Kind {
	case "merge_conflict":
		message = "拉取已暂停：默认仓库存在未解决的合并冲突"
	case "divergent":
		message = "拉取已暂停：本地与远端提交分叉，无法快进拉取"
	}
	fmt.Fprintf(writer, "\n⚠ %s\n仓库：%s\n", message, d.RepositoryPath)
	if d.LocalCommit != "" || d.RemoteCommit != "" {
		fmt.Fprintf(writer, "本地提交：%s，远端提交：%s\n本地领先：%d，远端领先：%d\n", shortCommit(d.LocalCommit), shortCommit(d.RemoteCommit), d.Ahead, d.Behind)
	}
	overlaps := 0
	for _, file := range d.Files {
		fmt.Fprintf(writer, "\n%s%s  %s", file.IndexStatus, file.WorktreeStatus, file.Path)
		if file.OriginalPath != "" {
			fmt.Fprintf(writer, "（原路径：%s）", file.OriginalPath)
		}
		fmt.Fprintln(writer)
		if file.ConflictType != "" {
			fmt.Fprintf(writer, "  未解决冲突：%s（%s）\n", conflictDescription(file.ConflictType), file.ConflictType)
		} else if file.IndexStatus == "?" {
			fmt.Fprintln(writer, "  未跟踪文件，无已提交基线差异")
		} else {
			if file.IndexStatus != " " {
				fmt.Fprintln(writer, "  含已暂存改动")
			}
			if file.WorktreeStatus != " " {
				fmt.Fprintln(writer, "  含未暂存改动")
			}
		}
		if file.OverlapsRemote {
			overlaps++
			fmt.Fprintln(writer, "  远端也涉及此路径：潜在冲突，尚未验证内容冲突")
		}
		for _, preview := range []struct{ title, body string }{
			{"已暂存差异", file.StagedDiff}, {"未暂存差异", file.UnstagedDiff},
			{"已有冲突差异", file.ConflictDiff}, {"远端差异", file.RemoteDiff},
		} {
			if preview.body != "" {
				fmt.Fprintf(writer, "  %s：\n%s\n", preview.title, strings.TrimRight(preview.body, "\n"))
			}
		}
		if file.Binary {
			fmt.Fprintln(writer, "  二进制文件：无法展示文本差异")
		}
		if file.Truncated {
			fmt.Fprintln(writer, "  差异预览已截断，可在仓库内使用 git diff / git diff --cached 查看完整差异")
		}
	}
	if d.OmittedFiles > 0 {
		fmt.Fprintf(writer, "\n另有 %d 个文件未展示；请在仓库内执行 git status --short 查看全部文件\n", d.OmittedFiles)
	}
	if d.Kind == "blocked_dirty" {
		if !d.OverlapKnown {
			fmt.Fprintln(writer, "\n无法确认本地与远端的路径重叠，请检查诊断信息")
		} else if overlaps == 0 && d.OmittedFiles == 0 {
			fmt.Fprintln(writer, "\n未发现同路径修改；当前拉取策略仍要求先处理未提交改动")
		}
	}
	if d.LocalCommits != "" {
		fmt.Fprintf(writer, "\n本地独有提交（最多 5 条）：\n%s", d.LocalCommits)
	}
	if d.RemoteCommits != "" {
		fmt.Fprintf(writer, "\n远端独有提交（最多 5 条）：\n%s", d.RemoteCommits)
	}
	for _, message := range d.InspectionErrors {
		fmt.Fprintf(writer, "\n诊断未完整：%s\n", message)
	}
	if d.StashRef == "" {
		fmt.Fprintln(writer, "\n未执行新的合并，工作区内容保持不变。")
	} else {
		fmt.Fprintln(writer, "\n未执行新的合并；本地改动已暂存，工作区已因此改变。")
	}
	switch d.Kind {
	case "merge_conflict":
		fmt.Fprintln(writer, "请先处理上述冲突并完成或中止当前 Git 合并/rebase 操作，再执行 skill-hub pull。--force 不会解决已有冲突。")
	case "divergent":
		fmt.Fprintln(writer, "请先核对双方提交，完成合并或 rebase 后再执行 skill-hub pull。--force 不会解决提交分叉。")
	default:
		fmt.Fprintln(writer, "请提交需要保留的修改后重新执行 skill-hub pull；或使用 skill-hub pull --force 自动 stash 后拉取（桥接执行暂不支持 --force）。")
	}
	if d.StashRef != "" {
		fmt.Fprintf(writer, "\n本地改动保存在 stash %s，未自动恢复；本次 --force 已执行暂存。\n查看：git -C %s stash show -p %s\n", d.StashRef, quotePullPath(d.RepositoryPath), d.StashRef)
	}
}

func quotePullPath(path string) string { return "'" + strings.ReplaceAll(path, "'", "'\"'\"'") + "'" }

func conflictDescription(code string) string {
	return map[string]string{"DD": "双方删除", "AU": "本地新增、远端未合并", "UD": "本地修改、远端删除", "UA": "远端新增、本地未合并", "DU": "本地删除、远端修改", "AA": "双方新增", "UU": "双方修改"}[code]
}

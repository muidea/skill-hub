# v0.14.2 技能反馈标题校验与版本回滚修复

## 问题修复

- 修复 `pkg/skill/integrity.go` 中 `skillHeadings` 将 Markdown fenced code blocks（含反引号 ```` ``` ````、波浪线 `~~~`、嵌套围栏、缩进围栏及 CRLF 行尾）与 YAML frontmatter 中的 `#` / `##` 注释误识别为一级/二级章节标题的问题，避免修改代码块内 Shell/脚本注释时被 `ArchiveToDefaultRepository` 误报 `更新会丢失既有技能核心信息: section: ...`。
- 修复 `skill-hub feedback`（CLI 本地路径 `internal/clis/skill-hub/feedback.go`）在用户 `[y/N]` 确认前即改写工作区 `SKILL.md` 版本号的问题，改为确认（或 `--force`）后再执行自动版本升级。
- 修复 `skill-hub feedback`（CLI 与服务端 `internal/modules/application/project_feedback/service/project_feedback.go`）在归档失败时未恢复工作区 `SKILL.md` 的问题，归档报错时自动回滚 `SKILL.md` 原内容与文件权限。

## 验证范围

- 单元测试覆盖：`pkg/skill/integrity_test.go`（fenced code blocks 注释编辑/删除、frontmatter 注释、嵌套围栏、行内三反引号、CRLF 场景）与 `internal/modules/application/project_feedback/service/project_feedback_test.go`（代码块注释反馈归档通过、归档校验失败自动回滚工作区版本号）。
- 工程门禁：`make lint`（`gofmt` / `go vet` / `staticcheck`）、`make test`（`go test ./... --count 1`）与 `make build`（由发布脚本复核）。

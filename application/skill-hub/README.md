# skill-hub Application

`application/skill-hub` 是 `skill-hub` 的主应用入口目录。

当前推荐入口：

- `application/skill-hub/cmd/main.go`

构建示例：

```bash
go build -o bin/skill-hub ./application/skill-hub/cmd
```

Agent 和自动化调用可使用全局 `--json`。stdout 只返回一个 JSON 文档，进度与诊断写入 stderr；命令不会读取 stdin，也不会自动同意覆盖或删除。成功结果保留各命令的数据结构，失败结果包含 `command_error` 并返回非零退出码。

```bash
skill-hub register my-skill --json
skill-hub import .agents/skills --dry-run --json
```

完整约定见 [CLI 命令规范](../../docs/Skill-Hub命令规范.md#cli-json-调用合同)。

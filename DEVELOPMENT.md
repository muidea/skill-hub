# skill-hub 开发指南

> 本文档面向Skill Hub的开发者、贡献者和高级用户，包含构建、发布和贡献指南。

## 项目结构

### 目录结构

```
skill-hub/
├── cmd/skill-hub/          # 主程序入口
├── internal/               # 内部包（不对外暴露）
│   ├── cli/               # CLI命令实现
│   ├── adapter/           # 适配器层（cursor, claude, opencode）
│   ├── engine/            # 技能引擎
│   ├── multirepo/         # 多仓库管理
│   ├── state/             # 状态管理
│   ├── config/            # 配置管理
│   ├── template/          # 模板引擎
│   └── git/               # Git操作封装
├── pkg/spec/              # 公共数据结构定义
├── examples/              # 技能示例
├── scripts/               # 构建和安装脚本
├── dist/                  # 构建输出目录
└── bin/                   # 本地构建输出
```

### 代码架构

#### 核心组件

1. **CLI层** (`internal/cli/`)
   - 命令解析和路由
   - 用户交互和输出
   - 错误处理和日志
   - 多仓库管理命令

2. **适配器层** (`internal/adapter/`)
   - `cursor/`: Cursor工具适配器
   - `claude/`: Claude Code适配器  
   - `opencode/`: OpenCode适配器
   - 统一的适配器接口

3. **技能引擎** (`internal/engine/`)
   - 技能加载和解析
   - 变量替换和模板渲染
   - 技能兼容性检查
   - 多仓库技能搜索和加载

4. **多仓库管理** (`internal/multirepo/`)
   - 多Git仓库管理
   - 仓库同步和状态跟踪
   - 默认仓库配置
   - 技能跨仓库搜索

5. **状态管理** (`internal/state/`)
   - 项目状态持久化
   - 技能启用状态跟踪
   - 配置同步管理

## 开发环境设置

### 环境要求

- **Go 1.24+**: 主要开发语言
- **Git**: 版本控制
- **make**: 构建工具（推荐）
- **Docker**: 容器化测试（可选）

### 初始化开发环境

```bash
# 1. 克隆仓库
git clone https://github.com/muidea/skill-hub.git
cd skill-hub

# 2. 安装依赖
go mod download
go mod verify

# 3. 运行测试
make test

# 4. 构建项目
make build

# 5. 安装开发版本
make install
```

### 开发工具推荐

```bash
# 代码格式化
go fmt ./...

# 代码检查
go vet ./...

# 测试覆盖率
go test -cover ./...

# 依赖管理
go mod tidy
go mod verify
```

## 构建系统

### Makefile 目标

项目使用Makefile作为构建系统，提供以下主要目标：

#### 开发构建
```bash
# 构建当前平台的二进制
make build

# 清理构建产物
make clean

# 运行所有测试
make test

# 代码检查
make lint
```

#### 发布构建
```bash
# 构建所有平台的发布版本
make release-all VERSION=1.0.0

# 创建发布包和校验文件
make release VERSION=1.0.0
```

#### 开发工具
```bash
# 安装到系统
make install

# 更新依赖
make deps

# 显示帮助
make help
```

### 构建配置

#### 版本信息
构建时自动注入版本信息：
```bash
# 自定义版本信息
make build VERSION=1.0.0 COMMIT=$(git rev-parse --short HEAD)

# 构建标志
LDFLAGS="-X 'skill-hub/internal/cli.version=$(VERSION)' \
         -X 'skill-hub/internal/cli.commit=$(COMMIT)' \
         -X 'skill-hub/internal/cli.date=$(DATE)'"
```

#### 多平台构建
支持以下平台组合：
- **Linux**: amd64, arm64
- **macOS**: amd64, arm64  
- **Windows**: amd64, arm64

```bash
# 交叉编译示例
GOOS=linux GOARCH=amd64 go build -o dist/skill-hub-linux-amd64 ./cmd/skill-hub
GOOS=darwin GOARCH=arm64 go build -o dist/skill-hub-darwin-arm64 ./cmd/skill-hub
```

## 测试

### 测试结构

```
internal/
├── adapter/
│   ├── cursor/
│   │   └── adapter_test.go    # Cursor适配器测试
│   ├── claude/
│   │   └── adapter_test.go    # Claude适配器测试
│   └── opencode/
│       └── adapter_test.go    # OpenCode适配器测试
├── cli/
│   └── apply_test.go          # CLI命令测试
├── engine/
│   └── manager_test.go        # 技能引擎测试
├── state/
│   └── manager_test.go        # 状态管理测试
└── template/
    └── template_test.go       # 模板引擎测试
```

### 运行测试

```bash
# 运行所有测试
make test

# 运行特定包测试
go test ./internal/adapter/cursor
go test ./internal/engine

# 运行测试并显示详细信息
go test -v ./...

# 运行测试并生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 测试策略

#### 单元测试
- 每个导出函数都应该有对应的测试
- 使用表格驱动测试（Table-Driven Tests）
- 模拟外部依赖（文件系统、网络等）

#### 集成测试
- 测试组件间的交互
- 验证端到端的工作流程
- 使用临时目录隔离测试环境

#### 测试工具
```go
// 使用testing.T的TempDir创建临时目录
tmpDir := t.TempDir()

// 使用testify/assert进行断言
import "github.com/stretchr/testify/assert"

func TestExample(t *testing.T) {
    result := SomeFunction()
    assert.Equal(t, expected, result)
    assert.NoError(t, err)
}
```

## 发布流程

### 手动发布

#### 1. 准备发布
```bash
# 更新版本号
# 更新CHANGELOG.md
# 确保所有测试通过
make test

# 清理之前的构建
make clean
```

#### 2. 构建发布版本
```bash
# 构建所有平台
make release-all VERSION=1.0.0

# 验证构建产物
ls -la dist/
sha256sum dist/*

# 创建发布说明
cat > release-notes.md << EOF
## 版本 1.0.0
- 新功能: xxx
- 修复: yyy
- 改进: zzz
EOF
```

#### 3. 创建Git标签
```bash
# 创建带注释的标签
git tag -a v1.0.0 -m "Release v1.0.0"

# 推送标签到远程
git push origin v1.0.0
```

### 自动发布（GitHub Actions）

项目配置了GitHub Actions工作流，自动处理发布流程：

#### 触发条件
- 创建新的git标签（格式：v*）
- 推送到master分支

#### 工作流程
1. **CI检查**: 运行测试和代码检查
2. **构建**: 为所有支持平台构建二进制
3. **发布**: 创建GitHub Release并上传构建产物
4. **验证**: 生成校验和文件并验证

#### 发布脚本
```bash
# 使用发布脚本
./scripts/create-release.sh

# 脚本功能：
# 1. 验证版本格式
# 2. 更新版本文件
# 3. 提交更改
# 4. 创建标签
# 5. 推送更改
```

### 版本管理

#### 版本号规范
使用语义化版本控制（SemVer）：
- **主版本号**: 不兼容的API修改
- **次版本号**: 向下兼容的功能性新增
- **修订号**: 向下兼容的问题修正

#### 版本文件
```bash
# 版本信息存储在代码中
internal/cli/version.go

# 构建时注入版本信息
-ldflags="-X 'skill-hub/internal/cli.version=$(VERSION)'"
```

## 贡献指南

### 贡献流程

#### 1. Fork 项目
- 访问 https://github.com/muidea/skill-hub
- 点击 "Fork" 按钮创建个人副本

#### 2. 创建功能分支
```bash
git clone https://github.com/your-username/skill-hub.git
cd skill-hub

# 创建功能分支
git checkout -b feature/amazing-feature
```

#### 3. 开发实现
```bash
# 实现功能
# 添加测试
# 更新文档

# 提交更改
git add .
git commit -m "Add amazing feature"

# 保持分支同步
git fetch upstream
git rebase upstream/master
```

#### 4. 推送到分支
```bash
git push origin feature/amazing-feature
```

#### 5. 创建Pull Request
- 访问你的GitHub仓库
- 点击 "Compare & pull request"
- 填写PR描述，包括：
  - 功能说明
  - 测试情况
  - 相关Issue
- 等待代码审查

### 开发要求

#### 代码风格
- 遵循Go官方代码风格
- 使用 `go fmt` 格式化代码
- 使用有意义的变量和函数名
- 添加必要的注释和文档

#### 测试要求
- 新功能必须包含测试
- 修复bug时添加回归测试
- 保持测试覆盖率不降低
- 使用表格驱动测试（TDT）

#### 文档要求
- 更新相关文档（README、代码注释等）
- 添加使用示例
- 更新CHANGELOG（如果适用）

#### 兼容性要求
- 保持向后兼容性
- 如需破坏性更改，提供迁移指南
- 考虑不同平台和环境的兼容性

### 代码审查

#### 审查要点
1. **功能正确性**: 实现是否符合需求
2. **代码质量**: 是否遵循代码规范
3. **测试覆盖**: 是否有足够的测试
4. **性能影响**: 是否影响系统性能
5. **安全性**: 是否存在安全风险

#### 审查流程
1. 至少需要一名核心贡献者审查
2. 所有CI检查必须通过
3. 解决审查意见后重新提交
4. 审查通过后由维护者合并

## 架构设计

### 适配器模式

skill-hub 使用适配器模式支持不同的AI工具：

```go
// 适配器接口
type Adapter interface {
    Supports(target string) bool
    Apply(skill *spec.Skill, vars map[string]string) error
    Remove(skillID string) error
    Extract(skillID string) (*spec.Skill, error)
}

// 具体适配器实现
type CursorAdapter struct { /* ... */ }
type ClaudeAdapter struct { /* ... */ }
type OpenCodeAdapter struct { /* ... */ }
```

### 技能引擎

技能引擎负责加载、解析和管理技能：

```go
type SkillManager struct {
    skillsDir string
}

func (m *SkillManager) LoadSkill(skillID string) (*spec.Skill, error)
func (m *SkillManager) GetSkillPrompt(skillID string) (string, error)
func (m *SkillManager) LoadAllSkills() ([]*spec.Skill, error)
```

### 状态管理

状态管理跟踪项目中的技能启用状态：

```go
type StateManager struct {
    projectPath string
}

func (m *StateManager) EnableSkill(skillID string, vars map[string]string) error
func (m *StateManager) DisableSkill(skillID string) error
func (m *StateManager) GetEnabledSkills() (map[string]SkillVars, error)
```

## 扩展开发

### 添加新适配器

#### 1. 创建适配器包
```bash
mkdir internal/adapter/newtool
touch internal/adapter/newtool/adapter.go
touch internal/adapter/newtool/adapter_test.go
```

#### 2. 实现适配器接口
```go
package newtool

import (
    "skill-hub/internal/adapter"
    "skill-hub/pkg/spec"
)

type NewToolAdapter struct {
    // 适配器实现
}

func (a *NewToolAdapter) Supports(target string) bool {
    return target == "newtool"
}

func (a *NewToolAdapter) Apply(skill *spec.Skill, vars map[string]string) error {
    // 实现应用逻辑
}
```

#### 3. 注册适配器
```go
// 在 adapter.go 中注册
func init() {
    adapter.Register("newtool", func() adapter.Adapter {
        return &NewToolAdapter{}
    })
}
```

#### 4. 添加测试
```go
func TestNewToolAdapter(t *testing.T) {
    // 测试适配器功能
}
```

### 添加新命令

#### 1. 创建命令文件
```bash
touch internal/cli/newcmd.go
touch internal/cli/newcmd_test.go
```

#### 2. 实现命令逻辑
```go
package cli

import (
    "github.com/spf13/cobra"
)

func NewNewCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "newcmd",
        Short: "新命令描述",
        RunE: func(cmd *cobra.Command, args []string) error {
            // 命令逻辑
        },
    }
    
    // 添加标志
    cmd.Flags().String("flag", "", "标志说明")
    
    return cmd
}
```

#### 3. 注册命令
```go
// 在 root.go 中注册
rootCmd.AddCommand(NewNewCmd())
```

## 性能优化

### 性能基准测试

```bash
# 运行基准测试
go test -bench=. -benchmem ./...

# 性能分析
go test -cpuprofile=cpu.out -memprofile=mem.out ./...
go tool pprof cpu.out
```

### 优化建议

1. **缓存技能加载结果**
2. **批量文件操作**
3. **减少内存分配**
4. **并发处理独立任务**

## 安全考虑

### 安全最佳实践

1. **输入验证**: 验证所有用户输入
2. **文件权限**: 设置适当的文件权限
3. **路径遍历**: 防止路径遍历攻击
4. **敏感信息**: 不记录敏感信息

### 安全测试

```bash
# 运行安全检查
go vet ./...
gosec ./...

# 依赖安全检查
go list -m all | grep -E "(vulnerability|CVE)"
```

## 故障排除

### 常见开发问题

#### 1. 测试失败
```bash
# 清理测试缓存
go clean -testcache

# 运行单个测试
go test -v -run TestSpecificFunction
```

#### 2. 构建失败
```bash
# 清理构建缓存
go clean -cache

# 更新依赖
go mod tidy
go mod download
```

#### 3. 依赖冲突
```bash
# 查看依赖图
go mod graph

# 更新到最新版本
go get -u ./...
```

### 调试技巧

```bash
# 使用delve调试
dlv debug ./cmd/skill-hub

# 添加调试日志
import "log"
log.Printf("Debug: %v", value)

# 使用pprof分析
import _ "net/http/pprof"
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

## 资源链接

- [Go官方文档](https://golang.org/doc/)
- [Cobra CLI框架](https://github.com/spf13/cobra)
- [GitHub Actions文档](https://docs.github.com/en/actions)
- [语义化版本控制](https://semver.org/)

## 获取帮助

- 查看 [GitHub Issues](https://github.com/muidea/skill-hub/issues)
- 加入讨论
- 阅读 [INSTALLATION.md](INSTALLATION.md) 获取使用指南
- 返回主文档 [README.md](README.md)
# Skill Hub 多Git仓库功能设计文档 (v1.0)

## 1. 概述

### 1.1 目标
为 skill-hub 添加多Git仓库支持，允许用户：
1. 从多个Git仓库中搜索、选择和应用技能
2. 使用主仓库作为存档库，将其他仓库的技能归档到主仓库
3. 保持现有工作流程不变，通过扩展现有命令实现功能

### 1.2 核心原则
- **无单独存档命令**：通过现有的 `feedback` 命令实现存档功能
- **统一存储结构**：所有仓库存储在 `~/.skill-hub/repositories/` 目录下
- **无数据迁移**：不迁移现有数据，保持向后兼容
- **保持现有工作流**：不改变项目工作空间管理（`.agents/skills/`）
- **冲突解决**：用户选择机制

## 2. 架构设计

### 2.1 存储结构
```
~/.skill-hub/
├── config.yaml                    # 扩展配置（YAML格式）
├── registry.json                  # 扩展索引（跨仓库技能索引）
├── state.json                     # 用户状态（保持不变）
└── repositories/                  # 所有仓库存储目录
    ├── main/                      # 主仓库（原 ~/.skill-hub/repo/）
    │   ├── .git/
    │   └── skills/
    ├── community/                 # 社区仓库示例
    │   ├── .git/
    │   └── skills/
    └── official/                  # 官方仓库示例
        ├── .git/
        └── skills/
```

### 2.2 配置扩展

#### 2.2.1 config.yaml 扩展
```yaml
# ~/.skill-hub/config.yaml
repo_path: "~/.skill-hub/repositories/main"  # 主仓库路径
claude_config_path: "~/.claude/config.json"
cursor_config_path: "~/.cursor/rules"
default_tool: "open_code"
git_remote_url: "git@github.com:muidea/skills-repo.git"
git_token: ""
git_branch: "master"

# 多仓库配置扩展
multi_repo:
  enabled: true
  default_repo: "main"          # 默认仓库（同时作为归档仓库）
  repositories:
    main:
      name: "main"
      url: "git@github.com:muidea/skills-repo.git"
      branch: "master"
      enabled: true
      description: "主技能仓库（默认归档仓库）"
      type: "user"              # 类型：user, community, official
      is_archive: true          # 标记为归档仓库
    community:
      name: "community"
      url: "https://github.com/skill-hub-community/awesome-skills.git"
      branch: "main"
      enabled: true
      description: "社区技能集合"
      type: "community"
      is_archive: false         # 非归档仓库
    official:
      name: "official"
      url: "https://github.com/skill-hub/official-skills.git"
      branch: "main"
      enabled: true
      description: "官方技能库"
      type: "official"
      is_archive: false         # 非归档仓库
```

#### 2.2.2 registry.json 扩展
```json
{
  "version": "2.0.0",
  "skills": [
    {
      "id": "go-refactor",
      "name": "Go Refactor Pro",
      "version": "1.2.0",
      "author": "unknown",
      "description": "生产级 Go 架构师。专注于安全重构、重复代码合并、解耦抽象及现代特性迁移。",
      "tags": ["go", "refactor", "architecture"],
      "repository": "main",           # 新增：源仓库名称
      "repository_path": "skills/go-refactor",  # 新增：仓库内路径
      "repository_commit": "abc123def456"       # 新增：仓库提交哈希
    },
    {
      "id": "demo",
      "name": "Demo Skill",
      "version": "1.0.0",
      "author": "community",
      "description": "演示技能示例",
      "tags": ["demo", "example"],
      "repository": "community",
      "repository_path": "skills/demo",
      "repository_commit": "def456ghi789"
    }
  ],
  "repositories": {
    "main": {
      "last_sync": "2025-02-17T10:00:00Z",
      "skill_count": 15,
      "enabled": true
    },
    "community": {
      "last_sync": "2025-02-17T09:30:00Z",
      "skill_count": 8,
      "enabled": true
    }
  }
}
```

#### 2.2.3 Go 数据结构

##### 2.2.3.1 扩展 spec.Skill
```go
// pkg/spec/skill.go 扩展
type Skill struct {
    ID               string        `yaml:"id" json:"id"`
    Name             string        `yaml:"name" json:"name"`
    Version          string        `yaml:"version" json:"version"`
    Author           string        `yaml:"author" json:"author"`
    Description      string        `yaml:"description" json:"description"`
    Tags             []string      `yaml:"tags" json:"tags"`
    Compatibility    string        `yaml:"compatibility,omitempty" json:"compatibility,omitempty"`
    Variables        []Variable    `yaml:"variables" json:"variables"`
    Dependencies     []string      `yaml:"dependencies" json:"dependencies"`
    Claude           *ClaudeConfig `yaml:"claude,omitempty" json:"claude,omitempty"`
    
    // 多仓库扩展字段
    Repository       string        `yaml:"repository,omitempty" json:"repository,omitempty"`           // 源仓库名称
    RepositoryPath   string        `yaml:"repository_path,omitempty" json:"repository_path,omitempty"` // 仓库内路径
    RepositoryCommit string        `yaml:"repository_commit,omitempty" json:"repository_commit,omitempty"` // 仓库提交哈希
}

// SkillMetadata 也需要相应扩展
type SkillMetadata struct {
    ID               string   `json:"id"`
    Name             string   `json:"name"`
    Version          string   `json:"version"`
    Author           string   `json:"author"`
    Description      string   `json:"description"`
    Tags             []string `json:"tags"`
    Compatibility    string   `json:"compatibility,omitempty"`
    Repository       string   `json:"repository,omitempty"`           // 源仓库名称
    RepositoryPath   string   `json:"repository_path,omitempty"`      // 仓库内路径
    RepositoryCommit string   `json:"repository_commit,omitempty"`    // 仓库提交哈希
}
```

##### 2.2.3.2 配置结构
```go
// internal/config/config.go 扩展
type RepositoryConfig struct {
    Name        string `yaml:"name" json:"name"`                // 仓库名称
    URL         string `yaml:"url" json:"url"`                  // Git远程URL
    Branch      string `yaml:"branch" json:"branch"`            // 默认分支
    Enabled     bool   `yaml:"enabled" json:"enabled"`          // 是否启用
    Description string `yaml:"description" json:"description"`  // 描述
    Type        string `yaml:"type" json:"type"`                // 类型：user/community/official
    IsArchive   bool   `yaml:"is_archive" json:"is_archive"`    // 是否为归档仓库
    LastSync    string `yaml:"last_sync,omitempty" json:"last_sync,omitempty"`
}

type MultiRepoConfig struct {
    Enabled      bool                       `yaml:"enabled" json:"enabled"`
    DefaultRepo  string                     `yaml:"default_repo" json:"default_repo"`  // 默认仓库（同时是归档仓库）
    Repositories map[string]RepositoryConfig `yaml:"repositories" json:"repositories"`
}

// 全局配置结构
type Config struct {
    RepoPath          string         `yaml:"repo_path" json:"repo_path"`
    ClaudeConfigPath  string         `yaml:"claude_config_path" json:"claude_config_path"`
    CursorConfigPath  string         `yaml:"cursor_config_path" json:"cursor_config_path"`
    DefaultTool       string         `yaml:"default_tool" json:"default_tool"`
    GitRemoteURL      string         `yaml:"git_remote_url" json:"git_remote_url"`
    GitToken          string         `yaml:"git_token" json:"git_token"`
    GitBranch         string         `yaml:"git_branch" json:"git_branch"`
    MultiRepo         *MultiRepoConfig `yaml:"multi_repo,omitempty" json:"multi_repo,omitempty"`
}

// 辅助函数：获取归档仓库
func (c *Config) GetArchiveRepository() (*RepositoryConfig, error) {
    if c.MultiRepo == nil || !c.MultiRepo.Enabled {
        // 单仓库模式，主仓库即为归档仓库
        return &RepositoryConfig{
            Name:        "main",
            URL:         c.GitRemoteURL,
            Branch:      c.GitBranch,
            Enabled:     true,
            Description: "主技能仓库",
            Type:        "user",
            IsArchive:   true,
        }, nil
    }
    
    // 多仓库模式，默认仓库即为归档仓库
    repo, exists := c.MultiRepo.Repositories[c.MultiRepo.DefaultRepo]
    if !exists {
        return nil, fmt.Errorf("默认仓库 '%s' 不存在", c.MultiRepo.DefaultRepo)
    }
    
    // 确保默认仓库标记为归档仓库
    repo.IsArchive = true
    return &repo, nil
}

// 辅助函数：检查是否为归档仓库
func (rc *RepositoryConfig) IsArchiveRepo() bool {
    return rc.IsArchive
}
```

### 2.3 数据模型更新

#### 2.3.1 冲突解决结构
```go
// pkg/spec/skill.go 扩展
type ConflictResolution struct {
    SkillID     string   `json:"skill_id"`
    Repository  string   `json:"selected_repo"`  // 用户选择的仓库
    Timestamp   string   `json:"timestamp"`
    UserChoice  bool     `json:"user_choice"`    // 是否为用户选择
}

// 冲突检测结果
type Conflict struct {
    SkillID    string          `json:"skill_id"`
    SkillName  string          `json:"skill_name"`
    Repositories []ConflictRepo `json:"repositories"`  // 包含此技能的仓库列表
}

type ConflictRepo struct {
    Repository string `json:"repository"`  // 仓库名称
    Version    string `json:"version"`     // 技能版本
    Commit     string `json:"commit"`      // 提交哈希
}
```

#### 2.3.2 冲突解决逻辑
- **无优先级**：仓库间平等，无优先级设置
- **用户选择**：冲突时由用户选择使用哪个仓库的技能
- **历史记录**：记录用户选择，避免重复询问
- **默认行为**：按仓库名称字母顺序排列供用户选择

## 3. 功能设计

### 3.1 仓库管理命令
```bash
# 添加仓库
skill-hub repo add <name> <url> [--branch <branch>]

# 列出仓库
skill-hub repo list

# 删除仓库
skill-hub repo remove <name>

# 同步仓库
skill-hub repo sync [<name>] [--all]

# 启用/禁用仓库
skill-hub repo enable <name>
skill-hub repo disable <name>

# 设置默认仓库
skill-hub repo default <name>
```

### 3.2 技能命令扩展

#### 3.2.1 `skill-hub use` 命令
```bash
# 基本用法（从所有仓库搜索）
skill-hub use <skill-id>

# 指定仓库
skill-hub use <skill-id> --repo <repo-name>

# 显示来源信息
skill-hub use <skill-id> --show-source

# 冲突时提示选择
skill-hub use <skill-id> --choose
```

#### 3.2.2 `skill-hub list` 命令
```bash
# 列出所有技能（显示来源）
skill-hub list [--repo <repo-name>] [--all]

# 输出格式
# ID           Name                    Repository    Version
# go-refactor  Go Refactor Pro        main          abc123
# demo         Demo Skill              community     def456
```

#### 3.2.3 `skill-hub search` 命令
```bash
# 搜索所有仓库
skill-hub search <keyword> [--repo <repo-name>]

# 搜索结果显示来源
```

#### 3.2.4 `skill-hub apply` 命令
- 自动处理多仓库技能
- 保持现有工作空间结构不变
- 在 `.agents/skills/` 中创建技能时记录来源信息

#### 多仓库扩展逻辑：
1. **技能来源判断**：
   - 如果技能来自默认仓库（归档仓库），直接同步更新
   - 如果技能来自其他仓库（如 `community`, `official`），在默认仓库进行新增或修改

2. **跨仓库存档策略**：
   - **统一归档**：所有修改都归档到默认仓库
   - **明确提示**：清楚显示技能来源和归档操作
   - **自动处理**：无需用户选择，自动在默认仓库进行新增/修改

3. **跨仓库存档流程**：
   ```
   用户执行 skill-hub feedback <skill-id>
       ↓
   检查技能来源
       ↓
   如果来源是默认仓库 → 正常同步更新到默认仓库
       ↓
   如果来源是其他仓库 → 在默认仓库进行新增或修改
       ↓
   更新技能来源信息为默认仓库
   ```

4. **交互提示示例**：
   ```
   技能 "python-utils" 来自 community 仓库
   默认仓库（归档仓库）是：main
   
   检测到技能修改，将归档到默认仓库 main
   确认归档? [y/N]: y
   
    归档成功! 技能已保存到默认仓库 main
    项目状态已更新：技能来源变更为 main
    ```

### 3.3 冲突处理机制

#### 3.3.1 冲突检测（use 命令）
当使用 `skill-hub use <skill-id>` 时：
1. 搜索所有启用的仓库，查找指定ID的技能
2. 如果只有一个仓库包含此技能，直接使用
3. 如果多个仓库包含同名技能，提示用户选择

#### 3.3.2 用户选择流程
```
用户执行 skill-hub use <skill-id>
    ↓
搜索所有仓库
    ↓
发现多个仓库包含此技能 → 显示仓库列表供用户选择
    ↓
用户选择特定仓库
    ↓
使用选定仓库的技能
    ↓
记录用户选择（可选）
```

#### 3.3.3 选择提示示例
```
发现多个仓库包含技能 "go-refactor":
  1) main 仓库 (v1.2.0)
  2) community 仓库 (v1.1.5)
  3) official 仓库 (v1.0.0)

请选择要使用的仓库 [1/2/3]: 1

已选择使用 main 仓库的 "go-refactor" 技能
```

#### 3.3.4 feedback 命令的简化处理
`feedback` 命令的冲突处理更简单：
```
用户执行 skill-hub feedback <skill-id>
    ↓
检查默认仓库中是否存在此技能
    ↓
如果不存在 → 在默认仓库新增技能
    ↓
如果存在 → 在默认仓库覆盖更新技能
    ↓
更新技能来源信息为默认仓库
```

**关键原则**：
- `use` 命令：用户选择使用哪个仓库的技能
- `feedback` 命令：只关心默认仓库，不存在则新增，存在则覆盖

## 4. 工作流程

### 4.1 技能应用流程
```
用户执行 skill-hub use <skill-id>
    ↓
搜索所有启用的仓库
    ↓
检查结果：
   - 只有一个仓库包含 → 直接使用
   - 多个仓库包含 → 提示用户选择仓库
    ↓
用户选择特定仓库（如需要）
    ↓
复制选定仓库的技能文件到项目工作空间（.agents/skills/）
    ↓
记录技能来源信息（仓库、路径、提交）到项目状态
    ↓
用户执行 skill-hub apply
    ↓
技能应用到目标工具
```

### 4.2 反馈/存档工作流程
```
用户修改技能并希望保存
    ↓
执行 skill-hub feedback <skill-id>
    ↓
检查默认仓库中是否存在此技能
    ↓
判断操作：
   - 默认仓库中不存在 → 在默认仓库新增技能
   - 默认仓库中存在 → 在默认仓库覆盖更新技能
    ↓
显示确认提示（显示技能来源和操作类型）
    ↓
执行操作：在默认仓库新增或覆盖更新
    ↓
更新技能来源信息为默认仓库
    ↓
更新 registry.json 和项目状态
    ↓
完成反馈
```

### 4.3 同步流程
```
用户执行 skill-hub repo sync --all
    ↓
遍历所有启用的仓库
    ↓
执行 git pull
    ↓
更新最后同步时间
    ↓
重建技能索引
```

## 5. 实现细节

### 5.1 新模块结构
```
internal/multirepo/
├── manager.go          # 多仓库管理器
├── resolver.go         # 冲突解决器
├── indexer.go          # 跨仓库索引器
├── feedback.go         # 反馈处理器（扩展现有功能）
└── registry.go         # 注册表管理器

internal/cli/
├── repo.go             # 仓库管理命令
└── 现有命令扩展（use.go, list.go, search.go, feedback.go等）
```

### 5.4 feedback 命令扩展实现
```go
// internal/cli/feedback.go 扩展
func runFeedbackWithMultiRepo(skillID string) error {
    // 1. 获取项目状态和技能来源
    stateManager, err := state.NewStateManager()
    if err != nil {
        return fmt.Errorf("初始化状态管理器失败: %w", err)
    }
    
    projectState, err := stateManager.GetProjectState(cwd)
    if err != nil {
        return fmt.Errorf("获取项目状态失败: %w", err)
    }
    
    // 2. 获取技能来源信息
    skillSource, exists := projectState.SkillSources[skillID]
    if !exists {
        // 回退到单仓库逻辑
        return runFeedbackLegacy(skillID)
    }
    
    // 3. 初始化多仓库管理器
    multiRepoManager, err := multirepo.NewManager()
    if err != nil {
        return fmt.Errorf("初始化多仓库管理器失败: %w", err)
    }
    
    // 4. 检查来源仓库
    sourceRepo, err := multiRepoManager.GetRepository(skillSource.Repository)
    if err != nil {
        return fmt.Errorf("获取来源仓库信息失败: %w", err)
    }
    
    // 5. 判断处理逻辑
    if sourceRepo.Name == "main" {
        // 来源是主仓库，直接同步
        return syncToRepository(skillID, "main", projectSkillDir)
    } else {
        // 来源是其他仓库，提供选择
        return handleExternalRepoFeedback(skillID, sourceRepo, projectSkillDir)
    }
}

func handleFeedbackWithMultiRepo(skillID string, projectSkillDir string) error {
    // 获取默认仓库（归档仓库）
    config, err := config.LoadConfig()
    if err != nil {
        return fmt.Errorf("加载配置失败: %w", err)
    }
    
    defaultRepo, err := config.GetArchiveRepository()
    if err != nil {
        return fmt.Errorf("获取默认仓库失败: %w", err)
    }
    
    // 检查默认仓库中是否存在此技能
    skillExists, err := checkSkillInRepository(skillID, defaultRepo.Name)
    if err != nil {
        return fmt.Errorf("检查技能存在性失败: %w", err)
    }
    
    // 显示操作信息
    if skillExists {
        fmt.Printf("技能 '%s' 在默认仓库 %s 中已存在，将覆盖更新\n", skillID, defaultRepo.Name)
    } else {
        fmt.Printf("技能 '%s' 在默认仓库 %s 中不存在，将新增\n", skillID, defaultRepo.Name)
    }
    
    // 显示修改内容
    changes, err := getSkillChanges(skillID, projectSkillDir, "")
    if err != nil {
        return err
    }
    
    if len(changes) == 0 {
        fmt.Println("✅ 技能内容未修改")
        return nil
    }
    
    fmt.Println("检测到以下修改：")
    for _, change := range changes {
        fmt.Printf("  - %s\n", change)
    }
    
    // 确认操作
    operation := "覆盖更新"
    if !skillExists {
        operation = "新增"
    }
    
    fmt.Printf("\n将在默认仓库 %s 中%s技能\n", defaultRepo.Name, operation)
    if !promptYesNo("确认执行?") {
        fmt.Println("操作已取消")
        return nil
    }
    
    // 在默认仓库执行新增或覆盖更新
    return archiveToDefaultRepository(skillID, defaultRepo.Name, projectSkillDir, skillExists)
}

func archiveToDefaultRepository(skillID, defaultRepo string, projectSkillDir string, skillExists bool) error {
    // 1. 复制技能文件到默认仓库
    defaultRepoPath := getRepositoryPath(defaultRepo)
    targetSkillDir := filepath.Join(defaultRepoPath, "skills", skillID)
    
    // 如果技能已存在，先备份原文件（可选）
    if skillExists {
        backupDir := filepath.Join(defaultRepoPath, "skills", skillID+".backup")
        if err := backupSkillDirectory(targetSkillDir, backupDir); err != nil {
            fmt.Printf("警告：备份原技能失败: %v\n", err)
        }
    }
    
    // 复制/覆盖技能文件
    if err := copySkillDirectory(projectSkillDir, targetSkillDir); err != nil {
        return fmt.Errorf("复制技能文件失败: %w", err)
    }
    
    // 2. 更新 registry.json
    registryManager, err := multirepo.NewRegistryManager()
    if err != nil {
        return err
    }
    
    // 更新技能信息（使用扩展的 spec.Skill 结构）
    skill := &spec.Skill{
        ID:               skillID,
        Repository:       defaultRepo,
        RepositoryPath:   fmt.Sprintf("skills/%s", skillID),
        RepositoryCommit: getCurrentCommit(defaultRepo),
        // 其他字段从项目工作区读取...
    }
    
    if skillExists {
        if err := registryManager.UpdateSkill(skill); err != nil {
            return fmt.Errorf("更新技能信息失败: %w", err)
        }
    } else {
        if err := registryManager.AddSkill(skill); err != nil {
            return fmt.Errorf("添加技能信息失败: %w", err)
        }
    }
    
    // 3. 更新项目状态
    stateManager, err := state.NewStateManager()
    if err != nil {
        return err
    }
    
    if err := stateManager.UpdateSkillSource(cwd, skillID, defaultRepo); err != nil {
        return fmt.Errorf("更新项目状态失败: %w", err)
    }
    
    // 4. 显示操作结果
    operation := "覆盖更新"
    if !skillExists {
        operation = "新增"
    }
    fmt.Printf("✅ 技能 '%s' 已在默认仓库 %s 中%s完成\n", skillID, defaultRepo, operation)
    return nil
}
    
    // 2. 更新 registry.json
    registryManager, err := multirepo.NewRegistryManager()
    if err != nil {
        return err
    }
    
    // 更新技能来源信息
    newSource := SkillSource{
        Repository: archiveRepo,
        Path:       fmt.Sprintf("skills/%s", skillID),
        Commit:     getCurrentCommit(archiveRepo),

    }
    
    if err := registryManager.UpdateSkillSource(skillID, newSource); err != nil {
        return fmt.Errorf("更新注册表失败: %w", err)
    }
    
    // 3. 更新项目状态
    stateManager, err := state.NewStateManager()
    if err != nil {
        return err
    }
    
    if err := stateManager.UpdateSkillSource(cwd, skillID, newSource); err != nil {
        return fmt.Errorf("更新项目状态失败: %w", err)
    }
    
    // 4. 可选：从原仓库删除（需要用户确认）
    if promptYesNo("是否从原仓库 %s 中删除此技能?", sourceRepo) {
        if err := removeFromRepository(skillID, sourceRepo); err != nil {
            fmt.Printf("警告：从原仓库删除失败: %v\n", err)
        }
    }
    
    fmt.Printf("✅ 技能 '%s' 已成功归档到默认仓库 %s\n", skillID, archiveRepo)
    return nil
}
```

### 5.2 关键接口
```go
// 多仓库管理器接口
type MultiRepoManager interface {
    // 仓库管理
    ListRepositories() ([]RepositoryConfig, error)
    GetRepository(name string) (*RepositoryConfig, error)
    AddRepository(config RepositoryConfig) error
    RemoveRepository(name string) error
    SyncRepository(name string) error
    EnableRepository(name string) error
    DisableRepository(name string) error
    
    // 技能管理
    FindSkill(skillID string) ([]SkillWithSource, error)
    SearchSkills(query string, repoFilter string) ([]SkillWithSource, error)
    ListSkills(repoFilter string) ([]SkillWithSource, error)
    
    // 冲突解决
    ResolveConflict(skillID string, userChoice string) (*SkillWithSource, error)
    GetConflictResolution(skillID string) (*ConflictResolution, error)
    
    // 反馈/存档功能
    FeedbackSkill(skillID, projectPath string, targetRepo string) error
    GetSkillSource(skillID string) (*SkillSource, error)
    ArchiveToMain(skillID, sourceRepo string) error
    
    // 索引管理
    RebuildIndex() error
    GetRegistry() (*Registry, error)
}
```

### 5.3 状态管理扩展
```go
// internal/state/manager.go 扩展
type ProjectState struct {
    // 现有字段...
    ProjectID    string                       `json:"project_id"`
    Skills       map[string]WorkspaceSkill    `json:"skills"`
    CreatedAt    string                       `json:"created_at"`
    UpdatedAt    string                       `json:"updated_at"`
    
    // 多仓库扩展字段
    SkillSources map[string]SkillSource       `json:"skill_sources"` // 技能来源映射
    Conflicts    []ConflictResolution         `json:"conflicts"`     // 冲突解决记录
    ActiveRepo   string                       `json:"active_repo"`   // 当前活动仓库
}

type WorkspaceSkill struct {
    SkillID      string      `json:"skill_id"`
    LocalPath    string      `json:"local_path"`    // 项目工作空间中的路径
    Source       SkillSource `json:"source"`        // 技能来源信息
    Applied      bool        `json:"applied"`       // 是否已应用
    LastApplied  string      `json:"last_applied"`  // 最后应用时间
    Modified     bool        `json:"modified"`      // 是否被修改
    Archived     bool        `json:"archived"`      // 是否已存档到主仓库
}

// state.json 结构
type GlobalState struct {
    CurrentProject string            `json:"current_project"`
    Projects       map[string]string `json:"projects"` // project_id -> state_file_path
    
    // 多仓库全局状态
    LastSync       map[string]string `json:"last_sync"`       // 仓库最后同步时间
    UserPreferences map[string]any    `json:"user_preferences"` // 用户偏好设置
}
```

## 6. 用户界面设计

### 6.1 交互提示
```
# 冲突提示
发现多个版本的技能 "go-refactor":
  1) main (v1.2.0)
  2) community (v1.1.5)
请选择要使用的版本 [1/2] (默认: 1):
```

### 6.2 输出格式
```
# skill-hub list --all
仓库: main
  go-refactor    Go Refactor Pro        v1.2.0
  demo           Demo Skill             v1.0.0

仓库: community
  go-refactor    Go Refactor Pro        v1.1.5
  python-utils   Python Utilities       v2.3.1

总计: 4个技能 (go-refactor 在 2个仓库中存在)
```

### 6.3 反馈确认（多仓库场景）

#### 场景1：技能在默认仓库中不存在（新增）
```
技能 "new-skill" 在默认仓库 main 中不存在，将新增

检测到以下修改：
  - SKILL.md: 新增技能描述
  - examples/: 新增示例文件

将在默认仓库 main 中新增技能
确认执行? [y/N]: y

新增成功! 技能 "new-skill" 已在默认仓库 main 中新增完成
```

#### 场景2：技能在默认仓库中已存在（覆盖更新）
```
技能 "python-utils" 在默认仓库 main 中已存在，将覆盖更新

检测到以下修改：
  - SKILL.md: 更新了描述
  - examples/: 修改了示例文件

将在默认仓库 main 中覆盖更新技能
确认执行? [y/N]: y

覆盖更新成功! 技能 "python-utils" 已在默认仓库 main 中更新完成
```

## 7. 错误处理

### 7.1 错误场景
1. **仓库不可访问**：网络问题或权限不足
2. **冲突无法解决**：用户未选择且无默认
3. **存档失败**：目标仓库写入权限问题
4. **同步冲突**：本地修改与远程冲突

### 7.2 错误恢复
- 仓库同步失败时保持上次可用状态
- 冲突解决失败时中止操作
- 反馈/存档失败时保留原技能不变
- 跨仓库操作失败时提供回滚选项
- 提供详细的错误信息和恢复建议

### 7.3 反馈命令的特殊错误处理
```go
// feedback 命令的多仓库错误处理
func handleFeedbackError(skillID, sourceRepo, targetRepo string, err error) error {
    switch {
    case errors.Is(err, ErrRepositoryNotFound):
        return fmt.Errorf("仓库 '%s' 不存在，请先添加该仓库", targetRepo)
    case errors.Is(err, ErrRepositoryDisabled):
        return fmt.Errorf("仓库 '%s' 已禁用，请先启用", targetRepo)
    case errors.Is(err, ErrPermissionDenied):
        return fmt.Errorf("无权限写入仓库 '%s'", targetRepo)
    case errors.Is(err, ErrSkillNotFoundInSource):
        return fmt.Errorf("技能 '%s' 在源仓库 '%s' 中不存在", skillID, sourceRepo)
    default:
        return fmt.Errorf("同步失败: %v", err)
    }
}
```

## 8. 测试策略

### 8.1 单元测试
- 多仓库管理器功能测试
- 冲突解决逻辑测试
- 存档功能测试
- 配置管理测试

### 8.2 集成测试
- 端到端技能应用流程
- 多仓库搜索和选择
- 存档工作流程
- 冲突解决交互

### 8.3 性能测试
- 大量仓库下的搜索性能
- 冲突检测性能
- 同步操作性能

## 9. 部署与迁移

### 9.1 向后兼容

#### 9.1.1 目录结构迁移
```
现有结构 (~/.skill-hub/)           →           新结构 (~/.skill-hub/)
├── config.yaml                           ├── config.yaml (更新)
├── registry.json                         ├── registry.json (v2.0.0)
├── repo/                                 ├── state.json (保持不变)
└── state.json                            └── repositories/
                                               └── main/ (原repo目录内容)
```

#### 9.1.2 配置文件迁移
1. **config.yaml**:
   - 更新 `repo_path: "~/.skill-hub/repositories/main"`
   - 添加 `multi_repo` 配置节
   - 保留原有配置项

2. **registry.json**:
   - 版本升级到 `"2.0.0"`
   - 为每个技能添加 `source` 字段
   - 添加 `repositories` 节记录仓库状态
   - 添加 `conflict_resolutions` 节

3. **state.json**:
   - 保持不变，运行时自动扩展字段

#### 9.1.3 数据迁移保证
- 无数据丢失：所有技能文件完整迁移
- 配置兼容：原有配置项保持不变
- 状态保持：用户项目状态不受影响
- 索引重建：自动重建跨仓库技能索引

### 9.2 配置迁移实现

#### 9.2.1 迁移步骤
```go
func migrateLegacyToMultiRepo() error {
    // 步骤1: 检查当前版本
    if isMultiRepoEnabled() {
        return nil // 已迁移
    }
    
    // 步骤2: 备份原有配置
    backupPath := "~/.skill-hub/backup-" + time.Now().Format("20060102-150405")
    if err := backupExistingConfig(backupPath); err != nil {
        return fmt.Errorf("备份失败: %v", err)
    }
    
    // 步骤3: 创建 repositories 目录
    reposDir := "~/.skill-hub/repositories"
    if err := os.MkdirAll(reposDir, 0755); err != nil {
        return fmt.Errorf("创建目录失败: %v", err)
    }
    
    // 步骤4: 移动主仓库
    oldRepoPath := "~/.skill-hub/repo"
    newRepoPath := "~/.skill-hub/repositories/main"
    if err := moveRepository(oldRepoPath, newRepoPath); err != nil {
        return fmt.Errorf("移动仓库失败: %v", err)
    }
    
    // 步骤5: 更新 config.yaml
    if err := updateConfigFile(); err != nil {
        return fmt.Errorf("更新配置失败: %v", err)
    }
    
    // 步骤6: 更新 registry.json
    if err := updateRegistryFile(); err != nil {
        return fmt.Errorf("更新索引失败: %v", err)
    }
    
    // 步骤7: 记录迁移完成
    recordMigrationComplete()
    
    return nil
}
```

#### 9.2.2 关键迁移函数
```go
func updateConfigFile() error {
    // 读取现有配置
    config, err := loadConfig("~/.skill-hub/config.yaml")
    if err != nil {
        return err
    }
    
    // 更新 repo_path
    config.RepoPath = "~/.skill-hub/repositories/main"
    
    // 添加 multi_repo 配置
    config.MultiRepo = &MultiRepoConfig{
        Enabled:     true,
        DefaultRepo: "main",
        ArchiveRepo: "main",
        Repositories: map[string]RepositoryConfig{
            "main": {
                Name:        "main",
                URL:         config.GitRemoteURL,
                Branch:      config.GitBranch,

                Enabled:     true,
                Description: "主技能仓库",
                Type:        "user",
            },
        },
    }
    
    // 保存配置
    return saveConfig(config, "~/.skill-hub/config.yaml")
}

func updateRegistryFile() error {
    // 读取现有 registry.json
    registry, err := loadRegistry("~/.skill-hub/registry.json")
    if err != nil {
        return err
    }
    
    // 升级版本
    registry.Version = "2.0.0"
    
    // 为每个技能添加 source 字段
    for i := range registry.Skills {
        skill := &registry.Skills[i]
        skill.Source = SkillSource{
            Repository: "main",
            Path:       fmt.Sprintf("skills/%s", skill.ID),
            Commit:     getLatestCommit("main", skill.ID),

        }
    }
    
    // 添加 repositories 节
    registry.Repositories = map[string]RepositoryStatus{
        "main": {
            LastSync:   time.Now().Format(time.RFC3339),
            SkillCount: len(registry.Skills),
            Enabled:    true,
        },
    }
    
    // 保存更新后的 registry.json
    return saveRegistry(registry, "~/.skill-hub/registry.json")
}
```

#### 9.2.3 回滚机制
```go
func rollbackMigration(backupPath string) error {
    // 恢复备份的配置
    // 恢复原目录结构
    // 删除新创建的目录
    // 恢复原有文件
    return nil
}
```

### 9.3 初始化与首次运行

#### 9.3.1 `init` 命令行为
`skill-hub init` 命令在设置 `git_url` 时，该Git仓库即为默认仓库（归档仓库）。

```bash
# 新用户初始化（设置git_url）
skill-hub init https://github.com/username/skills-repo.git

# 输出：
正在初始化Skill Hub工作区: ~/.skill-hub
将克隆远程仓库: https://github.com/username/skills-repo.git
✓ 目录已就绪: ~/.skill-hub
✓ 目录已就绪: ~/.skill-hub/repositories/main
✓ 创建配置文件: ~/.skill-hub/config.yaml
✓ 克隆远程仓库完成
✅ skill-hub 初始化完成！

# 生成的配置：
# - git_url 指定的仓库作为默认仓库（main）
# - 默认仓库自动标记为归档仓库 (is_archive: true)
# - 启用多仓库功能 (multi_repo.enabled: true)
```

#### 9.3.2 无git_url初始化
```bash
# 无git_url初始化（本地模式）
skill-hub init

# 输出：
正在初始化Skill Hub工作区: ~/.skill-hub
✓ 目录已就绪: ~/.skill-hub
✓ 目录已就绪: ~/.skill-hub/repositories/main
✓ 创建配置文件: ~/.skill-hub/config.yaml
✅ skill-hub 初始化完成！

# 生成的配置：
# - 创建空的本地仓库作为默认仓库（main）
# - 默认仓库自动标记为归档仓库
# - 启用多仓库功能
```

#### 9.3.3 现有用户迁移
```
检测到旧版本配置，正在迁移...
正在创建多仓库目录结构...
移动主仓库到 ~/.skill-hub/repositories/main/
更新配置文件...
重建技能索引...
迁移完成! 您的技能库已升级支持多仓库功能

添加新仓库: skill-hub repo add <name> <url>
查看可用仓库: skill-hub repo list
```

#### 9.3.4 init 命令的多仓库扩展实现
```go
// internal/cli/init.go 扩展
func runInitWithMultiRepo(args []string, target string) error {
    // ... 现有初始化逻辑 ...
    
    // 确定仓库名称（默认为"main"）
    repoName := "main"
    
    // 创建多仓库配置
    config := &Config{
        RepoPath:         filepath.Join(skillHubDir, "repositories", repoName),
        ClaudeConfigPath: "~/.claude/config.json",
        CursorConfigPath: "~/.cursor/rules",
        DefaultTool:      target,
        GitRemoteURL:     gitURL,
        GitToken:         "",
        GitBranch:        "master",
        MultiRepo: &MultiRepoConfig{
            Enabled:     true,
            DefaultRepo: repoName, // git_url 指定的仓库作为默认仓库
            Repositories: map[string]RepositoryConfig{
                repoName: {
                    Name:        repoName,
                    URL:         gitURL,
                    Branch:      "master",

                    Enabled:     true,
                    Description: "默认技能仓库",
                    Type:        "user",
                    IsArchive:   true, // 自动标记为归档仓库
                },
            },
        },
    }
    
    // 保存配置
    if err := saveConfig(config, configPath); err != nil {
        return fmt.Errorf("保存配置失败: %w", err)
    }
    
    // 创建仓库目录
    repoDir := filepath.Join(skillHubDir, "repositories", repoName)
    if err := os.MkdirAll(repoDir, 0755); err != nil {
        return fmt.Errorf("创建仓库目录失败: %w", err)
    }
    
    if gitURL != "" {
        // 克隆远程仓库
        fmt.Printf("正在克隆仓库到: %s\n", repoDir)
        if err := git.Clone(gitURL, repoDir); err != nil {
            return fmt.Errorf("克隆仓库失败: %w", err)
        }
        fmt.Println("✓ 克隆远程仓库完成")
    } else {
        // 初始化空仓库
        fmt.Printf("正在初始化空仓库: %s\n", repoDir)
        if err := git.Init(repoDir); err != nil {
            return fmt.Errorf("初始化仓库失败: %w", err)
        }
        fmt.Println("✓ 初始化空仓库完成")
    }
    
    fmt.Println("✅ skill-hub 多仓库模式初始化完成！")
    if gitURL != "" {
        fmt.Printf("默认仓库（归档仓库）: %s\n", gitURL)
    } else {
        fmt.Println("默认仓库（归档仓库）: 本地空仓库")
    }
    fmt.Println("使用 'skill-hub repo add' 添加更多仓库")
    
    return nil
}
```


## 10. 未来扩展

### 10.1 计划功能
1. **智能推荐**：基于使用习惯推荐仓库
2. **自动同步**：定时同步仓库
3. **仓库评分**：社区仓库质量评分
4. **依赖管理**：技能间依赖关系
5. **批量操作**：批量应用、存档

### 10.2 API扩展
- REST API 用于远程管理
- Webhook 支持
- 插件系统

## 11. 附录

### 11.1 配置示例

#### 11.1.1 config.yaml 示例
```yaml
# ~/.skill-hub/config.yaml
repo_path: "~/.skill-hub/repositories/main"
claude_config_path: "~/.claude/config.json"
cursor_config_path: "~/.cursor/rules"
default_tool: "open_code"
git_remote_url: "git@github.com:muidea/skills-repo.git"
git_token: ""
git_branch: "master"

multi_repo:
  enabled: true
  default_repo: "main"          # 默认仓库即为归档仓库
  repositories:
    main:
      name: "main"
      url: "git@github.com:muidea/skills-repo.git"
      branch: "master"
      enabled: true
      description: "默认技能仓库（归档仓库）"
      type: "user"
      is_archive: true          # 标记为归档仓库
    community:
      name: "community"
      url: "https://github.com/skill-hub-community/awesome-skills.git"
      branch: "main"
      enabled: true
      description: "社区技能集合"
      type: "community"
      is_archive: false         # 非归档仓库
    official:
      name: "official"
      url: "https://github.com/skill-hub/official-skills.git"
      branch: "main"
      enabled: true
      description: "官方技能库"
      type: "official"
      is_archive: false         # 非归档仓库
```

#### 11.1.2 registry.json 示例
```json
{
  "version": "2.0.0",
  "skills": [
    {
      "id": "go-refactor",
      "name": "Go Refactor Pro",
      "version": "1.2.0",
      "author": "unknown",
      "description": "生产级 Go 架构师。专注于安全重构、重复代码合并、解耦抽象及现代特性迁移。",
      "tags": ["go", "refactor", "architecture"],
      "repository": "main",
      "repository_path": "skills/go-refactor",
      "repository_commit": "abc123def456"
    },
    {
      "id": "demo",
      "name": "Demo Skill",
      "version": "1.0.0",
      "author": "community",
      "description": "演示技能示例",
      "tags": ["demo", "example"],
      "repository": "community",
      "repository_path": "skills/demo",
      "repository_commit": "def456ghi789"
    }
  ],
  "repositories": {
    "main": {
      "last_sync": "2025-02-17T10:00:00Z",
      "skill_count": 15,
      "enabled": true
    },
    "community": {
      "last_sync": "2025-02-17T09:30:00Z",
      "skill_count": 8,
      "enabled": true
    }
  },
  "conflict_resolutions": {
    "go-refactor": {
      "selected_repo": "main",
      "timestamp": "2025-02-17T10:30:00Z",
      "user_choice": true
    }
  }
}
```

### 11.2 命令参考

#### 11.2.1 初始化命令
```bash
# 初始化并设置默认仓库（归档仓库）
skill-hub init https://github.com/username/skills-repo.git

# 初始化本地空仓库
skill-hub init

# 指定目标环境
skill-hub init https://github.com/username/skills-repo.git --target open_code
```

#### 11.2.2 完整工作流示例
```bash
# 1. 初始化（设置默认仓库）
skill-hub init https://github.com/username/skills-repo.git

# 2. 添加社区仓库
skill-hub repo add community https://github.com/skill-hub-community/awesome-skills.git

# 3. 同步仓库
skill-hub repo sync --all

# 4. 搜索技能
skill-hub search "go refactor"

# 5. 使用技能（处理冲突）
skill-hub use go-refactor --choose

# 6. 应用技能
skill-hub apply

# 7. 修改技能后反馈（自动处理多仓库）
skill-hub feedback go-refactor

# 8. 查看技能来源
skill-hub list --show-source

# 9. 设置不同的默认仓库
skill-hub repo default team

# 10. 查看仓库状态
skill-hub repo list
```

#### 11.2.3 关键说明
- `init` 命令设置的 `git_url` 仓库即为默认仓库（归档仓库）
- 默认仓库自动标记为 `is_archive: true`
- 多仓库功能默认启用
- 反馈命令自动使用默认仓库作为归档目标

#### 多仓库使用和反馈场景示例：
```bash
# 场景1：使用技能（多个仓库存在同名技能）
skill-hub use go-refactor
# 输出：
# 发现多个仓库包含技能 "go-refactor":
#   1) main 仓库 (v1.2.0)
#   2) community 仓库 (v1.1.5)
# 请选择要使用的仓库 [1/2]: 1
# 已选择使用 main 仓库的 "go-refactor" 技能

# 场景2：反馈技能（默认仓库中不存在）
skill-hub feedback new-skill
# 输出：
# 技能 'new-skill' 在默认仓库 main 中不存在，将新增
# 检测到修改...
# 将在默认仓库 main 中新增技能
# 确认执行? [y/N]: y
# 新增成功!

# 场景3：反馈技能（默认仓库中已存在）
skill-hub feedback existing-skill
# 输出：
# 技能 'existing-skill' 在默认仓库 main 中已存在，将覆盖更新
# 检测到修改...
# 将在默认仓库 main 中覆盖更新技能
# 确认执行? [y/N]: y
# 覆盖更新成功!

# 场景4：查看差异（演习模式）
skill-hub feedback my-skill --dry-run

# 场景5：指定仓库使用技能
skill-hub use python-utils --repo community
# 输出：使用 community 仓库的 python-utils 技能
```

---

**文档版本**: v1.9  
**创建日期**: 2025-02-17  
**最后更新**: 2025-02-17  
**状态**: 设计完成，已调整冲突处理逻辑

## 更新记录

### v1.9 (2025-02-17)
- 调整冲突处理逻辑：
  - `use` 命令：多个仓库存在同名 skill 时，由用户选择使用哪个仓库的 skill
  - `feedback` 命令：只判断默认仓库是否存在，不存在则新增，存在则覆盖更新
  - 简化 `feedback` 处理逻辑，移除复杂的源仓库判断
  - 更新所有相关交互示例和实现代码

### v1.8 (2025-02-17)
- 移除所有 priority 相关内容：
  - 移除 `repo add` 命令的 `--priority` 参数
  - 移除配置中的 `priority` 字段
  - 移除输出显示中的优先级信息
  - 更新冲突解决逻辑，不再依赖优先级
  - 清理所有相关描述和示例

### v1.7 (2025-02-17)
- 更新数据结构：
  - 直接扩展 `spec.Skill` 结构，增加源仓库信息字段
  - 移除 `SkillWithSource` 等新结构定义
  - 仓库间没有 priority 优先级设置
  - 更新所有相关配置示例和数据结构定义

### v1.6 (2025-02-17)
- 更新 `feedback` 命令归档策略：
  - `feedback` 命令只归档至默认仓库
  - 如果技能来自其他仓库，则在默认仓库进行新增或修改
  - 移除用户选择逻辑，直接归档到默认仓库
  - 更新所有相关交互示例和实现代码

### v1.5 (2025-02-17)
- 更新 `feedback` 命令确认流程：
  - 当技能的源仓库不是默认仓库时，给出明确的确认提示
  - 显示技能来源和默认仓库信息
  - 提供清晰的用户选择选项
  - 更新所有相关交互示例和实现代码

### v1.4 (2025-02-17)
- 补充 `init` 命令说明：
  - 首次执行 `init` 命令时，设置的 `git_url` 仓库即为默认仓库（归档仓库）
  - 详细说明 `init` 命令的多仓库初始化行为
  - 添加 `init` 命令的扩展实现代码
  - 更新命令参考部分，包含完整的初始化工作流
  - 说明无 `git_url` 初始化时的本地仓库创建

### v1.3 (2025-02-17)
- 调整归档仓库策略：
  - 默认仓库即为归档仓库，移除单独的 `archive_repo` 配置
  - 添加 `is_archive` 字段标记仓库类型
  - 更新 `GetArchiveRepository()` 辅助函数
  - 调整反馈命令的默认行为：自动归档到默认仓库
  - 更新所有相关配置示例和实现代码

### v1.2 (2025-02-17)
- 根据 `feedback` 命令现状更新设计：
  - 移除 `--archive` 参数设计，使用现有 `feedback` 命令
  - 扩展 `feedback` 命令支持多仓库场景
  - 添加跨仓库存档交互流程
  - 详细设计反馈命令的多仓库实现
  - 更新错误处理和用户交互设计

### v1.1 (2025-02-17)
- 根据实际存储结构更新设计：
  - 确认 `~/.skill-hub/` 包含 `config.yaml`, `registry.json`, `state.json`, `repo/`
  - 更新存储结构为：`repositories/main/`, `repositories/community/`, `repositories/official/`
  - 详细设计配置迁移方案
  - 更新 registry.json v2.0.0 格式
  - 完善迁移实现细节和回滚机制

### v1.0 (2025-02-17)
- 初始版本设计完成
- 包含完整的多仓库功能设计
- 涵盖架构、功能、实现、测试等全部方面
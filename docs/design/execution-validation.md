# 执行验证配置设计文档

## 1. 概述

### 1.1 问题背景

不同代码库使用不同的代码质量工具：
- 格式化工具：`gofmt`、`prettier`、`black`、`rustfmt` 等
- Lint 工具：`golangci-lint`、`eslint`、`pylint`、`clippy` 等
- 测试框架：`go test`、`jest`、`pytest`、`cargo test` 等

每个项目可能有不同的：
- 工具选择和版本
- 配置文件位置和名称
- 执行命令和参数
- 是否启用某些检查

### 1.2 设计目标

- **零配置可用**：自动检测项目类型和工具配置，无需手动配置即可工作
- **灵活可定制**：支持仓库级配置覆盖，满足特殊需求
- **UI 可管理**：提供可视化界面配置和管理
- **可扩展**：支持添加新的工具和语言

### 1.3 混合方案策略

```
┌─────────────────────────────────────────────────────────────────────┐
│                        执行验证配置策略                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────┐                                                │
│  │  1. 检查仓库配置 │                                                │
│  └────────┬────────┘                                                │
│           │                                                         │
│      ┌────┴────┐                                                    │
│      │         │                                                    │
│   已配置     未配置                                                 │
│      │         │                                                    │
│      ▼         ▼                                                    │
│  使用配置   ┌─────────────────┐                                     │
│             │ 2. 自动检测项目 │                                     │
│             └────────┬────────┘                                     │
│                      │                                              │
│                ┌─────┴─────┐                                        │
│                │           │                                        │
│            检测成功    检测失败/禁用                                │
│                │           │                                        │
│                ▼           ▼                                        │
│           使用检测结果   使用全局默认                               │
│                                                                     │
│  ┌─────────────────┐                                                │
│  │  3. 执行验证    │                                                │
│  │  - 格式化代码   │                                                │
│  │  - Lint 检查    │                                                │
│  │  - 运行测试     │                                                │
│  └─────────────────┘                                                │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. 配置结构设计

### 2.1 配置层级

| 层级 | 来源 | 优先级 | 说明 |
|------|------|--------|------|
| 1 | 仓库级配置 | 最高 | 用户在 UI 中为特定仓库配置 |
| 2 | 自动检测结果 | 中 | 系统自动检测项目工具配置 |
| 3 | 全局默认配置 | 最低 | 系统默认配置 |

### 2.2 执行验证配置结构

```go
// ExecutionValidationConfig 执行验证配置
type ExecutionValidationConfig struct {
    // 是否启用执行验证
    Enabled bool `json:"enabled"`

    // 格式化配置
    Formatting FormattingConfig `json:"formatting"`

    // Lint 配置
    Linting LintingConfig `json:"linting"`

    // 测试配置
    Testing TestingConfig `json:"testing"`

    // 自动检测配置
    AutoDetection AutoDetectionConfig `json:"autoDetection"`
}

// FormattingConfig 格式化配置
type FormattingConfig struct {
    // 是否启用格式化
    Enabled bool `json:"enabled"`

    // 格式化工具列表（按顺序执行）
    Tools []ToolCommand `json:"tools"`

    // 格式化失败时的处理策略
    FailureStrategy FailureStrategy `json:"failureStrategy"`
}

// LintingConfig Lint 检查配置
type LintingConfig struct {
    // 是否启用 Lint
    Enabled bool `json:"enabled"`

    // Lint 工具列表（按顺序执行）
    Tools []ToolCommand `json:"tools"`

    // Lint 失败时的处理策略
    FailureStrategy FailureStrategy `json:"failureStrategy"`

    // 是否自动修复 Lint 问题
    AutoFix bool `json:"autoFix"`
}

// TestingConfig 测试配置
type TestingConfig struct {
    // 是否启用测试
    Enabled bool `json:"enabled"`

    // 测试命令
    Command ToolCommand `json:"command"`

    // 测试失败时的处理策略
    FailureStrategy FailureStrategy `json:"failureStrategy"`

    // 无测试文件时的处理
    NoTestsStrategy NoTestsStrategy `json:"noTestsStrategy"`
}

// ToolCommand 工具命令配置
type ToolCommand struct {
    // 工具名称
    Name string `json:"name"`

    // 执行命令
    Command string `json:"command"`

    // 命令参数
    Args []string `json:"args"`

    // 环境变量
    Env map[string]string `json:"env"`

    // 工作目录（相对于 worktree root）
    WorkingDir string `json:"workingDir"`

    // 超时时间（秒）
    Timeout int `json:"timeout"`

    // 是否检查配置文件存在
    CheckConfigFile string `json:"checkConfigFile"`
}

// FailureStrategy 失败处理策略
type FailureStrategy string

const (
    // FailFast - 立即失败，停止后续操作
    FailFast FailureStrategy = "fail_fast"

    // AutoFix - 尝试自动修复后重试
    AutoFix FailureStrategy = "auto_fix"

    // WarnContinue - 记录警告，继续执行
    WarnContinue FailureStrategy = "warn_continue"

    // Skip - 跳过此步骤
    Skip FailureStrategy = "skip"
)

// NoTestsStrategy 无测试文件时的处理策略
type NoTestsStrategy string

const (
    // SkipNoTests - 跳过测试
    SkipNoTests NoTestsStrategy = "skip"

    // WarnNoTests - 记录警告，继续执行
    WarnNoTests NoTestsStrategy = "warn"

    // FailNoTests - 失败，要求添加测试
    FailNoTests NoTestsStrategy = "fail"
)

// AutoDetectionConfig 自动检测配置
type AutoDetectionConfig struct {
    // 是否启用自动检测
    Enabled bool `json:"enabled"`

    // 检测模式
    Mode DetectionMode `json:"mode"`

    // 检测到的项目信息（只读，由系统填充）
    DetectedProject *DetectedProject `json:"detectedProject,omitempty"`
}

// DetectionMode 检测模式
type DetectionMode string

const (
    // AutoDetectFull - 完全自动检测
    AutoDetectFull DetectionMode = "auto_full"

    // AutoDetectBasic - 基础检测（仅检测语言和框架）
    AutoDetectBasic DetectionMode = "auto_basic"

    // ManualOnly - 禁用自动检测，仅使用配置
    ManualOnly DetectionMode = "manual_only"
)

// DetectedProject 检测到的项目信息
type DetectedProject struct {
    // 项目类型
    Type ProjectType `json:"type"`

    // 主要语言
    PrimaryLanguage string `json:"primaryLanguage"`

    // 语言列表（多语言项目）
    Languages []string `json:"languages"`

    // 检测到的工具
    DetectedTools []DetectedTool `json:"detectedTools"`

    // 检测时间
    DetectedAt time.Time `json:"detectedAt"`

    // 检测置信度（0-100）
    Confidence int `json:"confidence"`
}

// ProjectType 项目类型
type ProjectType string

const (
    ProjectTypeGo         ProjectType = "go"
    ProjectTypeNodeJS     ProjectType = "nodejs"
    ProjectTypePython     ProjectType = "python"
    ProjectTypeRust       ProjectType = "rust"
    ProjectTypeJava       ProjectType = "java"
    ProjectTypeMixed      ProjectType = "mixed"
    ProjectTypeUnknown    ProjectType = "unknown"
)

// DetectedTool 检测到的工具
type DetectedTool struct {
    // 工具类型
    Type ToolType `json:"type"`

    // 工具名称
    Name string `json:"name"`

    // 配置文件路径（如果有）
    ConfigFile string `json:"configFile"`

    // 推荐命令
    RecommendedCommand ToolCommand `json:"recommendedCommand"`

    // 检测依据
    DetectionBasis string `json:"detectionBasis"`
}

// ToolType 工具类型
type ToolType string

const (
    ToolTypeFormatter ToolType = "formatter"
    ToolTypeLinter    ToolType = "linter"
    ToolTypeTester    ToolType = "tester"
)
```

### 2.3 配置示例

#### Go 项目（自动检测）

```json
{
  "enabled": true,
  "autoDetection": {
    "enabled": true,
    "mode": "auto_full"
  },
  "formatting": {
    "enabled": true,
    "tools": [
      {
        "name": "gofmt",
        "command": "gofmt",
        "args": ["-w", "."],
        "timeout": 60
      }
    ],
    "failureStrategy": "auto_fix"
  },
  "linting": {
    "enabled": true,
    "tools": [
      {
        "name": "golangci-lint",
        "command": "golangci-lint",
        "args": ["run", "--fix"],
        "checkConfigFile": ".golangci.yml",
        "timeout": 120,
        "env": {}
      }
    ],
    "failureStrategy": "auto_fix",
    "autoFix": true
  },
  "testing": {
    "enabled": true,
    "command": {
      "name": "go-test",
      "command": "go",
      "args": ["test", "-v", "./..."],
      "timeout": 300
    },
    "failureStrategy": "fail_fast",
    "noTestsStrategy": "warn"
  }
}
```

#### Node.js 项目（自动检测）

```json
{
  "enabled": true,
  "autoDetection": {
    "enabled": true,
    "mode": "auto_full"
  },
  "formatting": {
    "enabled": true,
    "tools": [
      {
        "name": "prettier",
        "command": "npx",
        "args": ["prettier", "--write", "."],
        "checkConfigFile": ".prettierrc",
        "timeout": 120
      }
    ],
    "failureStrategy": "auto_fix"
  },
  "linting": {
    "enabled": true,
    "tools": [
      {
        "name": "eslint",
        "command": "npx",
        "args": ["eslint", "--fix", "."],
        "checkConfigFile": ".eslintrc",
        "timeout": 120
      }
    ],
    "failureStrategy": "auto_fix",
    "autoFix": true
  },
  "testing": {
    "enabled": true,
    "command": {
      "name": "jest",
      "command": "npx",
      "args": ["jest", "--passWithNoTests"],
      "timeout": 300
    },
    "failureStrategy": "fail_fast",
    "noTestsStrategy": "skip"
  }
}
```

#### 禁用自动检测，使用 Makefile

```json
{
  "enabled": true,
  "autoDetection": {
    "enabled": false,
    "mode": "manual_only"
  },
  "formatting": {
    "enabled": true,
    "tools": [
      {
        "name": "make-format",
        "command": "make",
        "args": ["format"],
        "timeout": 120
      }
    ],
    "failureStrategy": "fail_fast"
  },
  "linting": {
    "enabled": true,
    "tools": [
      {
        "name": "make-lint",
        "command": "make",
        "args": ["lint"],
        "timeout": 120
      }
    ],
    "failureStrategy": "fail_fast",
    "autoFix": false
  },
  "testing": {
    "enabled": true,
    "command": {
      "name": "make-test",
      "command": "make",
      "args": ["test"],
      "timeout": 300
    },
    "failureStrategy": "fail_fast",
    "noTestsStrategy": "fail"
  }
}
```

---

## 3. 自动检测机制设计

### 3.1 检测流程

```
┌─────────────────────────────────────────────────────────────────────┐
│                        项目自动检测流程                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────┐                                                │
│  │ 1. 语言检测     │                                                │
│  │ - 检查项目文件  │                                                │
│  │ - 识别主要语言  │                                                │
│  └────────┬────────┘                                                │
│           │                                                         │
│           ▼                                                         │
│  ┌─────────────────┐                                                │
│  │ 2. 工具检测     │                                                │
│  │ - 检查配置文件  │                                                │
│  │ - 检查依赖文件  │                                                │
│  │ - 检查 Makefile │                                                │
│  └────────┬────────┘                                                │
│           │                                                         │
│           ▼                                                         │
│  ┌─────────────────┐                                                │
│  │ 3. 生成配置     │                                                │
│  │ - 组装推荐命令  │                                                │
│  │ - 设置默认策略  │                                                │
│  └────────┬────────┘                                                │
│           │                                                         │
│           ▼                                                         │
│  ┌─────────────────┐                                                │
│  │ 4. 保存结果     │                                                │
│  │ - 存入仓库配置  │                                                │
│  │ - UI 可查看     │                                                │
│  └─────────────────┘                                                │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 语言检测规则

| 检测文件 | 识别语言 | 置信度 |
|---------|---------|--------|
| `go.mod` | Go | 95% |
| `package.json` | Node.js | 90% |
| `requirements.txt` / `pyproject.toml` / `setup.py` | Python | 90% |
| `Cargo.toml` | Rust | 95% |
| `pom.xml` / `build.gradle` | Java | 90% |
| `Makefile` | 多语言 | 70%（需进一步检测） |

### 3.3 工具检测规则

#### Go 项目

| 检测文件/条件 | 检测工具 | 推荐命令 |
|--------------|---------|---------|
| 默认（Go 项目必有） | `gofmt` | `gofmt -w .` |
| `.golangci.yml` / `.golangci.toml` | `golangci-lint` | `golangci-lint run --fix` |
| 无 `.golangci.yml` 但有 `go.mod` | `go vet` | `go vet ./...` |
| `_test.go` 文件存在 | `go test` | `go test -v ./...` |
| `Makefile` 中有 `lint/format/test` target | 使用 Make | `make lint` 等 |

#### Node.js 项目

| 检测文件/条件 | 检测工具 | 推荐命令 |
|--------------|---------|---------|
| `.prettierrc` / `.prettierrc.json` / `prettier` in `package.json` | `prettier` | `npx prettier --write .` |
| `.eslintrc` / `.eslintrc.json` / `eslint` in `package.json` | `eslint` | `npx eslint --fix .` |
| `biome.json` | `biome` | `npx biome check --apply .` |
| `lint-staged` in `package.json` | `lint-staged` | `npx lint-staged` |
| `jest` / `mocha` / `vitest` in `package.json` | 测试框架 | 根据配置生成命令 |
| `Makefile` 中有 `lint/format/test` target | 使用 Make | `make lint` 等 |

#### Python 项目

| 检测文件/条件 | 检测工具 | 推荐命令 |
|--------------|---------|---------|
| `pyproject.toml` 中 `[tool.black]` | `black` | `black .` |
| `.flake8` / `setup.cfg` 中 flake8 配置 | `flake8` | `flake8 .` |
| `pyproject.toml` 中 `[tool.ruff]` | `ruff` | `ruff check --fix .` |
| `pytest` in requirements | `pytest` | `pytest` |
| `Makefile` 中有 `lint/format/test` target | 使用 Make | `make lint` 等 |

#### Rust 项目

| 检测文件/条件 | 检测工具 | 推荐命令 |
|--------------|---------|---------|
| 默认（Rust 项目必有） | `rustfmt` | `cargo fmt` |
| 默认（Rust 项目必有） | `clippy` | `cargo clippy --fix` |
| `Cargo.toml` | `cargo test` | `cargo test` |

### 3.4 检测优先级

当存在多种工具时，按以下优先级选择：

1. **Makefile target**（最高优先级）- 项目已有标准化的命令
2. **项目配置文件**（如 `.golangci.yml`）- 项目明确选择的工具
3. **依赖文件中的依赖**（如 `package.json` 中的 `eslint`）
4. **语言默认工具**（如 Go 的 `gofmt`）

### 3.5 检测器接口设计

```go
// internal/infrastructure/detection/project_detector.go

type ProjectDetector interface {
    // 检测项目信息
    Detect(ctx context.Context, worktreePath string) (*DetectedProject, error)

    // 生成执行验证配置
    GenerateConfig(detected *DetectedProject) *ExecutionValidationConfig
}

// GoProjectDetector Go 项目检测器
type GoProjectDetector struct{}

func (d *GoProjectDetector) Detect(ctx context.Context, worktreePath string) (*DetectedProject, error) {
    project := &DetectedProject{
        Type:             ProjectTypeGo,
        PrimaryLanguage:  "go",
        Languages:        []string{"go"},
        DetectedTools:    []DetectedTool{},
        DetectedAt:       time.Now(),
    }

    // 1. 检查 go.mod 存在
    goModPath := filepath.Join(worktreePath, "go.mod")
    if !fileExists(goModPath) {
        return nil, errors.New("not a go project")
    }

    // 2. 检测格式化工具
    project.DetectedTools = append(project.DetectedTools, DetectedTool{
        Type:            ToolTypeFormatter,
        Name:            "gofmt",
        DetectionBasis:  "go project default",
        RecommendedCommand: ToolCommand{
            Name:    "gofmt",
            Command: "gofmt",
            Args:    []string{"-w", "."},
            Timeout: 60,
        },
    })

    // 3. 检测 Lint 工具
    golangciPath := filepath.Join(worktreePath, ".golangci.yml")
    if fileExists(golangciPath) {
        project.DetectedTools = append(project.DetectedTools, DetectedTool{
            Type:            ToolTypeLinter,
            Name:            "golangci-lint",
            ConfigFile:      ".golangci.yml",
            DetectionBasis:  ".golangci.yml exists",
            RecommendedCommand: ToolCommand{
                Name:           "golangci-lint",
                Command:        "golangci-lint",
                Args:           []string{"run", "--fix"},
                CheckConfigFile: ".golangci.yml",
                Timeout:        120,
            },
        })
    } else {
        project.DetectedTools = append(project.DetectedTools, DetectedTool{
            Type:            ToolTypeLinter,
            Name:            "go vet",
            DetectionBasis:  "go project default",
            RecommendedCommand: ToolCommand{
                Name:    "go vet",
                Command: "go",
                Args:    []string{"vet", "./..."},
                Timeout: 60,
            },
        })
    }

    // 4. 检测测试
    hasTests := hasGoTestFiles(worktreePath)
    project.DetectedTools = append(project.DetectedTools, DetectedTool{
        Type:            ToolTypeTester,
        Name:            "go test",
        DetectionBasis:  hasTests ? "_test.go files found" : "go project default",
        RecommendedCommand: ToolCommand{
            Name:    "go test",
            Command: "go",
            Args:    []string{"test", "-v", "./..."},
            Timeout: 300,
        },
    })

    // 5. 检查 Makefile
    makefileTools := d.detectMakefileTargets(worktreePath)
    if len(makefileTools) > 0 {
        project.DetectedTools = makefileTools
    }

    project.Confidence = 95
    return project, nil
}

// detectMakefileTargets 检测 Makefile 中的 target
func (d *GoProjectDetector) detectMakefileTargets(worktreePath string) []DetectedTool {
    makefilePath := filepath.Join(worktreePath, "Makefile")
    if !fileExists(makefilePath) {
        return nil
    }

    tools := []DetectedTool{}
    targets := parseMakefileTargets(makefilePath)

    if targets["format"] || targets["fmt"] {
        tools = append(tools, DetectedTool{
            Type:            ToolTypeFormatter,
            Name:            "make format",
            DetectionBasis:  "Makefile has format/fmt target",
            RecommendedCommand: ToolCommand{
                Name:    "make format",
                Command: "make",
                Args:    []string{"format"},
                Timeout: 120,
            },
        })
    }

    if targets["lint"] {
        tools = append(tools, DetectedTool{
            Type:            ToolTypeLinter,
            Name:            "make lint",
            DetectionBasis:  "Makefile has lint target",
            RecommendedCommand: ToolCommand{
                Name:    "make lint",
                Command: "make",
                Args:    []string{"lint"},
                Timeout: 120,
            },
        })
    }

    if targets["test"] || targets["check"] {
        tools = append(tools, DetectedTool{
            Type:            ToolTypeTester,
            Name:            "make test",
            DetectionBasis:  "Makefile has test/check target",
            RecommendedCommand: ToolCommand{
                Name:    "make test",
                Command: "make",
                Args:    []string{"test"},
                Timeout: 300,
            },
        })
    }

    return tools
}
```

---

## 4. 执行流程集成

### 4.1 任务执行流程变更

```
┌─────────────────────────────────────────────────────────────────────┐
│                     任务执行流程（增加验证步骤）                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────┐                                                │
│  │ 1. Agent 执行   │                                                │
│  │   任务实现      │                                                │
│  └────────┬────────┘                                                │
│           │                                                         │
│           ▼                                                         │
│  ┌─────────────────┐                                                │
│  │ 2. 获取验证配置 │                                                │
│  │   仓库配置 >    │                                                │
│  │   自动检测 >    │                                                │
│  │   全局默认      │                                                │
│  └────────┬────────┘                                                │
│           │                                                         │
│           ▼                                                         │
│  ┌─────────────────┐                                                │
│  │ 3. 格式化代码   │                                                │
│  │   (如启用)      │                                                │
│  └────────┬────────┘                                                │
│           │                                                         │
│      ┌────┴────┐                                                    │
│      │         │                                                    │
│   成功      失败                                                    │
│      │         │                                                    │
│      │         ├─ auto_fix: 自动修复重试                            │
│      │         ├─ fail_fast: 暂停等待指令                           │
│      │         ├─ warn_continue: 记录警告继续                       │
│      │         └─ skip: 跳过                                        │
│      │                                                             │
│      ▼                                                             │
│  ┌─────────────────┐                                                │
│  │ 4. Lint 检查    │                                                │
│  │   (如启用)      │                                                │
│  └────────┬────────┘                                                │
│           │                                                         │
│      ┌────┴────┐                                                    │
│      │         │                                                    │
│   成功      失败                                                    │
│      │         │                                                    │
│      │         ├─ auto_fix: 执行 lint --fix 重试                    │
│      │         ├─ fail_fast: 暂停等待指令                           │
│      │         ├─ warn_continue: 记录警告继续                       │
│      │         └─ skip: 跳过                                        │
│      │                                                             │
│      ▼                                                             │
│  ┌─────────────────┐                                                │
│  │ 5. 运行测试     │                                                │
│  │   (如启用)      │                                                │
│  └────────┬────────┘                                                │
│           │                                                         │
│      ┌────┴────┐                                                    │
│      │         │                                                    │
│   成功      失败                                                    │
│      │         │                                                    │
│      │         ├─ fail_fast: Agent 尝试修复重试                     │
│      │         ├─ warn_continue: 记录警告继续                       │
│      │         └─ skip: 跳过                                        │
│      │                                                             │
│      ▼                                                             │
│  ┌─────────────────┐                                                │
│  │ 6. 任务完成     │                                                │
│  │   记录结果      │                                                │
│  └─────────────────┘                                                │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 4.2 执行验证服务设计

```go
// internal/domain/service/execution_validator.go

type ExecutionValidator interface {
    // 执行完整验证流程
    Validate(ctx context.Context, req *ValidationRequest) (*ValidationResult, error)

    // 仅执行格式化
    Format(ctx context.Context, req *ValidationRequest) (*FormatResult, error)

    // 仅执行 Lint
    Lint(ctx context.Context, req *ValidationRequest) (*LintResult, error)

    // 仅执行测试
    Test(ctx context.Context, req *ValidationRequest) (*TestResult, error)
}

type ValidationRequest struct {
    SessionID     uuid.UUID
    TaskID        uuid.UUID
    WorktreePath  string
    Config        *ExecutionValidationConfig
}

type ValidationResult struct {
    FormatResult  *FormatResult
    LintResult    *LintResult
    TestResult    *TestResult
    OverallStatus ValidationStatus
    Warnings      []string
    Duration      time.Duration
}

type ValidationStatus string

const (
    ValidationPassed    ValidationStatus = "passed"
    ValidationFailed    ValidationStatus = "failed"
    ValidationWarned    ValidationStatus = "warned"
    ValidationSkipped   ValidationStatus = "skipped"
)

// ExecutionValidatorImpl 实现
type ExecutionValidatorImpl struct {
    configResolver ConfigResolver
    commandExecutor CommandExecutor
    outputParser    OutputParser
}

func (v *ExecutionValidatorImpl) Validate(ctx context.Context, req *ValidationRequest) (*ValidationResult, error) {
    result := &ValidationResult{
        Warnings: []string{},
    }
    start := time.Now()

    // 1. 格式化
    if req.Config.Formatting.Enabled {
        formatResult, err := v.Format(ctx, req)
        if err != nil {
            return nil, err
        }
        result.FormatResult = formatResult

        if !formatResult.Success {
            switch req.Config.Formatting.FailureStrategy {
            case FailFast:
                result.OverallStatus = ValidationFailed
                return result, nil
            case WarnContinue:
                result.Warnings = append(result.Warnings, formatResult.Message)
            case AutoFix:
                // 已在 Format 中自动修复，继续
            }
        }
    }

    // 2. Lint
    if req.Config.Linting.Enabled {
        lintResult, err := v.Lint(ctx, req)
        if err != nil {
            return nil, err
        }
        result.LintResult = lintResult

        if !lintResult.Success {
            switch req.Config.Linting.FailureStrategy {
            case FailFast:
                result.OverallStatus = ValidationFailed
                return result, nil
            case WarnContinue:
                result.Warnings = append(result.Warnings, lintResult.Message)
            case AutoFix:
                // 已在 Lint 中自动修复，继续
            }
        }
    }

    // 3. 测试
    if req.Config.Testing.Enabled {
        testResult, err := v.Test(ctx, req)
        if err != nil {
            return nil, err
        }
        result.TestResult = testResult

        if !testResult.Success {
            switch req.Config.Testing.FailureStrategy {
            case FailFast:
                result.OverallStatus = ValidationFailed
                return result, nil
            case WarnContinue:
                result.Warnings = append(result.Warnings, testResult.Message)
            }
        }
    }

    // 4. 确定最终状态
    if len(result.Warnings) > 0 {
        result.OverallStatus = ValidationWarned
    } else {
        result.OverallStatus = ValidationPassed
    }

    result.Duration = time.Since(start)
    return result, nil
}
```

---

## 5. 数据库设计变更

### 5.1 repositories 表 config 字段扩展

```sql
-- repositories 表 config 字段扩展结构
-- 原 config 字段保留，新增 execution_validation 子字段

-- config 字段完整结构示例：
{
  "maxConcurrency": 3,
  "complexityThreshold": 70,
  "forceDesignConfirm": false,
  "defaultModel": "claude-3-opus",
  "taskRetryLimit": 3,
  "executionValidation": {
    "enabled": true,
    "autoDetection": {
      "enabled": true,
      "mode": "auto_full",
      "detectedProject": {
        "type": "go",
        "primaryLanguage": "go",
        "languages": ["go"],
        "detectedTools": [...],
        "detectedAt": "2026-04-04T10:00:00Z",
        "confidence": 95
      }
    },
    "formatting": {
      "enabled": true,
      "tools": [...],
      "failureStrategy": "auto_fix"
    },
    "linting": {
      "enabled": true,
      "tools": [...],
      "failureStrategy": "auto_fix",
      "autoFix": true
    },
    "testing": {
      "enabled": true,
      "command": {...},
      "failureStrategy": "fail_fast",
      "noTestsStrategy": "warn"
    }
  }
}
```

### 5.2 新增 execution_validation_results 表

记录每次验证的结果，便于追溯和统计。

```sql
CREATE TABLE execution_validation_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES work_sessions(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,

    -- 格式化结果
    format_success BOOLEAN,
    format_output TEXT,
    format_duration_ms BIGINT,

    -- Lint 结果
    lint_success BOOLEAN,
    lint_output TEXT,
    lint_issues_found INT,
    lint_issues_fixed INT,
    lint_duration_ms BIGINT,

    -- 测试结果
    test_success BOOLEAN,
    test_output TEXT,
    test_passed INT,
    test_failed INT,
    test_duration_ms BIGINT,

    -- 总体结果
    overall_status VARCHAR(50) NOT NULL,  -- passed / failed / warned / skipped
    warnings JSONB DEFAULT '[]',

    -- 时间
    total_duration_ms BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_session FOREIGN KEY (session_id) REFERENCES work_sessions(id) ON DELETE CASCADE,
    CONSTRAINT fk_task FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX idx_validation_results_session ON execution_validation_results(session_id);
CREATE INDEX idx_validation_results_task ON execution_validation_results(task_id);
CREATE INDEX idx_validation_results_status ON execution_validation_results(overall_status);
```

---

## 6. API 设计扩展

### 6.1 仓库配置 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/repositories/:id/validation-config` | 获取执行验证配置 |
| PUT | `/api/v1/repositories/:id/validation-config` | 更新执行验证配置 |
| POST | `/api/v1/repositories/:id/detect-project` | 触发项目检测 |
| GET | `/api/v1/repositories/:id/detection-result` | 获取检测结果 |

### 6.2 API 详细定义

#### GET /api/v1/repositories/:id/validation-config

```json
// Response
{
  "enabled": true,
  "autoDetection": {
    "enabled": true,
    "mode": "auto_full",
    "detectedProject": {
      "type": "go",
      "primaryLanguage": "go",
      "languages": ["go"],
      "detectedTools": [
        {
          "type": "formatter",
          "name": "gofmt",
          "detectionBasis": "go project default",
          "recommendedCommand": {
            "name": "gofmt",
            "command": "gofmt",
            "args": ["-w", "."],
            "timeout": 60
          }
        },
        {
          "type": "linter",
          "name": "golangci-lint",
          "configFile": ".golangci.yml",
          "detectionBasis": ".golangci.yml exists",
          "recommendedCommand": {
            "name": "golangci-lint",
            "command": "golangci-lint",
            "args": ["run", "--fix"],
            "checkConfigFile": ".golangci.yml",
            "timeout": 120
          }
        }
      ],
      "detectedAt": "2026-04-04T10:00:00Z",
      "confidence": 95
    }
  },
  "formatting": {
    "enabled": true,
    "tools": [...],
    "failureStrategy": "auto_fix"
  },
  "linting": {...},
  "testing": {...}
}
```

#### PUT /api/v1/repositories/:id/validation-config

```json
// Request - 禁用自动检测，使用自定义命令
{
  "enabled": true,
  "autoDetection": {
    "enabled": false,
    "mode": "manual_only"
  },
  "formatting": {
    "enabled": true,
    "tools": [
      {
        "name": "make-format",
        "command": "make",
        "args": ["format"],
        "timeout": 120
      }
    ],
    "failureStrategy": "fail_fast"
  },
  "linting": {
    "enabled": true,
    "tools": [
      {
        "name": "make-lint",
        "command": "make",
        "args": ["lint"],
        "timeout": 120
      }
    ],
    "failureStrategy": "fail_fast",
    "autoFix": false
  },
  "testing": {
    "enabled": true,
    "command": {
      "name": "make-test",
      "command": "make",
      "args": ["test"],
      "timeout": 300
    },
    "failureStrategy": "fail_fast",
    "noTestsStrategy": "fail"
  }
}

// Response
{
  "success": true,
  "message": "Configuration updated"
}
```

#### POST /api/v1/repositories/:id/detect-project

触发重新检测项目工具配置。

```json
// Request
{
  "force": true  // 是否强制重新检测（即使已有检测结果）
}

// Response
{
  "success": true,
  "detectedProject": {
    "type": "go",
    "primaryLanguage": "go",
    "detectedTools": [...],
    "confidence": 95
  }
}
```

---

## 7. UI 设计

### 7.1 仓库配置页面结构

```
┌─────────────────────────────────────────────────────────────────────┐
│                     Repository Configuration                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Repository: owner/repo                                    [Save]  │
│                                                                     │
│  ════════════════════════════════════════════════════════════════  │
│  ═══ 执行验证配置 (Execution Validation)                          ══ │
│  ════════════════════════════════════════════════════════════════  │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ [✓] 启用执行验证                                            │   │
│  │                                                             │   │
│  │ 自动检测                                                    │   │
│  │ ┌───────────────────────────────────────────────────────┐   │   │
│  │ │ ○ 完全自动检测 (推荐)                                  │   │   │
│  │ │   自动检测语言、工具和配置                              │   │   │
│  │ │                                                         │   │   │
│  │ │ ○ 基础检测                                              │   │   │
│  │ │   仅检测语言，其他手动配置                              │   │   │
│  │ │                                                         │   │   │
│  │ │ ● 禁用自动检测                                          │   │   │
│  │ │   所有配置手动填写                                      │   │   │
│  │ └───────────────────────────────────────────────────────┘   │   │
│  │                                                             │   │
│  │ [重新检测项目]                                              │   │
│  │                                                             │   │
│  │ ┌───────────────────────────────────────────────────────┐   │   │
│  │ │ 检测结果                                               │   │   │
│  │ │                                                         │   │   │
│  │ │ 项目类型: Go                                            │   │   │
│  │ │ 主要语言: go                                            │   │   │
│  │ │ 检测置信度: 95%                                         │   │   │
│  │ │ 检测时间: 2026-04-04 10:00                              │   │   │
│  │ │                                                         │   │   │
│  │ │ 检测到的工具:                                           │   │   │
│  │ │ ┌─────────────────────────────────────────────────────┐│   │   │
│  │ │ │ 格式化: gofmt (go project default)                  ││   │   │
│  │ │ │ Lint: golangci-lint (.golangci.yml exists)          ││   │   │
│  │ │ │ 测试: go test (_test.go files found)                ││   │   │
│  │ │ └─────────────────────────────────────────────────────┘│   │   │
│  │ └───────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ════════════════════════════════════════════════════════════════  │
│  ═══ 格式化配置                                                  ══ │
│  ════════════════════════════════════════════════════════════════  │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ [✓] 启用格式化                                              │   │
│  │                                                             │   │
│  │ 格式化工具                                                   │   │
│  │ ┌─────────────────────────────────────────────────────────┐ │   │
│  │ │ 工具: [gofmt        ▼]                                  │ │   │
│  │ │ 命令: [gofmt          ]                                 │ │   │
│  │ │ 参数: [-w, .          ]                                 │ │   │
│  │ │ 超时: [60        ] 秒                                   │ │   │
│  │ │ 配置文件检查: [          ] (可选)                        │ │   │
│  │ │                                                         │ │   │
│  │ │ [+ 添加另一个格式化工具]                                 │ │   │
│  │ └─────────────────────────────────────────────────────────┘ │   │
│  │                                                             │   │
│  │ 失败处理策略                                                 │   │
│  │ ┌─────────────────────────────────────────────────────────┐ │   │
│  │ │ ● 自动修复重试 (推荐)                                    │ │   │
│  │ │ ○ 立即失败，等待指令                                      │ │   │
│  │ │ ○ 记录警告，继续执行                                      │ │   │
│  │ │ ○ 跳过此步骤                                              │ │   │
│  │ └─────────────────────────────────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ════════════════════════════════════════════════════════════════  │
│  ═══ Lint 配置                                                   ══ │
│  ════════════════════════════════════════════════════════════════  │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ [✓] 启用 Lint 检查                                          │   │
│  │                                                             │   │
│  │ Lint 工具                                                   │   │
│  │ ┌─────────────────────────────────────────────────────────┐ │   │
│  │ │ 工具: [golangci-lint▼]                                  │ │   │
│  │ │ 命令: [golangci-lint   ]                                │ │   │
│  │ │ 参数: [run, --fix      ]                                │ │   │
│  │ │ 超时: [120       ] 秒                                   │ │   │
│  │ │ 配置文件检查: [.golangci.yml▼]                           │ │   │
│  │ │                                                         │ │   │
│  │ │ [+ 添加另一个 Lint 工具]                                 │ │   │
│  │ └─────────────────────────────────────────────────────────┘ │   │
│  │                                                             │   │
│  │ [✓] 启用自动修复 (lint --fix)                               │   │
│  │                                                             │   │
│  │ 失败处理策略                                                 │   │
│  │ ┌─────────────────────────────────────────────────────────┐ │   │
│  │ │ ● 自动修复重试 (推荐)                                    │ │   │
│  │ │ ○ 立即失败，等待指令                                      │   │
│  │ │ ○ 记录警告，继续执行                                      │ │   │
│  │ │ ○ 跳过此步骤                                              │ │   │
│  │ └─────────────────────────────────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ════════════════════════════════════════════════════════════════  │
│  ═══ 测试配置                                                    ══ │
│  ════════════════════════════════════════════════════════════════  │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ [✓] 启用测试                                                │   │
│  │                                                             │   │
│  │ 测试命令                                                    │   │
│  │ ┌─────────────────────────────────────────────────────────┐ │   │
│  │ │ 工具: [go test      ▼]                                  │ │   │
│  │ │ 命令: [go             ]                                 │ │   │
│  │ │ 参数: [test, -v, ./... ]                                │ │   │
│  │ │ 超时: [300       ] 秒                                   │ │   │
│  │ └─────────────────────────────────────────────────────────┘ │   │
│  │                                                             │   │
│  │ 无测试文件时的处理                                           │   │
│  │ ┌─────────────────────────────────────────────────────────┐ │   │
│  │ │ ○ 跳过测试                                              │ │   │
│  │ │ ● 记录警告，继续执行                                      │ │   │
│  │ │ ○ 失败，要求添加测试                                      │ │   │
│  │ └─────────────────────────────────────────────────────────┘ │   │
│  │                                                             │   │
│  │ 失败处理策略                                                 │   │
│  │ ┌─────────────────────────────────────────────────────────┐ │   │
│  │ │ ● 立即失败，Agent 尝试修复                                │ │   │
│  │ │ ○ 记录警告，继续执行                                      │ │   │
│  │ │ ○ 跳过此步骤                                              │ │   │
│  │ └─────────────────────────────────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│                                              [保存配置] [重置默认] │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 7.2 UI 组件设计

#### 工具命令编辑器组件

```tsx
// web/src/components/features/ToolCommandEditor.tsx

interface ToolCommandEditorProps {
  value: ToolCommand;
  onChange: (value: ToolCommand) => void;
  toolSuggestions?: string[];  // 工具名称建议列表
  onRemove?: () => void;
}

function ToolCommandEditor({ value, onChange, toolSuggestions, onRemove }: ToolCommandEditorProps) {
  return (
    <div className="border rounded-lg p-4 space-y-3">
      <div className="flex justify-between items-center">
        <Label>工具配置</Label>
        {onRemove && <Button variant="ghost" size="sm" onClick={onRemove}>删除</Button>}
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <Label>工具名称</Label>
          <Combobox
            value={value.name}
            onChange={(name) => onChange({ ...value, name })}
            options={toolSuggestions || []}
            placeholder="选择或输入工具名称"
          />
        </div>

        <div>
          <Label>执行命令</Label>
          <Input
            value={value.command}
            onChange={(e) => onChange({ ...value, command: e.target.value })}
            placeholder="如: gofmt, npx, make"
          />
        </div>
      </div>

      <div>
        <Label>命令参数</Label>
        <ArrayInput
          value={value.args}
          onChange={(args) => onChange({ ...value, args })}
          placeholder="添加参数"
        />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <Label>超时时间（秒）</Label>
          <Input
            type="number"
            value={value.timeout}
            onChange={(e) => onChange({ ...value, timeout: parseInt(e.target.value) })}
          />
        </div>

        <div>
          <Label>配置文件检查（可选）</Label>
          <Input
            value={value.checkConfigFile || ''}
            onChange={(e) => onChange({ ...value, checkConfigFile: e.target.value })}
            placeholder="如: .golangci.yml"
          />
        </div>
      </div>
    </div>
  );
}
```

#### 检测结果展示组件

```tsx
// web/src/components/features/DetectionResultDisplay.tsx

interface DetectionResultDisplayProps {
  detectedProject: DetectedProject | null;
  onRefresh: () => void;
  isDetecting: boolean;
}

function DetectionResultDisplay({ detectedProject, onRefresh, isDetecting }: DetectionResultDisplayProps) {
  if (isDetecting) {
    return (
      <div className="border rounded-lg p-4">
        <div className="flex items-center space-x-2">
          <Spinner />
          <span>正在检测项目...</span>
        </div>
      </div>
    );
  }

  if (!detectedProject) {
    return (
      <div className="border rounded-lg p-4">
        <Alert variant="info">
          尚未检测项目，点击下方按钮开始检测
        </Alert>
        <Button onClick={onRefresh} className="mt-3">检测项目</Button>
      </div>
    );
  }

  return (
    <div className="border rounded-lg p-4 space-y-3">
      <div className="flex justify-between items-center">
        <h4 className="font-medium">检测结果</h4>
        <Button variant="outline" size="sm" onClick={onRefresh}>
          重新检测
        </Button>
      </div>

      <div className="grid grid-cols-3 gap-4 text-sm">
        <div>
          <span className="text-muted-foreground">项目类型:</span>
          <Badge>{detectedProject.type}</Badge>
        </div>
        <div>
          <span className="text-muted-foreground">主要语言:</span>
          <span>{detectedProject.primaryLanguage}</span>
        </div>
        <div>
          <span className="text-muted-foreground">置信度:</span>
          <span>{detectedProject.confidence}%</span>
        </div>
      </div>

      <div>
        <h5 className="text-sm font-medium mb-2">检测到的工具:</h5>
        <div className="space-y-1">
          {detectedProject.detectedTools.map((tool, idx) => (
            <div key={idx} className="flex items-center space-x-2 text-sm">
              <Badge variant={
                tool.type === 'formatter' ? 'default' :
                tool.type === 'linter' ? 'warning' : 'success'
              }>
                {tool.type === 'formatter' ? '格式化' :
                 tool.type === 'linter' ? 'Lint' : '测试'}
              </Badge>
              <span className="font-medium">{tool.name}</span>
              <span className="text-muted-foreground text-xs">
                ({tool.detectionBasis})
              </span>
            </div>
          ))}
        </div>
      </div>

      <p className="text-xs text-muted-foreground">
        检测时间: {formatDate(detectedProject.detectedAt)}
      </p>
    </div>
  );
}
```

### 7.3 前端页面结构补充

在 `architecture-design.md` 的前端目录结构中补充：

```
web/src/
├── components/
│   └── features/
│       ├── ValidationConfigForm/        # 执行验证配置表单
│       │   ├── index.tsx
│       │   ├── FormattingSection.tsx    # 格式化配置部分
│       │   ├── LintingSection.tsx       # Lint 配置部分
│       │   ├── TestingSection.tsx       # 测试配置部分
│       │   └── AutoDetectionSection.tsx # 自动检测配置部分
│       ├── ToolCommandEditor.tsx        # 工具命令编辑器
│       ├── DetectionResultDisplay.tsx   # 检测结果展示
│       └── ValidationResultView.tsx     # 验证结果展示（任务详情页）
│
├── pages/
│   ├── RepositoryConfig.tsx             # 仓库配置页（扩展）
│   └── TaskDetail.tsx                   # 任务详情页（扩展，显示验证结果）
│
├── hooks/
│   ├── useValidationConfig.ts           # 验证配置 Hook
│   └── useProjectDetection.ts           # 项目检测 Hook
│
├── services/
│   └── validationService.ts             # 验证配置 API 服务
│
├── types/
│   └── validation.ts                    # 验证相关类型定义
```

---

## 8. 全局默认配置

### 8.1 配置文件扩展

```yaml
# config.yaml 扩展

executionValidation:
  # 全局默认配置（当仓库未配置且检测失败时使用）
  default:
    enabled: true

    formatting:
      enabled: true
      failureStrategy: auto_fix

    linting:
      enabled: true
      autoFix: true
      failureStrategy: auto_fix

    testing:
      enabled: true
      failureStrategy: fail_fast
      noTestsStrategy: warn

  # 检测配置
  detection:
    # 检测超时时间
    timeout: 30s

    # 检测器优先级（按语言）
    detectorPriority:
      - go
      - nodejs
      - python
      - rust
      - java

  # 各语言默认工具配置
  languageDefaults:
    go:
      formatting:
        tools:
          - name: gofmt
            command: gofmt
            args: ["-w", "."]
            timeout: 60
      linting:
        tools:
          - name: go vet
            command: go
            args: ["vet", "./..."]
            timeout: 60
      testing:
        command:
          name: go test
          command: go
          args: ["test", "-v", "./..."]
          timeout: 300

    nodejs:
      formatting:
        tools:
          - name: prettier
            command: npx
            args: ["prettier", "--write", "."]
            timeout: 120
            checkConfigFile: ".prettierrc"
      linting:
        tools:
          - name: eslint
            command: npx
            args: ["eslint", "--fix", "."]
            timeout: 120
            checkConfigFile: ".eslintrc"
      testing:
        command:
          name: jest
          command: npx
          args: ["jest", "--passWithNoTests"]
          timeout: 300

    python:
      formatting:
        tools:
          - name: black
            command: black
            args: ["."]
            timeout: 120
      linting:
        tools:
          - name: flake8
            command: flake8
            args: ["."]
            timeout: 60
      testing:
        command:
          name: pytest
          command: pytest
          args: []
          timeout: 300
```

---

## 9. 执行流程与 Agent 集成

### 9.1 Agent 任务执行 Prompt 扩展

执行任务时，需要告知 Agent 执行验证的要求：

```markdown
## 任务执行要求

请执行以下任务：
{taskDescription}

### 执行后验证

任务完成后，系统将自动执行以下验证：

1. **代码格式化**: {formatTools}
2. **Lint 检查**: {lintTools}
3. **运行测试**: {testCommand}

请确保：
- 编写的代码符合项目的格式规范
- 代码通过 Lint 检查
- 相关测试通过

如果验证失败，系统会尝试自动修复，但请尽量在实现时就注意代码质量。
```

### 9.2 Agent Runner 集成

```go
// internal/infrastructure/agent/claude/claude_runner.go

func (a *ClaudeCodeAgent) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error) {
    // 1. 构建 Prompt（包含验证要求）
    prompt := a.buildExecutePromptWithValidation(req)

    // 2. 执行 Agent
    result, err := a.runAgent(ctx, prompt, req.WorktreePath)
    if err != nil {
        return nil, err
    }

    // 3. 执行验证
    if req.ValidationConfig != nil && req.ValidationConfig.Enabled {
        validationResult, err := a.validator.Validate(ctx, &ValidationRequest{
            SessionID:    req.SessionID,
            TaskID:       req.TaskID,
            WorktreePath: req.WorktreePath,
            Config:       req.ValidationConfig,
        })
        if err != nil {
            return nil, err
        }

        result.ValidationResult = validationResult

        // 验证失败时，尝试让 Agent 修复
        if validationResult.OverallStatus == ValidationFailed {
            if req.ValidationConfig.Testing.FailureStrategy == FailFast {
                // 让 Agent 修复测试失败
                fixResult, err := a.attemptAutoFix(ctx, req, validationResult)
                if err != nil {
                    return nil, err
                }
                if fixResult.Success {
                    result.ValidationResult = fixResult.ValidationResult
                } else {
                    result.Success = false
                    result.Error = &ExecuteError{
                        Type:    ErrorTypeTestFailed,
                        Message: fixResult.Message,
                    }
                }
            }
        }
    }

    return result, nil
}
```

---

## 10. 错误处理与重试

### 10.1 验证失败错误类型

```go
const (
    ErrorTypeFormatFailed    ErrorType = "format_failed"
    ErrorTypeLintFailed      ErrorType = "lint_failed"
    ErrorTypeTestFailed      ErrorType = "test_failed"
    ErrorTypeValidationTimeout ErrorType = "validation_timeout"
)
```

### 10.2 自动修复尝试

```go
// attemptAutoFix 尝试自动修复验证失败
func (a *ClaudeCodeAgent) attemptAutoFix(
    ctx context.Context,
    req *ExecuteRequest,
    validationResult *ValidationResult,
) (*AutoFixResult, error) {
    // 根据失败类型构建修复 Prompt
    var fixPrompt string

    if validationResult.TestResult != nil && !validationResult.TestResult.Success {
        fixPrompt = fmt.Sprintf(`
测试失败，请修复以下问题：

测试输出：
%s

失败的测试：
%s

请分析失败原因并修复代码，确保测试通过。
`, validationResult.TestResult.Output, validationResult.TestResult.FailedTests)
    }

    if validationResult.LintResult != nil && !validationResult.LintResult.Success {
        fixPrompt = fmt.Sprintf(`
Lint 检查失败，请修复以下问题：

Lint 输出：
%s

请修复 Lint 问题，确保代码质量。
`, validationResult.LintResult.Output)
    }

    // 执行修复
    fixResult, err := a.runAgent(ctx, fixPrompt, req.WorktreePath)
    if err != nil {
        return nil, err
    }

    // 重新验证
    newValidation, err := a.validator.Validate(ctx, &ValidationRequest{
        SessionID:    req.SessionID,
        TaskID:       req.TaskID,
        WorktreePath: req.WorktreePath,
        Config:       req.ValidationConfig,
    })
    if err != nil {
        return nil, err
    }

    return &AutoFixResult{
        Success:          newValidation.OverallStatus == ValidationPassed,
        ValidationResult: newValidation,
        Message:          fixResult.Output,
    }, nil
}
```

---

## 11. 配置迁移与兼容性

### 11.1 从现有配置迁移

对于已有的仓库配置，系统会：

1. 检查是否存在 `executionValidation` 配置
2. 若不存在，触发自动检测
3. 将检测结果存入配置
4. UI 展示检测结果，用户可确认或修改

### 11.2 配置版本管理

```go
type ConfigVersion struct {
    Version     int       `json:"version"`
    UpdatedAt   time.Time `json:"updatedAt"`
    UpdatedBy   string    `json:"updatedBy"`
    ChangeReason string   `json:"changeReason"`
}
```

每次配置变更时记录版本，便于回溯。

---

## 12. 测试策略

### 12.1 单元测试

- 各语言检测器测试
- 配置解析测试
- 命令执行测试
- 失败策略测试

### 12.2 集成测试

- 完整验证流程测试
- Agent + 验证集成测试
- 自动修复流程测试

### 12.3 测试项目

准备不同语言的测试项目：
- Go 项目（有/无 golangci-lint 配置）
- Node.js 项目（有/无 prettier/eslint 配置）
- Python 项目（有/无 black/flake8 配置）
- Makefile 项目
- 多语言项目
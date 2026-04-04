# Agent 调用层设计文档

## 1. 概述

本文档定义 Litchi 系统与 Claude Code CLI 的集成方式，包括调用协议、参数配置、输出处理等。

---

## 2. Claude Code CLI 关键参数

### 2.1 核心调用模式参数

| 参数 | 说明 | Litchi 使用场景 |
|------|------|-----------------|
| `-p, --print` | 非交互模式，输出后退出 | **必须**：所有自动化调用 |
| `--output-format` | 输出格式：`text`、`json`、`stream-json` | `stream-json` 用于实时进度监控 |
| `--input-format` | 输入格式：`text`、`stream-json` | `stream-json` 支持实时交互 |
| `--json-schema` | JSON Schema 验证结构化输出 | 设计方案、任务列表输出验证 |

### 2.2 工具控制参数

| 参数 | 说明 | Litchi 使用场景 |
|------|------|-----------------|
| `--allowedTools` | 允许的工具列表 | 限制危险操作 |
| `--disallowedTools` | 禁止的工具列表 | 禁止 force-push 等 |
| `--tools` | 指定可用工具集合 | 精细控制工具权限 |
| `--permission-mode` | 权限模式 | `acceptEdits` 自动接受编辑 |

**权限模式选项**：
| 模式 | 说明 |
|------|------|
| `default` | 默认，每次请求权限 |
| `acceptEdits` | 自动接受文件编辑 |
| `bypassPermissions` | 跳过所有权限检查（危险） |
| `dontAsk` | 不询问，拒绝危险操作 |
| `plan` | 规划模式，不执行 |
| `auto` | 自动模式 |

### 2.3 会话管理参数

| 参数 | 说明 | Litchi 使用场景 |
|------|------|-----------------|
| `--session-id <uuid>` | 指定会话 ID | 关联 WorkSession |
| `-c, --continue` | 继续最近对话 | 恢复中断任务 |
| `-r, --resume <value>` | 恢复指定对话 | 按 session ID 恢复 |
| `--fork-session` | 创建新会话 ID | 回退后新会话 |
| `--no-session-persistence` | 禁用会话持久化 | 一次性任务 |

### 2.4 Git/Worktree 参数

| 参数 | 说明 | Litchi 使用场景 |
|------|------|-----------------|
| `-w, --worktree [name]` | 创建 Git Worktree | **核心**：每个 Issue 独立 Worktree |
| `--tmux` | 创建 tmux 会话 | 可选：可视化监控 |

### 2.5 模型与 Agent 参数

| 参数 | 说明 | Litchi 使用场景 |
|------|------|-----------------|
| `--model` | 模型选择：`sonnet`、`opus`、`haiku` | 按复杂度选择模型 |
| `--agent` | 使用预定义 Agent | 澄清/设计/执行不同 Agent |
| `--agents <json>` | JSON 定义自定义 Agent | 动态 Agent 配置 |
| `--effort` | 努力级别：`low`、`medium`、`high`、`max` | 简单任务降级 |

### 2.6 系统提示参数

| 参数 | 说明 | Litchi 使用场景 |
|------|------|-----------------|
| `--system-prompt` | 系统提示 | 阶段特定指令 |
| `--append-system-prompt` | 添加系统提示 | 补充上下文 |
| `--add-dir` | 添加允许访问目录 | 多仓库场景 |

### 2.7 预算与限制参数

| 参数 | 说明 | Litchi 使用场景 |
|------|------|-----------------|
| `--max-budget-usd` | 最大 API 调用费用 | 成本控制 |
| `--fallback-model` | 备用模型 | 主模型过载时降级 |

### 2.8 MCP 配置参数

| 参数 | 说明 | Litchi 使用场景 |
|------|------|-----------------|
| `--mcp-config` | MCP 服务器配置文件 | 扩展工具能力 |
| `--strict-mcp-config` | 仅使用指定 MCP | 安全控制 |

---

## 3. Agent 接口定义

### 3.1 领域服务接口

> **说明**：此接口定义与 `architecture-design.md` 第 13.1 节保持一致，是系统的统一 Agent 抽象层。

```go
// internal/domain/service/agent_runner.go

type AgentRunner interface {
    // 执行任务
    Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error)
    
    // 分析代码库
    AnalyzeCodebase(ctx context.Context, req *AnalyzeRequest) (*CodeAnalysis, error)
    
    // 生成设计方案
    GenerateDesign(ctx context.Context, req *DesignRequest) (*DesignOutput, error)
    
    // 需求澄清对话
    Clarify(ctx context.Context, req *ClarifyRequest) (*ClarifyOutput, error)
    
    // 恢复中断的会话
    Resume(ctx context.Context, req *ResumeRequest) (*ExecuteResult, error)
}
```

**接口方法说明**：

| 方法 | 用途 | 调用阶段 |
|------|------|---------|
| Execute | 执行具体任务 | Execution |
| AnalyzeCodebase | 分析代码库结构、依赖关系 | Design |
| GenerateDesign | 生成设计方案并评估复杂度 | Design |
| Clarify | 需求澄清对话 | Clarification |
| Resume | 恢复中断的会话 | 服务重启、用户指令继续 |
```

### 3.2 请求/响应结构

```go
// ExecuteRequest - 任务执行请求
type ExecuteRequest struct {
    SessionID      uuid.UUID        // WorkSession ID，用于 --session-id
    WorktreePath   string           // Git Worktree 路径
    Task           *Task            // 待执行任务
    PermissionMode PermissionMode   // 权限模式
    AllowedTools   []string         // 允许的工具
    MaxBudgetUSD   float64          // 最大预算
    Model          string           // 模型选择
    Context        string           // 上下文信息（设计方案等）
}

type PermissionMode string

const (
    PermissionDefault      PermissionMode = "default"
    PermissionAcceptEdits  PermissionMode = "acceptEdits"
    PermissionDontAsk      PermissionMode = "dontAsk"
    PermissionPlan         PermissionMode = "plan"
)

// ExecuteResult - 执行结果
type ExecuteResult struct {
    Success      bool            // 是否成功
    Output       string          // 输出内容
    TestResults  []TestResult    // 测试结果
    FilesChanged []string        // 变更的文件列表
    Duration     time.Duration   // 执行耗时
    TokenUsage   TokenUsage      // Token 使用量
    CostUSD      float64         // 费用
    Error        *AgentError     // 错误信息（如有）
}

type AgentError struct {
    Type        ErrorType       // 错误类型
    Message     string          // 错误消息
    Suggestion  string          // 解决建议
    Retryable   bool            // 是否可重试
}

type ErrorType string

const (
    // 执行相关错误
    ErrorTypeTestFailed      ErrorType = "test_failed"
    ErrorTypeBuildFailed     ErrorType = "build_failed"
    ErrorTypeTimeout         ErrorType = "timeout"
    ErrorTypeBudgetExceeded  ErrorType = "budget_exceeded"
    ErrorTypePermissionDenied ErrorType = "permission_denied"

    // Agent 系统错误
    ErrorTypeAgentCrashed    ErrorType = "agent_crashed"      // 进程异常终止
    ErrorTypeOutputParseFail ErrorType = "output_parse_fail"   // 输出解析失败
    ErrorTypeSessionLost     ErrorType = "session_lost"       // 会话上下文丢失
    ErrorTypeToolBlocked     ErrorType = "tool_blocked"       // 工具权限被拒绝

    // 外部服务错误
    ErrorTypeGitHubRateLimit ErrorType = "github_rate_limit"  // GitHub API 限流
    ErrorTypeGitHubError     ErrorType = "github_api_error"   // GitHub API 其他错误
    ErrorTypeNetworkError    ErrorType = "network_error"      // 网络连接错误
    ErrorTypeGitError        ErrorType = "git_error"          // Git 操作错误

    // 环境错误
    ErrorTypeTestEnvUnavailable ErrorType = "test_env_unavailable" // 测试环境不可用
    ErrorTypeWorktreeError      ErrorType = "worktree_error"      // Worktree 操作错误

    // 通用错误
    ErrorTypeAPIError        ErrorType = "api_error"
    ErrorTypeUnknown         ErrorType = "unknown"
)

// SeverityLevel 错误严重程度
type SeverityLevel string

const (
    SeverityCritical SeverityLevel = "critical"  // 系统级故障
    SeverityHigh     SeverityLevel = "high"      // 会话阻塞
    SeverityMedium   SeverityLevel = "medium"    // 阶段阻塞
    SeverityLow      SeverityLevel = "low"       // 任务临时失败
)

// RecoveryCategory 错误可恢复性
type RecoveryCategory string

const (
    RecoveryAuto     RecoveryCategory = "auto"       // 自动恢复
    RecoverySemiAuto RecoveryCategory = "semi_auto"  // 半自动恢复
    RecoveryManual   RecoveryCategory = "manual"     // 需人工干预
    RecoveryNone     RecoveryCategory = "none"       // 不可恢复
)

// DesignRequest - 设计方案请求
type DesignRequest struct {
    SessionID        uuid.UUID
    WorktreePath     string
    Issue            *Issue            // Issue 信息
    ClarifiedNeeds   []string          // 已澄清的需求
    CodeAnalysis     *CodeAnalysis     // 代码分析结果
    Model            string            // 模型
    OutputSchema     *json.Schema      // 输出 JSON Schema
}

// DesignOutput - 设计方案输出
type DesignOutput struct {
    DesignContent    string            // 设计方案内容
    ComplexityScore  int               // 复杂度评分
    EstimatedFiles   []string          // 预估涉及文件
    EstimatedLOC     int               // 预估代码行数
    ModulesAffected  []string          // 涉及模块
    BreakingChanges  bool              // 是否有破坏性变更
    TestDifficulty   string            // 测试难度
}

// ClarifyRequest - 澄清请求
type ClarifyRequest struct {
    SessionID    uuid.UUID
    Issue        *Issue
    History      []ConversationTurn   // 对话历史
    NewAnswer    string               // 用户回答（如有）
}

// ClarifyOutput - 澄清输出
type ClarifyOutput struct {
    Questions        []string        // 待回答问题
    ConfirmedPoints  []string        // 已确认需求点
    ReadyForDesign   bool            // 是否可进入设计阶段
}
```

---

## 4. Claude Code 实现类

### 4.1 ClaudeCodeAgent 结构

```go
// internal/infrastructure/agent/claude/claude_runner.go

type ClaudeCodeAgent struct {
    config     *ClaudeConfig
    executor   *CommandExecutor
    parser     *OutputParser
}

type ClaudeConfig struct {
    BinaryPath       string            // claude 可执行文件路径
    DefaultModel     string            // 默认模型
    DefaultTimeout   time.Duration     // 默认超时
    MaxBudgetUSD     float64           // 全局最大预算
    MCPConfigPath    string            // MCP 配置文件路径
    DangerousOps     []string          // 需审批的危险操作
}

func NewClaudeCodeAgent(config *ClaudeConfig) *ClaudeCodeAgent {
    return &ClaudeCodeAgent{
        config:   config,
        executor: NewCommandExecutor(),
        parser:   NewOutputParser(),
    }
}
```

### 4.2 Execute 实现

```go
func (a *ClaudeCodeAgent) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error) {
    // 1. 构建命令参数
    args := a.buildExecuteArgs(req)
    
    // 2. 执行命令
    output, err := a.executor.Run(ctx, args)
    if err != nil {
        return nil, a.parseError(err)
    }
    
    // 3. 解析输出
    result, err := a.parser.ParseExecuteOutput(output)
    if err != nil {
        return nil, err
    }
    
    // 4. 提取变更文件
    result.FilesChanged = a.extractChangedFiles(req.WorktreePath)
    
    return result, nil
}

func (a *ClaudeCodeAgent) buildExecuteArgs(req *ExecuteRequest) []string {
    args := []string{
        "-p",                              // 非交互模式
        "--output-format", "stream-json",  // 流式 JSON 输出
        "-w", req.WorktreePath,            // Worktree 路径
        "--session-id", req.SessionID.String(), // 会话 ID
    }
    
    // 权限模式
    args = append(args, "--permission-mode", string(req.PermissionMode))
    
    // 模型
    if req.Model != "" {
        args = append(args, "--model", req.Model)
    }
    
    // 允许的工具
    if len(req.AllowedTools) > 0 {
        args = append(args, "--allowedTools")
        args = append(args, req.AllowedTools...)
    }
    
    // 预算
    if req.MaxBudgetUSD > 0 {
        args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", req.MaxBudgetUSD))
    }
    
    // 任务描述作为 prompt
    args = append(args, a.buildTaskPrompt(req))
    
    return args
}

func (a *ClaudeCodeAgent) buildTaskPrompt(req *ExecuteRequest) string {
    prompt := fmt.Sprintf(`
## 任务
%s

## 上下文
%s

## 要求
1. 实现任务描述的功能
2. 运行相关测试验证
3. 如果测试失败，尝试修复
4. 输出变更的文件列表和测试结果

## 输出格式
请以 JSON 格式输出：
{
  "success": true/false,
  "output": "...",
  "testResults": [...],
  "filesChanged": [...]
}
`, req.Task.Description, req.Context)
    
    return prompt
}
```

### 4.3 GenerateDesign 实现

```go
func (a *ClaudeCodeAgent) GenerateDesign(ctx context.Context, req *DesignRequest) (*DesignOutput, error) {
    // JSON Schema 验证输出格式
    schema := `{
        "type": "object",
        "properties": {
            "designContent": {"type": "string"},
            "complexityScore": {"type": "integer", "minimum": 0, "maximum": 100},
            "estimatedFiles": {"type": "array", "items": {"type": "string"}},
            "estimatedLOC": {"type": "integer"},
            "modulesAffected": {"type": "array", "items": {"type": "string"}},
            "breakingChanges": {"type": "boolean"},
            "testDifficulty": {"type": "string", "enum": ["simple", "medium", "complex"]}
        },
        "required": ["designContent", "complexityScore"]
    }`
    
    args := []string{
        "-p",
        "--output-format", "json",
        "--json-schema", schema,
        "-w", req.WorktreePath,
        "--session-id", req.SessionID.String(),
    }
    
    if req.Model != "" {
        args = append(args, "--model", req.Model)
    }
    
    // 使用设计 Agent
    args = append(args, "--agent", "design")
    
    // 设计 prompt
    args = append(args, a.buildDesignPrompt(req))
    
    output, err := a.executor.Run(ctx, args)
    if err != nil {
        return nil, err
    }
    
    return a.parser.ParseDesignOutput(output)
}
```

### 4.4 Clarify 实现

```go
func (a *ClaudeCodeAgent) Clarify(ctx context.Context, req *ClarifyRequest) (*ClarifyOutput, error) {
    args := []string{
        "-p",
        "--output-format", "json",
        "--session-id", req.SessionID.String(),
        "--agent", "clarification",
    }
    
    // 有新回答时恢复会话
    if req.NewAnswer != "" {
        args = append(args, "-c")  // continue
    }
    
    args = append(args, a.buildClarifyPrompt(req))
    
    output, err := a.executor.Run(ctx, args)
    if err != nil {
        return nil, err
    }
    
    return a.parser.ParseClarifyOutput(output)
}
```

### 4.5 Resume 实现

```go
func (a *ClaudeCodeAgent) Resume(ctx context.Context, req *ResumeRequest) (*ExecuteResult, error) {
    args := []string{
        "-p",
        "--output-format", "stream-json",
        "-r", req.SessionID.String(),  // resume by session ID
        "--session-id", req.SessionID.String(),
    }
    
    // 添加继续执行的指令
    args = append(args, req.Instruction)
    
    output, err := a.executor.Run(ctx, args)
    if err != nil {
        return nil, err
    }
    
    return a.parser.ParseExecuteOutput(output)
}
```

---

## 5. 命令执行器

### 5.1 CommandExecutor

```go
// internal/infrastructure/agent/claude/command_executor.go

type CommandExecutor struct {
    binaryPath string
    timeout    time.Duration
}

func NewCommandExecutor() *CommandExecutor {
    return &CommandExecutor{
        binaryPath: "claude",  // 默认 PATH 中的 claude
        timeout:   30 * time.Minute,
    }
}

func (e *CommandExecutor) Run(ctx context.Context, args []string) (string, error) {
    // 创建子进程
    cmd := exec.CommandContext(ctx, e.binaryPath, args...)
    
    // 设置工作目录（如果需要）
    if worktree := e.extractWorktree(args); worktree != "" {
        cmd.Dir = worktree
    }
    
    // 捕获输出
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    
    // 执行
    start := time.Now()
    err := cmd.Run()
    duration := time.Since(start)
    
    if err != nil {
        return "", &ExecutionError{
            Stderr:    stderr.String(),
            Duration:  duration,
            ExitCode:  cmd.ProcessState.ExitCode(),
            Inner:     err,
        }
    }
    
    return stdout.String(), nil
}

func (e *CommandExecutor) extractWorktree(args []string) string {
    for i, arg := range args {
        if arg == "-w" || arg == "--worktree" {
            if i+1 < len(args) {
                return args[i+1]
            }
        }
    }
    return ""
}
```

### 5.2 流式执行（用于实时监控）

```go
type StreamExecutor struct {
    binaryPath string
}

type StreamEvent struct {
    Type      string          `json:"type"`      // message, tool_use, tool_result, error, complete
    Data      json.RawMessage `json:"data"`
    Timestamp time.Time       `json:"timestamp"`
}

func (e *StreamExecutor) RunStream(ctx context.Context, args []string) (<-chan StreamEvent, error) {
    args = append(args, "--output-format", "stream-json")
    
    cmd := exec.CommandContext(ctx, e.binaryPath, args...)
    
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return nil, err
    }
    
    if err := cmd.Start(); err != nil {
        return nil, err
    }
    
    events := make(chan StreamEvent, 100)
    
    go func() {
        defer close(events)
        defer cmd.Wait()
        
        scanner := bufio.NewScanner(stdout)
        for scanner.Scan() {
            line := scanner.Text()
            
            var event StreamEvent
            if err := json.Unmarshal([]byte(line), &event); err == nil {
                event.Timestamp = time.Now()
                events <- event
            }
        }
    }()
    
    return events, nil
}
```

---

## 6. 输出解析器

### 6.1 OutputParser

```go
// internal/infrastructure/agent/claude/output_parser.go

type OutputParser struct{}

func NewOutputParser() *OutputParser {
    return &OutputParser{}
}

func (p *OutputParser) ParseExecuteOutput(raw string) (*ExecuteResult, error) {
    // 尝试解析 JSON
    var result ExecuteResult
    if err := json.Unmarshal([]byte(raw), &result); err == nil {
        return &result, nil
    }
    
    // 非 JSON 输出，提取关键信息
    result.Success = !strings.Contains(raw, "failed") && !strings.Contains(raw, "error")
    result.Output = raw
    
    // 提取测试结果
    result.TestResults = p.extractTestResults(raw)
    
    return &result, nil
}

func (p *OutputParser) ParseDesignOutput(raw string) (*DesignOutput, error) {
    var output DesignOutput
    if err := json.Unmarshal([]byte(raw), &output); err != nil {
        return nil, fmt.Errorf("failed to parse design output: %w", err)
    }
    return &output, nil
}

func (p *OutputParser) ParseClarifyOutput(raw string) (*ClarifyOutput, error) {
    var output ClarifyOutput
    if err := json.Unmarshal([]byte(raw), &output); err != nil {
        return nil, fmt.Errorf("failed to parse clarify output: %w", err)
    }
    return &output, nil
}

func (p *OutputParser) extractTestResults(raw string) []TestResult {
    // 解析测试输出格式
    results := []TestResult{}
    
    // 匹配常见测试框架输出
    // Go: PASS/FAIL
    // Jest: ✓/✗
    // Pytest: PASSED/FAILED
    
    lines := strings.Split(raw, "\n")
    for _, line := range lines {
        if strings.Contains(line, "PASS") || strings.Contains(line, "✓") {
            results = append(results, TestResult{
                Name:   p.extractTestName(line),
                Status: "passed",
            })
        } else if strings.Contains(line, "FAIL") || strings.Contains(line, "✗") {
            results = append(results, TestResult{
                Name:   p.extractTestName(line),
                Status: "failed",
            })
        }
    }
    
    return results
}
```

---

## 7. 工具权限配置

### 7.1 按阶段的工具权限

```go
// internal/infrastructure/agent/tool_policy.go

type ToolPolicy struct {
    AllowedTools   []string
    DisallowedTools []string
}

// 阶段 -> 工具权限映射
var StageToolPolicies = map[Stage]ToolPolicy{
    StageClarification: {
        AllowedTools: []string{"Read", "Grep", "Glob", "WebFetch"},
        DisallowedTools: []string{"Edit", "Write", "Bash"},
    },
    StageDesign: {
        AllowedTools: []string{"Read", "Grep", "Glob", "WebFetch", "WebSearch"},
        DisallowedTools: []string{"Edit", "Write", "Bash(rm:*)", "Bash(git push:*)"},
    },
    StageExecution: {
        AllowedTools: []string{"default"},  // 使用默认工具集
        DisallowedTools: []string{"Bash(git push --force:*)", "Bash(rm -rf:*)"},
    },
}

// 危险操作 -> 需审批
var DangerousOperations = map[string]bool{
    "Bash(git push --force:*)":    true,
    "Bash(git reset --hard:*)":    true,
    "Bash(git branch -D:*)":       true,
    "Bash(rm -rf:*)":              true,
}

func GetToolPolicy(stage Stage, requiresApproval bool) ToolPolicy {
    policy := StageToolPolicies[stage]
    
    if requiresApproval {
        // 执行阶段需审批时，限制更多
        policy.DisallowedTools = append(policy.DisallowedTools, 
            "Bash(git push:*)", "Bash(git commit:*)")
    }
    
    return policy
}
```

### 7.2 工具名称格式

```
工具格式: ToolName 或 ToolName(pattern:*)
示例:
- Read                     # 允许所有 Read
- Bash                     # 允许所有 Bash
- Bash(git:*)              # 允许 git 相关 Bash
- Bash(git push:*)         # 允许 git push
- Edit                     # 允许所有 Edit
- Edit(**/*.go)            # 允许编辑 .go 文件
```

---

## 8. 自定义 Agent 配置

### 8.1 Agent 定义 JSON

```go
// Agent 类型定义
var CustomAgents = map[string]AgentDefinition{
    "clarification": {
        Description: "需求澄清 Agent，分析 Issue 并提出澄清问题",
        Prompt: `
你是一个需求澄清 Agent。你的任务是：
1. 分析 GitHub Issue 内容
2. 识别模糊点、缺失信息、潜在风险
3. 提出澄清问题
4. 等待用户回答后继续分析
5. 当需求清晰时，输出确认的需求点列表

输出格式：
{
  "questions": ["问题1", "问题2"],
  "confirmedPoints": ["需求点1", "需求点2"],
  "readyForDesign": true/false
}
`,
    },
    "design": {
        Description: "设计方案 Agent，分析代码库并生成设计方案",
        Prompt: `
你是一个设计方案 Agent。你的任务是：
1. 分析相关代码模块
2. 理解需求点
3. 生成设计方案（架构、接口、数据流）
4. 评估复杂度（代码量、模块数、破坏性变更、测试难度）

输出格式：
{
  "designContent": "设计方案内容...",
  "complexityScore": 65,
  "estimatedFiles": ["file1.go", "file2.go"],
  "estimatedLOC": 200,
  "modulesAffected": ["module1", "module2"],
  "breakingChanges": false,
  "testDifficulty": "medium"
}
`,
    },
    "executor": {
        Description: "任务执行 Agent，实现具体功能并验证",
        Prompt: `
你是一个任务执行 Agent。你的任务是：
1. 根据设计方案实现具体功能
2. 运行相关测试验证
3. 如果测试失败，尝试修复
4. 输出执行结果

注意：
- 遵循设计方案
- 保持代码风格一致
- 添加必要的测试
- 不要进行破坏性操作（force push 等）
`,
    },
}

type AgentDefinition struct {
    Description string `json:"description"`
    Prompt      string `json:"prompt"`
}

// 生成 --agents JSON 参数
func BuildAgentsJSON() string {
    data, _ := json.Marshal(CustomAgents)
    return string(data)
}
```

### 8.2 使用自定义 Agent

```go
func (a *ClaudeCodeAgent) runWithAgent(ctx context.Context, agentName string, args []string, prompt string) (string, error) {
    // 方式1: 使用预配置的 agent（需在 settings.json 中配置）
    args = append(args, "--agent", agentName)
    
    // 方式2: 动态传入 agent 定义
    agentsJSON := BuildAgentsJSON()
    args = append(args, "--agents", agentsJSON)
    
    args = append(args, prompt)
    
    return a.executor.Run(ctx, args)
}
```

---

## 9. 调用流程示例

### 9.1 需求澄清阶段

```go
func (s *ClarificationService) ProcessIssue(ctx context.Context, session *WorkSession) error {
    req := &agent.ClarifyRequest{
        SessionID: session.ID,
        Issue:     session.Issue,
        History:   session.Clarification.ConversationTurns,
    }
    
    output, err := s.agentRunner.Clarify(ctx, req)
    if err != nil {
        return err
    }
    
    // 更新 Clarification 实体
    session.Clarification.PendingQuestions = output.Questions
    session.Clarification.ConfirmedPoints = output.ConfirmedPoints
    
    if output.ReadyForDesign {
        // 触发进入设计阶段
        s.eventDispatcher.Dispatch(ClarificationCompleted{SessionID: session.ID})
    }
    
    return nil
}
```

### 9.2 设计方案阶段

```go
func (s *DesignService) GenerateDesign(ctx context.Context, session *WorkSession) error {
    // 先分析代码库
    analyzeReq := &agent.AnalyzeRequest{
        WorktreePath: session.Execution.WorktreePath,
        Issue:        session.Issue,
    }
    
    analysis, err := s.agentRunner.AnalyzeCodebase(ctx, analyzeReq)
    if err != nil {
        return err
    }
    
    // 生成设计方案
    designReq := &agent.DesignRequest{
        SessionID:      session.ID,
        WorktreePath:   session.Execution.WorktreePath,
        Issue:          session.Issue,
        ClarifiedNeeds: session.Clarification.ConfirmedPoints,
        CodeAnalysis:   analysis,
        Model:          s.selectModel(session), // 按复杂度选模型
    }
    
    output, err := s.agentRunner.GenerateDesign(ctx, designReq)
    if err != nil {
        return err
    }
    
    // 创建 Design 实体和版本
    session.Design.CreateVersion(output.DesignContent, "初始设计")
    session.Design.ComplexityScore = output.ComplexityScore
    
    // 判断是否需要人工确认
    if output.ComplexityScore > s.config.ComplexityThreshold || s.config.ForceDesignConfirm {
        session.Design.RequireConfirmation = true
        // 发布等待审批事件
        s.eventDispatcher.Dispatch(DesignCreated{SessionID: session.ID})
    } else {
        // 自动批准
        session.Design.Confirm()
        s.eventDispatcher.Dispatch(DesignApproved{SessionID: session.ID})
    }
    
    return nil
}
```

### 9.3 任务执行阶段

```go
func (s *TaskService) ExecuteTask(ctx context.Context, session *WorkSession, task *Task) error {
    // 获取工具权限策略
    policy := toolPolicy.GetToolPolicy(StageExecution, task.RequiresApproval)
    
    req := &agent.ExecuteRequest{
        SessionID:      session.ID,
        WorktreePath:   session.Execution.WorktreePath,
        Task:           task,
        PermissionMode: agent.PermissionAcceptEdits, // 自动接受编辑
        AllowedTools:   policy.AllowedTools,
        MaxBudgetUSD:   s.config.MaxBudgetPerTask,
        Model:          s.config.DefaultModel,
        Context:        s.buildContext(session),
    }
    
    // 流式执行，实时监控
    events, err := s.streamExecutor.RunStream(ctx, s.buildArgs(req))
    if err != nil {
        return err
    }
    
    // 处理事件流
    for event := range events {
        s.handleEvent(session, task, event)
    }
    
    return nil
}

func (s *TaskService) handleEvent(session *WorkSession, task *Task, event StreamEvent) {
    switch event.Type {
    case "tool_use":
        // 记录工具使用
        s.logger.Info("tool used", zap.String("tool", event.Data))
        
    case "message":
        // 更新进度
        s.websocketPusher.Push(session.ID, event)
        
    case "error":
        // 处理错误
        task.MarkFailed(event.Data)
        s.eventDispatcher.Dispatch(TaskFailed{SessionID: session.ID, TaskID: task.ID})
        
    case "complete":
        // 完成
        task.MarkCompleted()
        s.eventDispatcher.Dispatch(TaskCompleted{SessionID: session.ID, TaskID: task.ID})
    }
}
```

### 9.4 恢复中断任务

```go
func (s *RecoveryService) ResumeSession(ctx context.Context, sessionID uuid.UUID) error {
    session, err := s.sessionRepo.FindById(ctx, sessionID)
    if err != nil {
        return err
    }
    
    req := &agent.ResumeRequest{
        SessionID:   sessionID,
        Instruction: "继续执行中断的任务",
    }
    
    result, err := s.agentRunner.Resume(ctx, req)
    if err != nil {
        return err
    }
    
    // 根据结果更新状态
    if result.Success {
        session.CurrentTask.MarkCompleted()
    } else {
        session.CurrentTask.MarkFailed(result.Error.Message, result.Error.Suggestion)
    }
    
    return s.sessionRepo.Save(ctx, session)
}
```

---

## 10. 错误处理

### 10.1 错误类型映射

```go
func (a *ClaudeCodeAgent) parseError(err error) *AgentError {
    execErr, ok := err.(*ExecutionError)
    if !ok {
        return &AgentError{
            Type:      ErrorTypeUnknown,
            Message:   err.Error(),
            Retryable: false,
        }
    }
    
    // 解析 stderr 获取具体错误类型
    stderr := execErr.Stderr
    
    if strings.Contains(stderr, "test failed") || strings.Contains(stderr, "FAIL") {
        return &AgentError{
            Type:       ErrorTypeTestFailed,
            Message:    "测试失败",
            Suggestion: a.extractTestFixSuggestion(stderr),
            Retryable:  true,
        }
    }
    
    if strings.Contains(stderr, "budget exceeded") {
        return &AgentError{
            Type:       ErrorTypeBudgetExceeded,
            Message:    "预算超限",
            Suggestion: "增加预算或使用更便宜的模型",
            Retryable:  false,
        }
    }
    
    if strings.Contains(stderr, "permission denied") {
        return &AgentError{
            Type:       ErrorTypePermissionDenied,
            Message:    "权限被拒绝",
            Suggestion: "检查工具权限配置",
            Retryable:  false,
        }
    }
    
    if execErr.ExitCode == -1 || strings.Contains(stderr, "timeout") {
        return &AgentError{
            Type:       ErrorTypeTimeout,
            Message:    "执行超时",
            Suggestion: "增加超时时间或简化任务",
            Retryable:  true,
        }
    }
    
    return &AgentError{
        Type:      ErrorTypeAPIError,
        Message:   stderr,
        Retryable: false,
    }
}
```

### 10.2 重试策略

```go
type RetryStrategy struct {
    MaxRetries   int
    Backoff      time.Duration
    MaxBackoff   time.Duration
    RetryableErrors []ErrorType
}

func DefaultRetryStrategy() *RetryStrategy {
    return &RetryStrategy{
        MaxRetries:   3,
        Backoff:      5 * time.Second,
        MaxBackoff:   60 * time.Second,
        RetryableErrors: []ErrorType{
            ErrorTypeTestFailed,
            ErrorTypeTimeout,
            ErrorTypeAPIError,
        },
    }
}

func (s *RetryStrategy) ShouldRetry(err *AgentError, attempt int) bool {
    if attempt >= s.MaxRetries {
        return false
    }
    
    for _, retryable := range s.RetryableErrors {
        if err.Type == retryable {
            return true
        }
    }
    
    return false
}

func (s *RetryStrategy) GetBackoff(attempt int) time.Duration {
    backoff := s.Backoff * time.Duration(1 << attempt)
    if backoff > s.MaxBackoff {
        backoff = s.MaxBackoff
    }
    return backoff
}
```

### 10.3 错误处理决策流程

```
错误发生
    │
    ▼
错误分类 (ErrorType)
    │
    ├──────────────┬──────────────┬──────────────┐
    │              │              │              │
    ▼              ▼              ▼              ▼
  L1 Critical    L2 High       L3 Medium      L4 Low
    │              │              │              │
    ▼              ▼              ▼              ▼
系统告警        进入           自动处理        自动重试
自动修复        ErrorRecovery   重试/降级       静默处理
    │              │              │              │
    ▼              ▼              │              │
失败→人工      自动恢复尝试     │              │
                │              │              │
                ├────┬────┤    │              │
                │    │    │    │              │
               成功 失败 超时   │              │
                │    │    │    │              │
                ▼    ▼    ▼    │              │
              继续 Paused Paused│             │
                    │    │      │              │
                    ▼    ▼      │              │
                 通知管理员    │              │
                             │              │
                             ▼              ▼
                          继续执行        继续执行
```

### 10.4 错误类型处理策略

| ErrorType | Severity | Recovery | MaxRetries | Backoff | NotifyTiming |
|-----------|----------|----------|------------|---------|--------------|
| test_failed | Medium | Auto | 3 | 指数退避 | 重试耗尽后 |
| build_failed | Medium | Auto | 3 | 指数退避 | 重试耗尽后 |
| timeout | Medium | Auto | 2 | 固定30s | 重试耗尽后 |
| agent_crashed | High | SemiAuto | 1 | 不等待 | 立即 |
| session_lost | High | Manual | 0 | - | 立即 |
| budget_exceeded | High | SemiAuto | 0 | - | 立即 |
| github_rate_limit | Medium | Auto | N/A | 等待重置 | 等待中 |
| network_error | Medium | Auto | 3 | 指数退避 | 重试耗尽后 |
| test_env_unavailable | Medium | SemiAuto | 5 | 固定5min | 检测失败后 |
| worktree_error | Medium | Auto | 2 | 固定10s | 重试耗尽后 |
| permission_denied | High | Manual | 0 | - | 立即 |
| tool_blocked | Medium | Manual | 0 | - | 立即 |

### 10.5 网络错误重试策略

```go
// NetworkErrorType 网络错误类型
type NetworkErrorType string

const (
    NetworkErrorTimeout    NetworkErrorType = "timeout"
    NetworkErrorConnection NetworkErrorType = "connection"
    NetworkErrorReset      NetworkErrorType = "reset"
    NetworkErrorDNS        NetworkErrorType = "dns"
)

// NetworkRetryStrategy 网络错误重试策略
type NetworkRetryStrategy struct {
    MaxRetries        int
    InitialBackoff    time.Duration
    MaxBackoff        time.Duration
    BackoffMultiplier float64
    RetryableTypes    []NetworkErrorType
}

func DefaultNetworkRetryStrategy() *NetworkRetryStrategy {
    return &NetworkRetryStrategy{
        MaxRetries:        3,
        InitialBackoff:    5 * time.Second,
        MaxBackoff:        60 * time.Second,
        BackoffMultiplier: 2.0,
        RetryableTypes: []NetworkErrorType{
            NetworkErrorTimeout,
            NetworkErrorConnection,
            NetworkErrorReset,
        },
    }
}

func (s *NetworkRetryStrategy) ShouldRetry(errType NetworkErrorType) bool {
    for _, t := range s.RetryableTypes {
        if t == errType {
            return true
        }
    }
    return false
}

func (s *NetworkRetryStrategy) GetBackoff(attempt int) time.Duration {
    backoff := s.InitialBackoff * time.Duration(1<<uint(attempt))
    if backoff > s.MaxBackoff {
        backoff = s.MaxBackoff
    }
    return backoff
}
```

### 10.6 GitHub 限流处理策略

```go
// RateLimitStrategy GitHub 限流处理策略
type RateLimitStrategy struct {
    WaitEnabled      bool
    MaxWaitDuration  time.Duration
    NotifyThreshold  int  // 剩余百分比阈值
    FallbackEnabled  bool
}

func DefaultRateLimitStrategy() *RateLimitStrategy {
    return &RateLimitStrategy{
        WaitEnabled:      true,
        MaxWaitDuration:  30 * time.Minute,
        NotifyThreshold:  10,
        FallbackEnabled:  true,
    }
}

// RateLimitHandler 限流处理器
type RateLimitHandler struct {
    strategy  *RateLimitStrategy
    notifier  NotificationService
    limiter   *rate.Limiter
}

func (h *RateLimitHandler) HandleRateLimit(ctx context.Context, resetTime time.Time) error {
    if !h.strategy.WaitEnabled {
        return errors.New("rate limit encountered, waiting disabled")
    }

    waitDuration := time.Until(resetTime)
    if waitDuration > h.strategy.MaxWaitDuration {
        return fmt.Errorf("rate limit wait time %v exceeds max %v",
            waitDuration, h.strategy.MaxWaitDuration)
    }

    // 通知管理员等待中
    h.notifier.Notify(Notification{
        Type:    "rate_limit",
        Level:   "warning",
        Message: fmt.Sprintf("GitHub API 限流，等待 %v 后重试", waitDuration),
    })

    // 等待限流重置
    select {
    case <-time.After(waitDuration):
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (h *RateLimitHandler) CheckRateLimit(remaining, limit int) bool {
    if limit == 0 {
        return false
    }
    percentage := (remaining * 100) / limit
    return percentage <= h.strategy.NotifyThreshold
}
```

### 10.7 增强版错误解析器

```go
// EnhancedError 增强版错误结构
type EnhancedError struct {
    Type        ErrorType
    Severity    SeverityLevel
    Recovery    RecoveryCategory
    Message     string
    Suggestion  string
    Retryable   bool
    MaxRetries  int
    Backoff     time.Duration
    Context     map[string]interface{} // 额外上下文
    Cause       error                  // 原始错误
    Timestamp   time.Time
}

func (a *ClaudeCodeAgent) parseErrorEnhanced(err error) *EnhancedError {
    execErr, ok := err.(*ExecutionError)
    if !ok {
        return &EnhancedError{
            Type:       ErrorTypeUnknown,
            Severity:   SeverityHigh,
            Recovery:   RecoveryManual,
            Message:    err.Error(),
            Retryable:  false,
            Timestamp:  time.Now(),
        }
    }

    stderr := execErr.Stderr

    // 测试失败
    if strings.Contains(stderr, "test failed") || strings.Contains(stderr, "FAIL") {
        return &EnhancedError{
            Type:       ErrorTypeTestFailed,
            Severity:   SeverityMedium,
            Recovery:   RecoveryAuto,
            Message:    "测试失败",
            Suggestion: a.extractTestFixSuggestion(stderr),
            Retryable:  true,
            MaxRetries: 3,
            Backoff:    5 * time.Second,
            Context:    map[string]interface{}{"test_output": stderr},
            Timestamp:  time.Now(),
        }
    }

    // 构建失败
    if strings.Contains(stderr, "build failed") || strings.Contains(stderr, "compilation error") {
        return &EnhancedError{
            Type:       ErrorTypeBuildFailed,
            Severity:   SeverityMedium,
            Recovery:   RecoveryAuto,
            Message:    "构建失败",
            Suggestion: "检查编译错误并修复",
            Retryable:  true,
            MaxRetries: 3,
            Backoff:    5 * time.Second,
            Timestamp:  time.Now(),
        }
    }

    // GitHub 限流
    if strings.Contains(stderr, "rate limit") || strings.Contains(stderr, "403") {
        resetTime := a.parseRateLimitReset(stderr)
        return &EnhancedError{
            Type:       ErrorTypeGitHubRateLimit,
            Severity:   SeverityMedium,
            Recovery:   RecoveryAuto,
            Message:    "GitHub API 限流",
            Suggestion: "等待限流重置后重试",
            Retryable:  true,
            MaxRetries: 0, // 使用等待策略而非重试
            Context:    map[string]interface{}{"reset_time": resetTime},
            Timestamp:  time.Now(),
        }
    }

    // 网络错误
    if isNetworkError(stderr) {
        return &EnhancedError{
            Type:       ErrorTypeNetworkError,
            Severity:   SeverityMedium,
            Recovery:   RecoveryAuto,
            Message:    "网络连接错误",
            Suggestion: "检查网络连接后重试",
            Retryable:  true,
            MaxRetries: 3,
            Backoff:    5 * time.Second,
            Timestamp:  time.Now(),
        }
    }

    // Agent 进程崩溃
    if execErr.ExitCode == -1 || strings.Contains(stderr, "signal") {
        return &EnhancedError{
            Type:       ErrorTypeAgentCrashed,
            Severity:   SeverityHigh,
            Recovery:   RecoverySemiAuto,
            Message:    "Agent 进程异常终止",
            Suggestion: "尝试重新启动任务",
            Retryable:  true,
            MaxRetries: 1,
            Backoff:    0, // 立即重试
            Timestamp:  time.Now(),
        }
    }

    // 预算超限
    if strings.Contains(stderr, "budget exceeded") {
        return &EnhancedError{
            Type:       ErrorTypeBudgetExceeded,
            Severity:   SeverityHigh,
            Recovery:   RecoverySemiAuto,
            Message:    "预算超限",
            Suggestion: "增加预算或使用更便宜的模型",
            Retryable:  false,
            Timestamp:  time.Now(),
        }
    }

    // 会话丢失
    if strings.Contains(stderr, "session not found") || strings.Contains(stderr, "session expired") {
        return &EnhancedError{
            Type:       ErrorTypeSessionLost,
            Severity:   SeverityHigh,
            Recovery:   RecoveryManual,
            Message:    "会话上下文丢失",
            Suggestion: "需要重新开始任务",
            Retryable:  false,
            Timestamp:  time.Now(),
        }
    }

    // 权限被拒绝
    if strings.Contains(stderr, "permission denied") {
        return &EnhancedError{
            Type:       ErrorTypePermissionDenied,
            Severity:   SeverityHigh,
            Recovery:   RecoveryManual,
            Message:    "权限被拒绝",
            Suggestion: "检查工具权限配置",
            Retryable:  false,
            Timestamp:  time.Now(),
        }
    }

    // 超时
    if execErr.ExitCode == -1 || strings.Contains(stderr, "timeout") {
        return &EnhancedError{
            Type:       ErrorTypeTimeout,
            Severity:   SeverityMedium,
            Recovery:   RecoveryAuto,
            Message:    "执行超时",
            Suggestion: "增加超时时间或简化任务",
            Retryable:  true,
            MaxRetries: 2,
            Backoff:    30 * time.Second,
            Timestamp:  time.Now(),
        }
    }

    return &EnhancedError{
        Type:       ErrorTypeAPIError,
        Severity:   SeverityHigh,
        Recovery:   RecoveryManual,
        Message:    stderr,
        Retryable:  false,
        Timestamp:  time.Now(),
    }
}

func isNetworkError(stderr string) bool {
    networkIndicators := []string{
        "connection refused",
        "connection reset",
        "network is unreachable",
        "no such host",
        "i/o timeout",
        "EOF",
    }
    for _, indicator := range networkIndicators {
        if strings.Contains(strings.ToLower(stderr), indicator) {
            return true
        }
    }
    return false
}
```

---

## 10.1 审计日志记录器

### 10.1.1 AuditLogger 结构

```go
// internal/infrastructure/audit/audit_logger.go

// AuditLogger 审计日志记录器
type AuditLogger struct {
    repo   AuditLogRepository
    config AuditConfig
}

// AuditConfig 审计配置
type AuditConfig struct {
    Enabled           bool
    RetentionDays     int
    MaxOutputLength   int
    SensitiveOps      map[string]bool
}

// AuditEntry 审计日志条目
type AuditEntry struct {
    Timestamp    time.Time
    SessionID    uuid.UUID
    Repository   string
    IssueNumber  int
    Actor        string
    ActorRole    string
    Operation    AuditOperation
    ResourceType string
    ResourceID   string
    Parameters   map[string]any
    Result       AuditResult
    Duration     time.Duration
    Output       string
    Error        error
}

// AuditOperation 审计操作类型
type AuditOperation string

const (
    AuditOpSessionStart     AuditOperation = "session_start"
    AuditOpSessionPause     AuditOperation = "session_pause"
    AuditOpSessionResume    AuditOperation = "session_resume"
    AuditOpSessionTerminate AuditOperation = "session_terminate"
    AuditOpStageTransition  AuditOperation = "stage_transition"
    AuditOpAgentCall        AuditOperation = "agent_call"
    AuditOpToolUse          AuditOperation = "tool_use"
    AuditOpFileRead         AuditOperation = "file_read"
    AuditOpFileWrite        AuditOperation = "file_write"
    AuditOpBashExecute      AuditOperation = "bash_execute"
    AuditOpGitOperation     AuditOperation = "git_operation"
    AuditOpPRCreate         AuditOperation = "pr_create"
    AuditOpApprovalRequest  AuditOperation = "approval_request"
    AuditOpApprovalDecision AuditOperation = "approval_decision"
)

// AuditResult 审计结果
type AuditResult string

const (
    AuditResultSuccess AuditResult = "success"
    AuditResultFailed  AuditResult = "failed"
    AuditResultDenied  AuditResult = "denied"
)
```

### 10.1.2 Record 方法

```go
// Record 记录审计日志
func (l *AuditLogger) Record(ctx context.Context, entry *AuditEntry) error {
    if !l.config.Enabled {
        return nil
    }

    // 截断输出
    if len(entry.Output) > l.config.MaxOutputLength {
        entry.Output = entry.Output[:l.config.MaxOutputLength] + "...[truncated]"
    }

    // 构建数据库模型
    model := &AuditLogModel{
        Timestamp:    entry.Timestamp,
        SessionID:    entry.SessionID,
        Repository:   entry.Repository,
        IssueNumber:  entry.IssueNumber,
        Actor:        entry.Actor,
        ActorRole:    entry.ActorRole,
        Operation:    string(entry.Operation),
        ResourceType: entry.ResourceType,
        ResourceID:   entry.ResourceID,
        Parameters:   entry.Parameters,
        Result:       string(entry.Result),
        DurationMs:   entry.Duration.Milliseconds(),
        Output:       entry.Output,
    }

    // 设置错误信息
    if entry.Error != nil {
        model.ErrorMessage = entry.Error.Error()
    }

    return l.repo.Save(ctx, model)
}

// RecordWithContext 便捷方法：从上下文构建并记录审计日志
func (l *AuditLogger) RecordWithContext(
    ctx context.Context,
    session *WorkSession,
    operation AuditOperation,
    result AuditResult,
    duration time.Duration,
    output string,
    err error,
) error {
    entry := &AuditEntry{
        Timestamp:    time.Now(),
        SessionID:    session.ID,
        Repository:   session.Issue.Repository,
        IssueNumber:  session.Issue.Number,
        Actor:        session.CurrentActor,
        ActorRole:    session.CurrentActorRole,
        Operation:    operation,
        Result:       result,
        Duration:     duration,
        Output:       output,
        Error:        err,
    }

    return l.Record(ctx, entry)
}
```

### 10.1.3 在 Agent 执行中使用审计日志

```go
// Execute 带审计日志的任务执行
func (a *ClaudeCodeAgent) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error) {
    start := time.Now()

    // 记录开始
    a.auditLogger.Record(ctx, &AuditEntry{
        SessionID:  req.SessionID,
        Operation:  AuditOpAgentCall,
        Actor:      req.Actor,
        ActorRole:  req.ActorRole,
        Result:     AuditResultSuccess,
        Parameters: map[string]any{
            "task":    req.Task.Description,
            "model":   req.Model,
            "tools":   req.AllowedTools,
        },
    })

    // 执行任务
    result, err := a.executeInternal(ctx, req)

    // 记录结果
    duration := time.Since(start)
    auditResult := AuditResultSuccess
    if err != nil {
        auditResult = AuditResultFailed
    }

    a.auditLogger.RecordWithContext(ctx, req.Session, AuditOpAgentCall, auditResult, duration, result.Output, err)

    return result, err
}
```

### 10.1.4 审计日志仓库接口

```go
// internal/domain/repository/audit_log_repository.go

type AuditLogRepository interface {
    // Save 保存审计日志
    Save(ctx context.Context, log *AuditLogModel) error

    // FindByID 根据 ID 查询
    FindByID(ctx context.Context, id uuid.UUID) (*AuditLogModel, error)

    // FindBySessionID 根据会话 ID 查询
    FindBySessionID(ctx context.Context, sessionID uuid.UUID, opts *QueryOptions) ([]*AuditLogModel, int64, error)

    // FindByRepository 根据仓库查询
    FindByRepository(ctx context.Context, repository string, opts *QueryOptions) ([]*AuditLogModel, int64, error)

    // Query 通用查询
    Query(ctx context.Context, opts *AuditLogQueryOptions) ([]*AuditLogModel, int64, error)

    // DeleteBefore 删除指定时间之前的日志
    DeleteBefore(ctx context.Context, before time.Time) (int64, error)
}

// AuditLogQueryOptions 查询选项
type AuditLogQueryOptions struct {
    StartTime   *time.Time
    EndTime     *time.Time
    SessionID   *uuid.UUID
    Repository  string
    Operation   string
    Actor       string
    Result      string
    Page        int
    PageSize    int
    OrderBy     string
    OrderDesc   bool
}
```

---

## 11. 配置示例

### 11.1 settings.json 配置

> **说明**：此配置与 `architecture-design.md` 第 8 节配置管理保持一致。

```json
{
  "agent": {
    "defaultModel": "claude-sonnet-4-6",
    "fallbackModel": "claude-haiku-4-5",
    "maxBudgetPerTask": 5.0,
    "defaultTimeout": "30m",
    "maxConcurrency": 3,
    "taskRetryLimit": 3,
    "approvalTimeout": "24h",
    "dangerousOperations": [
      "Bash(git push --force:*)",
      "Bash(git reset --hard:*)",
      "Bash(rm -rf:*)"
    ]
  },
  "clarity": {
    "threshold": 60,
    "autoProceedThreshold": 80,
    "forceClarifyThreshold": 40
  },
  "complexity": {
    "threshold": 70,
    "forceDesignConfirm": false,
    "highComplexityModel": "claude-opus-4-6",
    "lowComplexityModel": "claude-haiku-4-5"
  },
  "retry": {
    "maxRetries": 3,
    "backoff": "5s",
    "maxBackoff": "60s"
  },
  "storage": {
    "agentCacheDir": ".litchi",
    "syncStrategy": "double-write"
  },
  "customAgents": {
    "clarification": {
      "description": "需求澄清 Agent",
      "prompt": "..."
    },
    "design": {
      "description": "设计方案 Agent",
      "prompt": "..."
    },
    "executor": {
      "description": "任务执行 Agent",
      "prompt": "..."
    }
  }
}
```

### 11.2 MCP 配置（可选扩展）

```json
{
  "mcpServers": {
    "github": {
      "command": "mcp-github",
      "env": {
        "GITHUB_TOKEN": "${GITHUB_TOKEN}"
      }
    },
    "filesystem": {
      "command": "mcp-filesystem",
      "args": ["--root", "/repos"]
    }
  }
}
```

---

## 12. Agent 输出存储

### 12.1 文件缓存目录

Agent 执行时的输入输出缓存存储在 `.litchi/issues/{issue-id}/` 目录：

```
.litchi/
└── issues/
    └── {issue-id}/
        ├── designs/
        │   ├── v1.md        # 设计方案版本
        │   ├── v2.md
        │   └── ...
        ├── tasks.md         # 任务列表
        └── context.json     # Agent 执行上下文
```

### 12.2 与数据库的关系

| 存储 | 位置 | 内容 | 更新时机 |
|------|------|------|---------|
| 数据库 | PostgreSQL | WorkSession 聚合状态 | 每次状态变更 |
| 文件缓存 | `.litchi/` | Agent 输入输出 | 每次状态变更（双写） |

**双写策略**：
1. 先更新数据库（事务提交）
2. 再更新文件缓存（异步或同步）
3. 服务恢复时以数据库为准，验证文件一致性

---

## 13. 总结

本文档定义了完整的 Agent 调用层设计，包括：

1. **CLI 参数映射**：将 Claude Code CLI 参数映射到 Go 结构
2. **接口定义**：统一的 AgentRunner 接口
3. **实现类**：ClaudeCodeAgent 具体实现
4. **工具权限**：按阶段控制工具使用
5. **自定义 Agent**：澄清/设计/执行三种 Agent 配置
6. **错误处理**：错误类型识别和重试策略
7. **配置示例**：完整的 JSON 配置格式

此设计可与架构设计文档中的 Agent 抽象层无缝对接。
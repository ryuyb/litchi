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

// AgentStage represents the execution stage type for Agent operations.
type AgentStage string

const (
    AgentStageClarification  AgentStage = "clarification"
    AgentStageDesign         AgentStage = "design"
    AgentStageTaskBreakdown  AgentStage = "task_breakdown"
    AgentStageTaskExecution  AgentStage = "task_execution"
    AgentStagePRCreation     AgentStage = "pr_creation"
)

type AgentRunner interface {
    // Execute executes an Agent task and returns the result.
    Execute(ctx context.Context, req *AgentRequest) (*AgentResponse, error)
    
    // ExecuteWithRetry executes a task with automatic retry on failure.
    ExecuteWithRetry(ctx context.Context, req *AgentRequest, policy RetryPolicy) (*AgentResponse, error)
    
    // ValidateRequest validates the request parameters.
    ValidateRequest(req *AgentRequest) error
    
    // PrepareContext prepares execution context from cache.
    PrepareContext(ctx context.Context, sessionID uuid.UUID, worktreePath string) (*AgentContext, error)
    
    // SaveContext saves execution context to cache.
    SaveContext(ctx context.Context, worktreePath string, cache *AgentContextCache) error
    
    // Cancel cancels the running execution for a session.
    Cancel(sessionID uuid.UUID) error
    
    // GetStatus retrieves the current execution status.
    GetStatus(sessionID uuid.UUID) (*AgentStatus, error)
    
    // IsRunning checks if Agent is currently executing for a session.
    IsRunning(sessionID uuid.UUID) bool
    
    // Shutdown gracefully shuts down the executor and cleans up resources.
    Shutdown(ctx context.Context) error
}
```

**设计理念**：

采用通用 `Execute` 方法 + `AgentStage` 参数的设计，而非按阶段细分具体方法（如 `Clarify`, `GenerateDesign` 等）。这种设计的优势：

1. **灵活性**：新增阶段只需添加常量，无需修改接口
2. **一致性**：所有阶段使用统一的请求/响应结构，便于中间件和拦截器处理
3. **可扩展性**：通过 `AgentRequest.Context` 传递阶段特定的上下文，避免接口膨胀

**接口方法说明**：

| 方法 | 用途 | 调用时机 |
|------|------|---------|
| Execute | 执行 Agent 任务 | 各阶段核心执行 |
| ExecuteWithRetry | 带重试的执行 | 需要自动重试的场景 |
| ValidateRequest | 验证请求参数 | 执行前校验 |
| PrepareContext | 从缓存准备上下文 | 恢复执行、上下文加载 |
| SaveContext | 保存上下文到缓存 | 状态持久化 |
| Cancel | 取消正在执行的任务 | 用户取消、超时 |
| GetStatus | 获取执行状态 | 状态查询、监控 |
| IsRunning | 检查是否在执行 | 并发控制 |
| Shutdown | 关闭执行器 | 服务停止 |

### 3.2 请求/响应结构

```go
// AgentRequest - Agent 执行请求
type AgentRequest struct {
    SessionID      uuid.UUID       // WorkSession ID，用于 --session-id
    Stage          AgentStage      // 执行阶段
    WorktreePath   string          // Git Worktree 路径
    Prompt         string          // 执行提示/任务描述
    Context        *AgentContext   // 执行上下文
    Timeout        time.Duration   // 执行超时
    AllowedTools   []string        // 允许的工具
    MaxRetries     int             // 最大重试次数
}

// AgentContext - 执行上下文
type AgentContext struct {
    IssueTitle      string          // Issue 标题
    IssueBody       string          // Issue 内容
    Repository      string          // 仓库名称
    Branch          string          // 当前分支
    DesignContent   string          // 设计文档内容
    Tasks           []TaskContext   // 任务列表上下文
    ClarifiedPoints []string        // 已澄清的需求点
    History         []HistoryEntry  // 执行历史
}

// TaskContext - 任务上下文
type TaskContext struct {
    ID           uuid.UUID   // 任务 ID
    Description  string      // 任务描述
    Status       string      // 任务状态
    Dependencies []uuid.UUID // 依赖任务 ID
}

// HistoryEntry - 执行历史条目
type HistoryEntry struct {
    Timestamp time.Time  // 时间戳
    Stage     AgentStage // 执行阶段
    Action    string    // 执行动作
    Result    string    // 执行结果
}

// AgentResponse - Agent 执行响应
type AgentResponse struct {
    SessionID     uuid.UUID        // WorkSession ID
    Stage         AgentStage       // 执行阶段
    Success       bool             // 是否成功
    Output        string           // 原始输出
    Result        AgentResult      // 结构化结果
    Duration      time.Duration    // 执行时长
    TokensUsed    int              // Token 使用量
    ToolCalls     []ToolCallRecord // 工具调用记录
    Error         *AgentErrorInfo  // 错误信息
    NeedsApproval bool             // 是否需要审批
}

// AgentResult - 结构化结果
type AgentResult struct {
    Type           string            // 结果类型
    Content        string            // 结果内容
    StructuredData map[string]any    // 结构化数据
    FilesChanged   []FileChange      // 文件变更
    TestsRun       []TestResult      // 测试结果
}

// FileChange - 文件变更记录
type FileChange struct {
    Path         string // 文件路径
    Action       string // 操作类型 (create, modify, delete)
    LinesAdded   int    // 新增行数
    LinesDeleted int    // 删除行数
}

// TestResult - 测试结果
type TestResult struct {
    Name     string        // 测试名称
    Status   string        // 测试状态 (passed, failed, skipped)
    Message  string        // 错误消息
    Duration time.Duration // 执行时长
}

// ToolCallRecord - 工具调用记录
type ToolCallRecord struct {
    Timestamp   time.Time // 调用时间
    ToolName    string    // 工具名称
    Input       string    // 输入参数
    Output      string    // 输出结果
    Success     bool      // 是否成功
    Blocked     bool      // 是否被拦截
    BlockReason string    // 拦截原因
}

// AgentErrorInfo - 错误信息
type AgentErrorInfo struct {
    Code        string // 错误码
    Category    string // 错误类别
    Message     string // 错误消息
    Detail      string // 详细信息
    Recoverable bool   // 是否可恢复
    Retryable   bool   // 是否可重试
    RetryCount  int    // 重试次数
}

// AgentStatus - 执行状态
type AgentStatus struct {
    SessionID    uuid.UUID  // WorkSession ID
    Status       string     // 状态 (idle, running, paused, cancelled, completed, failed)
    CurrentStage AgentStage // 当前阶段
    StartTime    time.Time  // 开始时间
    Progress     float64    // 进度 (0-100)
    Message      string     // 状态消息
}

// AgentContextCache - 上下文缓存（用于持久化）
type AgentContextCache struct {
    SessionID        uuid.UUID   // WorkSession ID
    CurrentStage     string      // 当前阶段
    Status           string      // 会话状态
    PauseReason      *string     // 暂停原因
    ClarifiedPoints  []string    // 已澄清需求点
    DesignVersion    int         // 设计版本
    ComplexityScore  *int        // 复杂度评分
    CurrentTaskID    *uuid.UUID  // 当前任务 ID
    CompletedTaskIDs []uuid.UUID // 已完成任务 ID
    Branch           string      // 分支名
    WorktreePath     string      // Worktree 路径
    UpdatedAt        time.Time   // 更新时间
}
```

### 3.3 按阶段的使用方式

虽然使用通用 `Execute` 方法，但不同阶段的调用参数有所不同：

| 阶段 | Stage 参数 | Prompt 内容 | Context 关键字段 |
|------|-----------|-------------|-----------------|
| Clarification | `clarification` | Issue 分析提示 | IssueTitle, IssueBody |
| Design | `design` | 设计生成提示 | IssueTitle, ClarifiedPoints |
| TaskBreakdown | `task_breakdown` | 任务拆分提示 | DesignContent |
| Execution | `task_execution` | 任务执行提示 | DesignContent, Tasks |
| PRCreation | `pr_creation` | PR 创建提示 | Branch, Tasks |

**调用示例**：

```go
// 澄清阶段
clarifyReq := &service.AgentRequest{
    SessionID:    session.ID,
    Stage:        service.AgentStageClarification,
    WorktreePath: session.Execution.WorktreePath,
    Prompt:       "分析以下 Issue 并提出澄清问题...",
    Context: &service.AgentContext{
        IssueTitle: session.Issue.Title,
        IssueBody:  session.Issue.Body,
    },
    Timeout: 10 * time.Minute,
}

// 设计阶段
designReq := &service.AgentRequest{
    SessionID:    session.ID,
    Stage:        service.AgentStageDesign,
    WorktreePath: session.Execution.WorktreePath,
    Prompt:       "根据已澄清的需求生成设计方案...",
    Context: &service.AgentContext{
        IssueTitle:      session.Issue.Title,
        ClarifiedPoints: session.Clarification.ConfirmedPoints,
    },
    Timeout: 15 * time.Minute,
}

// 执行阶段
execReq := &service.AgentRequest{
    SessionID:    session.ID,
    Stage:        service.AgentStageTaskExecution,
    WorktreePath: session.Execution.WorktreePath,
    Prompt:       "实现以下任务...",
    Context: &service.AgentContext{
        DesignContent: session.Design.CurrentVersion.Content,
        Tasks:         taskContexts,
    },
    Timeout:      30 * time.Minute,
    AllowedTools: []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"},
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
func (a *ClaudeCodeAgent) Execute(ctx context.Context, req *service.AgentRequest) (*service.AgentResponse, error) {
    // 1. 验证请求
    if err := a.ValidateRequest(req); err != nil {
        return nil, err
    }
    
    // 2. 检查是否已在执行
    if a.IsRunning(req.SessionID) {
        return nil, errors.New(errors.ErrAgentAlreadyRunning)
    }
    
    // 3. 获取允许的工具（根据阶段）
    if len(req.AllowedTools) == 0 {
        req.AllowedTools = a.permissionCtrl.GetAllowedTools(req.Stage)
    }
    
    // 4. 构建命令
    cmd := a.commandBuilder.BuildCommand(req)
    
    // 5. 执行进程
    result, err := a.processExecutor.Execute(ctx, cmd, req.SessionID)
    if err != nil {
        return a.buildErrorResponse(req, result, err), err
    }
    
    // 6. 解析输出
    response, err := a.outputParser.Parse(result.Stdout, req.Stage)
    if err != nil {
        // 解析失败，返回原始输出
        response = &service.AgentResponse{
            SessionID: req.SessionID,
            Stage:     req.Stage,
            Success:   result.ExitCode == 0,
            Output:    result.Stdout,
            Duration:  result.Duration,
        }
    }
    
    response.SessionID = req.SessionID
    response.Stage = req.Stage
    response.Duration = result.Duration
    
    return response, nil
}

func (a *ClaudeCodeAgent) buildExecuteArgs(req *service.AgentRequest) []string {
    args := []string{
        "-p",                              // 非交互模式
        "--output-format", "json",         // JSON 输出
        "-w", req.WorktreePath,            // Worktree 路径
        "--session-id", req.SessionID.String(), // 会话 ID
    }
    
    // 允许的工具
    if len(req.AllowedTools) > 0 {
        args = append(args, "--allowedTools")
        args = append(args, req.AllowedTools...)
    }
    
    // 禁止危险工具
    args = append(args, "--disallowedTools")
    args = append(args, permission.DefaultDangerousTools...)
    
    // 任务描述作为 prompt
    args = append(args, req.Prompt)
    
    return args
}
```

### 4.3 ExecuteWithRetry 实现

```go
func (a *ClaudeCodeAgent) ExecuteWithRetry(
    ctx context.Context, 
    req *service.AgentRequest, 
    policy valueobject.RetryPolicy,
) (*service.AgentResponse, error) {
    return a.retryHandler.ExecuteWithRetry(ctx, req, policy, a.Execute)
}
```

### 4.4 上下文管理实现

```go
// PrepareContext 从缓存准备执行上下文
func (a *ClaudeCodeAgent) PrepareContext(
    ctx context.Context, 
    sessionID uuid.UUID, 
    worktreePath string,
) (*service.AgentContext, error) {
    cache, err := a.cacheRepo.Load(ctx, worktreePath)
    if err != nil {
        a.logger.Warn("failed to load cache, using empty context",
            zap.String("sessionId", sessionID.String()),
            zap.Error(err))
        return &service.AgentContext{}, nil
    }
    
    return a.cacheToContext(cache), nil
}

// SaveContext 保存执行上下文到缓存
func (a *ClaudeCodeAgent) SaveContext(
    ctx context.Context, 
    worktreePath string, 
    cache *service.AgentContextCache,
) error {
    infraCache := a.domainToInfraCache(cache)
    return a.cacheRepo.Save(ctx, worktreePath, infraCache)
}
```

### 4.5 生命周期管理实现

```go
// Cancel 取消正在执行的任务
func (a *ClaudeCodeAgent) Cancel(sessionID uuid.UUID) error {
    if !a.IsRunning(sessionID) {
        return errors.New(errors.ErrAgentNotRunning)
    }
    
    return a.processExecutor.Cancel(sessionID)
}

// GetStatus 获取执行状态
func (a *ClaudeCodeAgent) GetStatus(sessionID uuid.UUID) (*service.AgentStatus, error) {
    a.mu.RLock()
    state, exists := a.runningSessions[sessionID]
    a.mu.RUnlock()
    
    if !exists {
        return &service.AgentStatus{
            SessionID: sessionID,
            Status:    "idle",
        }, nil
    }
    
    return &service.AgentStatus{
        SessionID:    state.SessionID,
        Status:       state.Status,
        CurrentStage: state.Stage,
        StartTime:    state.StartTime,
        Progress:     state.Progress,
        Message:      state.Message,
    }, nil
}

// IsRunning 检查是否正在执行
func (a *ClaudeCodeAgent) IsRunning(sessionID uuid.UUID) bool {
    return a.processExecutor.IsRunning(sessionID)
}

// Shutdown 关闭执行器
func (a *ClaudeCodeAgent) Shutdown(ctx context.Context) error {
    a.logger.Info("shutting down claude agent")
    return a.processExecutor.Shutdown(ctx)
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
    req := &service.AgentRequest{
        SessionID:    session.ID,
        Stage:        service.AgentStageClarification,
        WorktreePath: session.Execution.WorktreePath,
        Prompt:       s.buildClarificationPrompt(session.Issue),
        Context: &service.AgentContext{
            IssueTitle: session.Issue.Title,
            IssueBody:  session.Issue.Body,
        },
        Timeout: 10 * time.Minute,
    }
    
    response, err := s.agentRunner.Execute(ctx, req)
    if err != nil {
        return err
    }
    
    // 解析响应，提取澄清问题
    questions := s.extractQuestions(response.Result)
    confirmedPoints := s.extractConfirmedPoints(response.Result)
    
    // 更新 Clarification 实体
    session.Clarification.PendingQuestions = questions
    session.Clarification.ConfirmedPoints = confirmedPoints
    
    if response.Result.StructuredData["readyForDesign"] == true {
        // 触发进入设计阶段
        s.eventDispatcher.Dispatch(ClarificationCompleted{SessionID: session.ID})
    }
    
    return nil
}
```

### 9.2 设计方案阶段

```go
func (s *DesignService) GenerateDesign(ctx context.Context, session *WorkSession) error {
    req := &service.AgentRequest{
        SessionID:    session.ID,
        Stage:        service.AgentStageDesign,
        WorktreePath: session.Execution.WorktreePath,
        Prompt:       s.buildDesignPrompt(session),
        Context: &service.AgentContext{
            IssueTitle:      session.Issue.Title,
            IssueBody:       session.Issue.Body,
            ClarifiedPoints: session.Clarification.ConfirmedPoints,
        },
        Timeout: 15 * time.Minute,
    }
    
    response, err := s.agentRunner.Execute(ctx, req)
    if err != nil {
        return err
    }
    
    // 从结构化数据中提取设计内容
    designContent := response.Result.Content
    complexityScore := response.Result.StructuredData["complexityScore"].(int)
    
    // 创建 Design 实体和版本
    session.Design.CreateVersion(designContent, "初始设计")
    session.Design.ComplexityScore = complexityScore
    
    // 判断是否需要人工确认
    if complexityScore > s.config.ComplexityThreshold || s.config.ForceDesignConfirm {
        session.Design.RequireConfirmation = true
        s.eventDispatcher.Dispatch(DesignCreated{SessionID: session.ID})
    } else {
        session.Design.Confirm()
        s.eventDispatcher.Dispatch(DesignApproved{SessionID: session.ID})
    }
    
    return nil
}
```

### 9.3 任务执行阶段

```go
func (s *TaskService) ExecuteTask(ctx context.Context, session *WorkSession, task *Task) error {
    // 构建任务上下文
    taskContexts := make([]service.TaskContext, len(session.Tasks))
    for i, t := range session.Tasks {
        taskContexts[i] = service.TaskContext{
            ID:           t.ID,
            Description:  t.Description,
            Status:       string(t.Status),
            Dependencies: t.Dependencies,
        }
    }
    
    req := &service.AgentRequest{
        SessionID:    session.ID,
        Stage:        service.AgentStageTaskExecution,
        WorktreePath: session.Execution.WorktreePath,
        Prompt:       s.buildTaskPrompt(task),
        Context: &service.AgentContext{
            DesignContent: session.Design.CurrentVersion.Content,
            Tasks:         taskContexts,
            Branch:        session.Execution.Branch,
        },
        Timeout:      30 * time.Minute,
        AllowedTools: []string{"Read", "Write", "Edit", "Bash", "Glob", "Grep"},
    }
    
    response, err := s.agentRunner.ExecuteWithRetry(ctx, req, s.retryPolicy)
    if err != nil {
        return err
    }
    
    // 处理执行结果
    if response.Success {
        task.MarkCompleted()
        s.eventDispatcher.Dispatch(TaskCompleted{SessionID: session.ID, TaskID: task.ID})
    } else {
        task.MarkFailed(response.Error.Message)
        s.eventDispatcher.Dispatch(TaskFailed{SessionID: session.ID, TaskID: task.ID})
    }
    
    return nil
}
```

### 9.4 恢复中断任务

```go
func (s *RecoveryService) ResumeSession(ctx context.Context, sessionID uuid.UUID) error {
    session, err := s.sessionRepo.FindById(ctx, sessionID)
    if err != nil {
        return err
    }
    
    // 准备上下文
    context, err := s.agentRunner.PrepareContext(ctx, sessionID, session.Execution.WorktreePath)
    if err != nil {
        return err
    }
    
    // 构建恢复请求
    req := &service.AgentRequest{
        SessionID:    sessionID,
        Stage:        service.AgentStageTaskExecution,
        WorktreePath: session.Execution.WorktreePath,
        Prompt:       "继续执行中断的任务",
        Context:      context,
        Timeout:      30 * time.Minute,
    }
    
    result, err := s.agentRunner.Execute(ctx, req)
    if err != nil {
        return err
    }
    
    // 根据结果更新状态
    if result.Success {
        session.CurrentTask.MarkCompleted()
    } else if result.Error != nil {
        session.CurrentTask.MarkFailed(result.Error.Message)
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
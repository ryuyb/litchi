---
name: code-review
description: |
  Use ONLY when user explicitly requests: "/review", "review this code", "检查代码", "code review", "review PR/commit", or asks for feedback on shared code snippets. DO NOT auto-trigger when you see code changes or detect issues proactively. User must ask for review first.
---

# Code Review

执行代码审查，结合项目设计文档输出结构化报告。

## 触发方式

**仅响应用户显式请求**，不自动触发。

触发短语：
- `/review`、`review 这个 PR`、`检查代码`
- `code review`、`review this code`
- 用户粘贴代码并请求反馈

## 输入

| 类型 | 来源 | 示例 |
|------|------|------|
| 文件路径 | 用户指定 | `review src/handler.go` |
| PR 编号 | GitHub | `review PR #123` |
| Git diff | 当前变更 | `review staged changes` |
| 代码片段 | 直接粘贴 | 用户在消息中粘贴代码 |

**无输入时**: 使用 `git diff HEAD` 审查未提交的文件变更。

## 工作流程

```
1. 收集代码 → 2. 加载设计文档 → 3. 识别语言 → 4. 加载规则 → 5. 分析问题 → 6. 输出报告
```

### Step 1: 收集审查范围

根据输入类型获取代码：
- **文件**: 直接读取
- **PR**: 用 `gh pr diff <number>` 获取
- **Git diff**: 用 `git diff` 或 `git diff --staged`
- **片段**: 直接分析用户提供的代码

### Step 2: 加载设计文档

**必须**先读取以下设计文档，作为审查依据：

| 文档 | 路径 | 审查重点 |
|------|------|---------|
| 架构设计 | `docs/design/architecture.md` | 分层架构、依赖方向、技术栈 |
| DDD 设计 | `docs/design/ddd.md` | 聚合边界、实体属性、领域规则 |
| 状态机设计 | `docs/design/state-machine.md` | 阶段转换、状态流转规则 |
| 执行验证设计 | `docs/design/execution-validation.md` | 验证流程、工具配置 |
| Agent Runner 设计 | `docs/design/agent-runner.md` | Agent 接口、错误处理 |

**加载策略**：
- 首次审查时读取全部设计文档
- 后续审查可复用已加载的内容
- 如果文档不存在，跳过并记录警告

### Step 3: 识别语言并加载规则

按文件扩展名选择规则文件：

| 扩展名 | 规则文件 |
|--------|----------|
| `.go` | `references/go-rules.md` |
| `.ts`, `.tsx` | `references/ts-rules.md` |
| 其他 | `references/common-rules.md` |

**多语言时**: 先读本文档的"核心约束"和"设计约束"，再按语言读对应规则文件。

### Step 4: 执行审查

按以下顺序检查：
1. **设计约束** → 架构分层、领域模型、状态机规则（基于设计文档）
2. **正确性** → 错误处理、边界条件、类型安全
3. **安全性** → 注入、XSS、敏感数据
4. **性能** → N+1、内存、渲染
5. **架构** → DDD 分层、依赖方向

### Step 5: 输出报告

**必须**使用以下格式：

```markdown
## Code Review Report

### 概要
- 审查文件数：X
- 发现问题数：Y (🔴 Z, 🟠 A, 🟡 B, 🟢 C)

### 🔴 Critical Issues
#### `file:line` - 问题标题
**问题**: 描述
**影响**: 后果
**建议**: 修复方法 + 代码示例

### 🟠 High Issues
（同上结构）

### 🟡 Medium Issues
（同上结构）

### 🟢 Low Issues
（同上结构）

### 亮点
- 正面评价（可选）
```

### Step 6: 停止条件

审查完成当：
- 所有指定文件已分析
- 设计约束已检查
- 问题已按严重程度分类输出
- 报告已呈现给用户

**不执行修复** - 只输出报告，除非用户明确请求修改代码。

---

## 核心约束

这些规则 **始终适用**，无论语言：

### 严重程度定义

| 级别 | 触发条件 | 示例 |
|------|----------|------|
| 🔴 Critical | 安全漏洞、数据丢失、崩溃风险、设计约束严重违反 | SQL 注入、领域模型错误 |
| 🟠 High | 逻辑错误、明显 bug、性能反模式、设计约束违反 | N+1 查询、分层违规 |
| 🟡 Medium | 可维护性问题、风格缺陷 | 缺少上下文的错误、命名不清 |
| 🟢 Low | 微小改进建议 | 可选的类型注解、注释优化 |

### 项目架构约束

本项目使用：
- **后端**: Go + Fiber v3 + GORM + Uber Fx
- **前端**: React + TanStack Start + TanStack Query + TanStack Store
- **架构**: DDD 分层（Domain → Application → Infrastructure → Presentation）

审查时检查：
- Domain 层不依赖 Infrastructure（无 `*gorm.DB` 等字段）
- 服务端数据获取使用 TanStack Query（不用手动 `fetch` + `useState`）
- 客户端状态使用 TanStack Store

---

## 设计约束检查

基于 `docs/design/` 目录下的设计文档，检查以下约束：

### 架构分层约束（来自 architecture.md）

| 规则 | 说明 | 严重程度 |
|------|------|---------|
| 依赖方向 | Domain → Application → Infrastructure → Presentation，内层不依赖外层 | 🟠 High |
| 聚合边界 | WorkSession 是聚合根，Issue/Clarification/Design/Task/Execution 是内部实体 | 🟠 High |
| 仓库接口 | 仓库接口定义在 Domain 层，实现在 Infrastructure 层 | 🟡 Medium |
| 领域服务 | 复杂度评估、阶段转换、任务调度等是领域服务 | 🟡 Medium |

**检查方法**：
- Domain 层文件不应 import Infrastructure 包
- 实体不应直接依赖外部服务

### 领域模型约束（来自 ddd.md）

| 实体/值对象 | 约束 | 严重程度 |
|------------|------|---------|
| Stage | 枚举值：Clarification/Design/TaskBreakdown/Execution/PullRequest/Completed | 🟠 High |
| TaskStatus | 枚举值：Pending/InProgress/Completed/Failed/Skipped | 🟠 High |
| WorkSession | 一个 WorkSession 对应一个 Issue | 🔴 Critical |
| 阶段转换 | 只能按顺序前进或逐级回退 | 🟠 High |
| Task | 依赖任务完成后才能执行 | 🟡 Medium |

**检查方法**：
- 状态值必须是定义的枚举之一
- 不允许跳过阶段

### 状态机约束（来自 state-machine.md）

| 规则 | 说明 | 严重程度 |
|------|------|---------|
| 正向转换 | Clarification → Design → TaskBreakdown → Execution → PullRequest → Completed | 🔴 Critical |
| 回退规则 | Execution → Design, Design → Clarification, PR 支持 R4/R5/R6 三级回退 | 🟠 High |
| Task 状态 | Pending → InProgress → Completed/Failed/Skipped，Failed 可重试 | 🟠 High |
| 暂停恢复 | 可从任何阶段暂停，恢复后回到原阶段 | 🟡 Medium |

**检查方法**：
- 状态转换代码必须遵循定义的规则
- 不允许非法的状态跳转

### 执行验证约束（来自 execution-validation.md）

| 规则 | 说明 | 严重程度 |
|------|------|---------|
| 验证顺序 | 格式化 → Lint → 测试 | 🟡 Medium |
| 失败策略 | FailFast/AutoFix/WarnContinue/Skip | 🟡 Medium |
| 自动检测 | 支持 Go/NodeJS/Python/Rust 项目检测 | 🟢 Low |

### Agent Runner 约束（来自 agent-runner.md）

| 规则 | 说明 | 严重程度 |
|------|------|---------|
| 接口实现 | Execute/AnalyzeCodebase/GenerateDesign/Clarify/Resume 方法 | 🟠 High |
| 错误处理 | 错误类型、重试策略、严重程度分类 | 🟡 Medium |
| 工具权限 | 按阶段控制工具使用，危险操作需审批 | 🟠 High |

**检查方法**：
- 实现 AgentRunner 接口的结构体必须实现所有方法
- 错误处理必须使用定义的错误类型

---

## 规则文件导航

审查具体语言时，读取对应规则文件：

- **Go 代码**: → `references/go-rules.md`
- **TypeScript/React**: → `references/ts-rules.md`
- **通用规则**: → `references/common-rules.md`

规则文件包含：具体检查项、代码示例、修复建议模板。
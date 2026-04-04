# 状态机详细设计文档

## 1. 概述

本文档详细定义 Litchi 系统中的核心状态机，包括：
- **WorkSession 阶段状态机**：管理整体工作流程的阶段流转
- **Task 状态机**：管理单个任务的执行状态
- **Clarification 状态机**：管理需求澄清过程
- **Design 状态机**：管理设计方案确认过程

---

## 2. WorkSession 阶段状态机

### 2.1 状态定义

| 状态 | 编码 | 说明 | 可执行操作 |
|------|------|------|-----------|
| Clarification | `clarification` | 需求澄清阶段 | 提问、回答、确认需求 |
| Design | `design` | 设计方案阶段 | 创建设计、修改设计、确认设计 |
| TaskBreakdown | `task_breakdown` | 任务拆解阶段 | 拆解任务、确认任务列表 |
| Execution | `execution` | 任务执行阶段 | 执行任务、测试验证 |
| PullRequest | `pull_request` | PR 创建阶段 | 创建 PR、等待合并 |
| Completed | `completed` | 已完成 | 无（终态） |
| Paused | `paused` | 已暂停 | 恢复、终止 |
| Terminated | `terminated` | 已终止 | 无（终态） |
| **ErrorRecovery** | `error_recovery` | 错误恢复中 | 自动恢复尝试 |

### 2.2 状态转换图

```
                                ┌─────────────────────────────────────┐
                                │                                     │
                                ▼                                     │
┌──────────────┐         ┌──────────────┐         ┌──────────────┐   │
│              │         │              │         │              │   │
│ Clarification│────────▶│    Design    │────────▶│TaskBreakdown │   │
│              │  (1)    │              │  (2)    │              │   │
│              │         │              │         │              │   │
└──────────────┘         └──────────────┘         └──────────────┘   │
       ▲                        │                        │          │
       │                        │                        │          │
       │                   ┌────┴────┐                   │          │
       │                   │         │                   │          │
       │                   │ rollback│                   │          │
       │                   │  (R2)   │                   │          │
       │                   │         │                   │          │
       │                   └────┬────┘                   │          │
       │                        │                        │          │
       │                        │              ┌─────────┴──────┐   │
       │                        │              │                │   │
       │                        │              │                │   │
       │                        ▼              ▼                │   │
       │                 ┌──────────────────────────┐          │   │
       │                 │                          │          │   │
       │                 │       Execution          │◀─────────┘   │
       │                 │                          │  (3)         │
       │                 │                          │              │
       │                 └──────────────────────────┘              │
       │                          ▲                                │
       │                          │                                │
       │                     ┌────┴────┐                           │
       │                     │         │                           │
       │                     │ rollback│                           │
       │                     │  (R1)   │                           │
       │                     │         │                           │
       │                     └────┬────┘                           │
       │                          │                                │
       │                          │              ┌─────────────────┘
       │                          ▼              │
       │                   ┌──────────────┐      │
       │                   │              │      │
       │                   │ PullRequest  │──────┘
       │                   │              │  (4)
       │                   │              │
       │                   └──────────────┘
       │                     │    │    │
       │                     │    │    │
       │          ┌──────────┘    │    └──────────┐
       │          │               │               │
       │    rollback (R6)   rollback (R5)   rollback (R4)
       │          │               │               │
       │          ▼               ▼               ▼
       │    ┌──────────┐    ┌──────────┐    ┌──────────┐
       │    │(到澄清)  │    │(到设计)  │    │(到执行)  │
       │    └──────────┘    └──────────┘    └──────────┘
       │
       │                          │
       │                          │ PR 合并 (5)
       │                          │
       │                          ▼
       │                   ┌──────────────┐
       │                   │              │
       │                   │  Completed   │
       │                   │              │
       │                   │   (终态)     │
       │                   └──────────────┘
       │
       │         ┌────────────────────────────────────────────┐
       │         │                                            │
       └────────▶│                    Paused                  │
       │         │                                            │
       │         │  可从任何阶段暂停，恢复后回到原阶段        │
       │         │                                            │
       │         └────────────────────────────────────────────┘
       │                          │
       │                          │ terminate
       │                          │
       │                          ▼
       │                   ┌──────────────┐
       │                   │              │
       │                   │ Terminated   │
       │                   │              │
       │                   │   (终态)     │
       │                   └──────────────┘
       │
       └─────────────────────── rollback (R3)
                               到 Clarification
```

### 2.3 正向转换规则

#### 转换 (1): Clarification → Design

| 条件 | 说明 |
|------|------|
| 前置条件 | `clarification.status == completed` |
| 前置条件 | 至少有一个已确认的需求点 |
| 前置条件 | 无待回答问题 |
| 前置条件 | 清晰度评分 ≥ `clarityThreshold`（默认 60） |
| 触发方式 | 见下方触发规则表 |
| 产生事件 | `ClarificationCompleted`、`StageTransitioned` |
| 后置行为 | 初始化 Design 实体 |

**触发规则（按清晰度评分）**：

| 评分范围 | 触发方式 |
|----------|----------|
| ≥ 80 | 自动转换，无需确认 |
| 60-79 | 自动转换，但 Design 需人工确认 |
| 40-59 | 需用户确认后转换 |
| < 40 | 不能转换，必须继续澄清 |
| 用户指令"开始设计" | 跳过评分检查，直接转换 |

#### 转换 (2): Design → TaskBreakdown

| 条件 | 说明 |
|------|------|
| 前置条件 | `design.confirmed == true` |
| 前置条件 | 存在至少一个设计版本 |
| 触发方式 | 用户确认设计（复杂度超阈值需人工确认），或自动确认 |
| 产生事件 | `DesignApproved`、`StageTransitioned` |
| 后置行为 | 准备任务拆解 |

#### 转换 (3): TaskBreakdown → Execution

| 条件 | 说明 |
|------|------|
| 前置条件 | 任务列表已生成 |
| 前置条件 | 任务依赖关系已确定 |
| 触发方式 | Agent 完成任务拆解后自动进入 |
| 产生事件 | `TaskListCreated`、`StageTransitioned` |
| 后置行为 | 创建 Git Worktree、创建分支、初始化 Execution 实体 |

#### 转换 (4): Execution → PullRequest

| 条件 | 说明 |
|------|------|
| 前置条件 | 所有任务状态为 `completed` 或 `skipped` |
| 前置条件 | 无失败任务（`failed` 状态需处理） |
| 触发方式 | 最后一个任务完成后自动进入 |
| 产生事件 | `StageTransitioned` |
| 后置行为 | 准备创建 PR |

#### 转换 (5): PullRequest → Completed

| 条件 | 说明 |
|------|------|
| 前置条件 | PR 已创建成功 |
| 前置条件 | PR 已合并（人工或自动合并） |
| 前置条件 | 无未处理的回退请求 |
| 触发方式 | PR 合并后自动进入 |
| 产生事件 | `PullRequestMerged`、`StageTransitioned`、`WorkSessionCompleted` |
| 后置行为 | 清理 Worktree（可选）、归档状态 |

**说明**：PR 创建后进入 `pull_request` 阶段等待合并，只有在 PR 合并后才能转换到 `completed` 终态。若 PR 阶段需要修改，可通过回退规则 (R4)、(R5)、(R6) 回退到执行、设计或澄清阶段。

### 2.4 回退转换规则

#### 回退 (R1): Execution → Design

| 条件 | 说明 |
|------|------|
| 前置条件 | 当前阶段为 `execution` |
| 前置条件 | 存在失败任务，或用户主动请求回退 |
| 触发方式 | 用户指令 `@bot 回退设计` 或 `@bot 修改设计` |
| 产生事件 | `StageRolledBack` |
| 后置行为 | 设计版本号 +1（创建新版本） |
| 后置行为 | 当前分支标记为 `deprecated` |
| 后置行为 | 保留已完成的任务记录 |
| 后置行为 | 清空失败任务状态 |

**示例**：
```
当前设计版本: v2
执行阶段任务失败，用户请求回退
→ 回退到 Design 阶段
→ 创建设计版本 v3
→ 分支 issue-123-feature 标记为废弃
→ 创建新分支 issue-123-feature-v3
```

#### 回退 (R2): Design → Clarification

| 条件 | 说明 |
|------|------|
| 前置条件 | 当前阶段为 `design` 或 `task_breakdown` |
| 触发方式 | 用户指令 `@bot 回退澄清` |
| 产生事件 | `StageRolledBack` |
| 后置行为 | 保留已确认的需求点 |
| 后置行为 | 清空设计版本 |
| 后置行为 | 可添加新的待澄清问题 |

#### 回退 (R3): Execution → Clarification

| 条件 | 说明 |
|------|------|
| 前置条件 | 当前阶段为 `execution` |
| 触发方式 | 用户指令 `@bot 回退澄清` |
| 产生事件 | `StageRolledBack` |
| 后置行为 | 保留已确认的需求点 |
| 后置行为 | 清空设计版本 |
| 后置行为 | 当前分支标记为 `deprecated` |

#### 回退 (R4): PullRequest → Execution（浅层回退）

| 条件 | 说明 |
|------|------|
| 前置条件 | 当前阶段为 `pull_request` |
| 前置条件 | PR 状态为 `open`（未合并） |
| 触发方式 | 用户指令 `@bot 修改代码` 或 CI 失败通知（配置允许） |
| 产生事件 | `StageRolledBack` |
| 后置行为 | 在当前分支追加修复任务 |
| 后置行为 | 保留现有任务完成记录 |
| 后置行为 | PR 状态保持 `open` |

**适用场景**：
- PR Review 发现小问题需要修复
- CI 检查失败需要修复代码
- 用户主动要求调整代码

**示例**：
```
当前阶段: pull_request
PR 状态: open
CI 检查: 测试失败

用户请求: @bot 修改代码
→ 回退到 Execution 阶段
→ 追加修复任务
→ 完成后重新提交 PR
→ PR 保持 open 状态
```

#### 回退 (R5): PullRequest → Design（深层回退）

| 条件 | 说明 |
|------|------|
| 前置条件 | 当前阶段为 `pull_request` |
| 前置条件 | PR 状态为 `open`（未合并） |
| 触发方式 | 用户指令 `@bot 回退设计` |
| 产生事件 | `StageRolledBack` |
| 后置行为 | 设计版本号 +1（创建新版本） |
| 后置行为 | 当前分支标记为 `deprecated` |
| 后置行为 | 关闭当前 PR |

**适用场景**：
- PR Review 发现设计问题需要重新设计
- 需求变更导致设计方案需要调整
- 用户主动要求重新设计

**示例**：
```
当前阶段: pull_request
当前设计版本: v2
PR 状态: open

用户请求: @bot 回退设计
→ 回退到 Design 阶段
→ 创建设计版本 v3
→ 分支 issue-123-feature 标记为废弃
→ 关闭 PR #456
→ 设计完成后创建新分支
```

#### 回退 (R6): PullRequest → Clarification（最深层回退）

| 条件 | 说明 |
|------|------|
| 前置条件 | 当前阶段为 `pull_request` |
| 前置条件 | PR 状态为 `open`（未合并） |
| 触发方式 | 用户指令 `@bot 回退澄清` |
| 产生事件 | `StageRolledBack` |
| 后置行为 | 保留已确认的需求点 |
| 后置行为 | 清空设计版本 |
| 后置行为 | 当前分支标记为 `deprecated` |
| 后置行为 | 关闭当前 PR |

**适用场景**：
- 需求理解偏差严重需要重新澄清
- 用户需求发生根本性变化
- 用户主动要求重新澄清需求

**示例**：
```
当前阶段: pull_request
PR 状态: open

用户请求: @bot 回退澄清
→ 回退到 Clarification 阶段
→ 保留已确认的需求点
→ 清空设计版本
→ 分支 issue-123-feature 标记为废弃
→ 关闭 PR #456
→ 重新澄清后进入设计阶段
```

#### PR 阶段回退对比

| 回退规则 | 目标阶段 | 回退深度 | 分支处理 | PR 处理 | 典型场景 |
|---------|---------|---------|---------|---------|---------|
| R4 | Execution | 浅层 | 保留分支 | 保持 open | 代码修复、CI 失败 |
| R5 | Design | 深层 | 废弃分支 | 关闭 PR | 设计调整、需求变更 |
| R6 | Clarification | 最深层 | 废弃分支 | 关闭 PR | 需求重新澄清 |

### 2.5 暂停与恢复

#### 暂停转换: Any → Paused

| 条件 | 说明 |
|------|------|
| 前置条件 | 当前阶段不为 `completed`、`terminated`、`paused` |
| 触发方式 | 用户指令 `@bot 暂停`、Task 失败等待指令、外部错误 |
| 产生事件 | `WorkSessionPaused` |
| 后置行为 | 保存当前状态到 `context.json` |
| 后置行为 | 记录暂停原因 |

**暂停原因枚举**：

| 原因 | 编码 | 说明 | 恢复条件 |
|------|------|------|---------|
| UserRequest | `user_request` | 仓库管理员主动暂停 | 管理员指令继续 |
| TaskFailed | `task_failed` | 任务执行失败，等待指令 | 管理员指令继续/跳过/回退 |
| ApprovalPending | `approval_pending` | 等待危险操作审批 | 管理员回复同意/拒绝 |
| ExternalError | `external_error` | API 限流、网络错误等 | 自动恢复或管理员干预 |
| ServiceRestart | `service_restart` | 服务重启时的中断状态 | 服务启动自动恢复 |
| PRReviewPending | `pr_review_pending` | PR 等待 Review 反馈 | 管理员指令继续/回退 |
| CIFailure | `ci_failure` | CI 检查失败，等待处理 | 管理员指令修复或回退 |
| **AgentCrashed** | `agent_crashed` | Agent 进程崩溃 | 管理员指令继续，尝试恢复会话 |
| **RateLimited** | `rate_limited` | GitHub API 限流 | 自动等待恢复 |
| **TestEnvUnavailable** | `test_env_unavailable` | 测试环境不可用 | 环境恢复或管理员强制执行 |
| **Timeout** | `timeout` | 操作超时 | 管理员指令继续或取消 |
| **ResourceExhausted** | `resource_exhausted` | 并发资源不足（队列已满） | 自动排队或管理员取消 |
| **BudgetExceeded** | `budget_exceeded` | 预算超限 | 管理员增加预算或使用备用模型 |
| **SessionLost** | `session_lost` | 会话上下文丢失 | 需重新触发 |

> **注意**：
> - 加粗的暂停原因为新增项
> - 暂停恢复等关键指令仅管理员可执行
> - 澄清阶段 Issue 作者也可参与，但暂停恢复仍需管理员执行

#### 暂停原因分类

| 分类 | 暂停原因 | 自动恢复 | 需人工干预 |
|------|---------|---------|-----------|
| **自动恢复** | `rate_limited` | ✅ 等待 API 重置 | ❌ |
| **自动恢复** | `resource_exhausted` | ✅ 排队等待 | ❌ |
| **半自动恢复** | `agent_crashed` | ✅ 尝试恢复会话 | ✅ 需确认继续 |
| **半自动恢复** | `test_env_unavailable` | ✅ 定时重检 | ✅ 可强制执行 |
| **半自动恢复** | `budget_exceeded` | ✅ 切换备用模型 | ✅ 需确认 |
| **需人工干预** | `task_failed` | ❌ | ✅ 需指令 |
| **需人工干预** | `approval_pending` | ❌ | ✅ 需审批 |
| **需人工干预** | `session_lost` | ❌ | ✅ 需重新触发 |
| **需人工干预** | `timeout` | ❌ | ✅ 需指令 |

**PR 阶段暂停特殊处理**：

当 WorkSession 处于 `pull_request` 阶段时，以下情况会触发暂停：

| 触发条件 | 暂停原因 | 恢复条件 |
|---------|---------|---------|
| PR Review 提出修改意见 | `pr_review_pending` | 用户指令继续或回退 |
| CI 检查失败 | `ci_failure` | 用户指令修复或回退 |
| 合并冲突 | `external_error` | 解决冲突后继续 |

#### 恢复转换: Paused → OriginalStage

| 条件 | 说明 |
|------|------|
| 前置条件 | 当前状态为 `paused` |
| 前置条件 | 暂停原因已解决（如审批通过、用户指令继续） |
| 触发方式 | 用户指令 `@bot 继续`、服务启动自动恢复 |
| 产生事件 | `WorkSessionResumed` |
| 后置行为 | 从断点继续执行 |

#### 终止转换: Paused → Terminated

| 条件 | 说明 |
|------|------|
| 前置条件 | 当前状态为 `paused` 或任意非终态阶段 |
| 触发方式 | 用户指令 `@bot 终止` |
| 产生事件 | `WorkSessionTerminated` |
| 后置行为 | 清理资源（Worktree、分支） |
| 后置行为 | 归档状态 |

---

## 3. Task 状态机

### 3.1 状态定义

| 状态 | 编码 | 说明 | 后续状态 |
|------|------|------|---------|
| Pending | `pending` | 待执行 | `in_progress` |
| InProgress | `in_progress` | 执行中 | `completed`、`failed`、`skipped` |
| Completed | `completed` | 已完成（终态） | 无 |
| Failed | `failed` | 已失败 | `in_progress`（重试）、`skipped` |
| Skipped | `skipped` | 已跳过（终态） | 无 |

### 3.2 状态转换图

```
┌─────────────┐
│             │
│   Pending   │
│             │
└─────────────┘
       │
       │ start (依赖任务已完成)
       │
       ▼
┌─────────────┐
│             │
│  InProgress │
│             │
└─────────────┘
       │
       ├──────────────────────┬──────────────────────┐
       │                      │                      │
       │ success               │ fail                 │ skip (用户指令)
       │                      │                      │
       ▼                      ▼                      ▼
┌─────────────┐        ┌─────────────┐        ┌─────────────┐
│             │        │             │        │             │
│  Completed  │        │   Failed    │        │   Skipped   │
│   (终态)    │        │             │        │   (终态)    │
│             │        └─────────────┘        │             │
└─────────────┘              │                └─────────────┘
                              │
                              │ retry (重试次数 < limit)
                              │
                              ▼
                       ┌─────────────┐
                       │             │
                       │  InProgress │
                       │             │
                       └─────────────┘
                              │
                              │ retry exhausted
                              │ 或用户指令
                              │
                              ▼
                       ┌─────────────┐
                       │             │
                       │   Skipped   │
                       │   (终态)    │
                       │             │
                       └─────────────┘
```

### 3.3 转换规则

#### 转换: Pending → InProgress

| 条件 | 说明 |
|------|------|
| 前置条件 | 所有依赖任务状态为 `completed` |
| 前置条件 | 当前无其他任务在执行（串行执行） |
| 触发方式 | TaskScheduler 自动调度 |
| 产生事件 | `TaskStarted` |
| 后置行为 | 设置 `execution.currentTask` |

#### 转换: InProgress → Completed

| 条件 | 说明 |
|------|------|
| 前置条件 | Agent 执行成功 |
| 前置条件 | 测试通过（如需测试） |
| 触发方式 | AgentRunner 报告成功 + 测试结果 |
| 产生事件 | `TaskCompleted` |
| 后置行为 | 记录执行结果、加入 `execution.completedTasks` |

#### 转换: InProgress → Failed

| 条件 | 说明 |
|------|------|
| 前置条件 | Agent 执行失败，或测试失败 |
| 前置条件 | 自动修复尝试失败 |
| 触发方式 | AgentRunner 报告失败 + 重试次数已达上限 |
| 产生事件 | `TaskFailed` |
| 后置行为 | 记录失败原因和建议、设置 `execution.failedTask` |
| 后置行为 | WorkSession 进入 `paused` 状态 |

#### 转换: Failed → InProgress (重试)

| 条件 | 说明 |
|------|------|
| 前置条件 | `retryCount < taskRetryLimit` |
| 前置条件 | 用户未请求跳过 |
| 触发方式 | 自动重试，或用户指令 `@bot 重试` |
| 产生事件 | `TaskRetryStarted` |
| 后置行为 | `retryCount++` |

#### 转换: Failed/Skipped → Skipped

| 条件 | 说明 |
|------|------|
| 前置条件 | 用户指令跳过，或重试次数耗尽 |
| 触发方式 | 用户指令 `@bot 跳过` |
| 产生事件 | `TaskSkipped` |
| 后置行为 | 清除 `execution.failedTask` |

---

## 4. Clarification 状态机

### 4.1 状态定义

| 状态 | 编码 | 说明 |
|------|------|------|
| InProgress | `in_progress` | 澄清进行中 |
| Completed | `completed` | 澄清完成 |

### 4.2 状态转换图

```
┌─────────────┐
│             │
│ InProgress  │
│             │
└─────────────┘
       │
       │ 所有问题已回答 + 至少一个需求确认
       │
       ▼
┌─────────────┐
│             │
│  Completed  │
│   (终态)    │
│             │
└─────────────┘
```

### 4.3 完成条件

| 条件 | 说明 |
|------|------|
| `pendingQuestions.length == 0` | 所有问题已回答 |
| `confirmedPoints.length > 0` | 至少一个需求已确认 |
| `clarityScore >= clarityThreshold` | 清晰度评分达标（默认 ≥60） |
| 用户指令"开始设计" | 可跳过评分检查 |

**清晰度评分等级判定**：

| 评分 | 处理 |
|------|------|
| ≥ 80 | 自动完成，进入 Design |
| 60-79 | 自动完成，但 Design 需确认 |
| 40-59 | 需用户确认才能完成 |
| < 40 | 不能完成，继续澄清 |

---

## 5. Design 状态机

### 5.1 状态定义

| 状态 | 编码 | 说明 |
|------|------|------|
| Drafting | `drafting` | 设计草稿中 |
| PendingApproval | `pending_approval` | 等待审批 |
| Approved | `approved` | 已批准 |
| Rejected | `rejected` | 已拒绝 |

### 5.2 状态转换图

```
┌─────────────┐
│             │
│  Drafting   │
│             │
└─────────────┘
       │
       │ create version
       │
       ├──────────────────────┐
       │                      │
       │ require_confirmation │ !require_confirmation
       │                      │
       ▼                      ▼
┌─────────────┐        ┌─────────────┐
│             │        │             │
│PendingApproval│      │  Approved   │
│             │        │   (终态)    │
└─────────────┘        │             │
       │                └─────────────┘
       │                      ▲
       ├──────────────────────┤
       │                      │
       │ approve               │ auto_approve
       │                      │
       ▼                      │
┌─────────────┐              │
│             │              │
│  Approved   │──────────────┘
│   (终态)    │
│             │
└─────────────┘

       │ reject (用户反馈)
       │
       ▼
┌─────────────┐
│             │
│  Rejected   │
│             │
└─────────────┘
       │
       │ create new version
       │
       ▼
┌─────────────┐
│             │
│  Drafting   │
│             │
└─────────────┘
```

### 5.3 转换规则

#### 转换: Drafting → PendingApproval

| 条件 | 说明 |
|------|------|
| 前置条件 | 设计版本已创建 |
| 前置条件 | `complexityScore > threshold` 或 `forceDesignConfirm == true` |
| 触发方式 | Agent 完成设计后自动判断 |
| 产生事件 | `DesignCreated` |

#### 转换: Drafting → Approved

| 条件 | 说明 |
|------|------|
| 前置条件 | 设计版本已创建 |
| 前置条件 | `complexityScore <= threshold` 且 `forceDesignConfirm == false` |
| 触发方式 | 自动批准 |
| 产生事件 | `DesignCreated`、`DesignApproved` |

#### 转换: PendingApproval → Approved

| 条件 | 说明 |
|------|------|
| 前置条件 | 用户回复"同意"或"确认" |
| 触发方式 | 用户指令 |
| 产生事件 | `DesignApproved` |

#### 转换: PendingApproval → Rejected

| 条件 | 说明 |
|------|------|
| 前置条件 | 用户反馈修改意见 |
| 触发方式 | 用户指令 |
| 产生事件 | `DesignRejected` |
| 后置行为 | 准备创建新版本 |

#### 转换: Rejected → Drafting

| 条件 | 说明 |
|------|------|
| 前置条件 | 用户提供了修改意见 |
| 触发方式 | Agent 根据反馈创建新版本 |
| 后置行为 | 版本号 +1，记录变更原因 |

---

## 6. 状态持久化

### 6.1 持久化策略

**双存储机制**：
- **数据库（主存储）**：PostgreSQL 存储 WorkSession 聚合的完整状态，用于系统持久化、查询统计、崩溃恢复
- **文件缓存（Agent 输入输出）**：`.litchi/issues/{issue-id}/` 目录存储 Agent 执行时的上下文缓存

**一致性保证**：状态变更时先更新数据库，再同步更新文件缓存。

### 6.2 状态存储结构

#### 数据库存储（见架构设计文档）

PostgreSQL 表结构详见 `architecture-design.md` 第 7 节数据库设计。

#### 文件缓存结构

位置：`.litchi/issues/{issue-id}/context.json`

```json
{
  "sessionId": "uuid",
  "currentStage": "execution",
  "status": "active",
  "pauseReason": null,

  "clarification": {
    "status": "completed",
    "confirmedPoints": ["需求点1", "需求点2"],
    "pendingQuestions": []
  },

  "design": {
    "status": "approved",
    "currentVersion": 2,
    "complexityScore": 65,
    "requireConfirmation": false,
    "confirmed": true
  },

  "execution": {
    "currentTaskId": "uuid-3",
    "completedTaskIds": ["uuid-1", "uuid-2"],
    "failedTaskId": null,
    "branch": "issue-123-feature",
    "branchDeprecated": false,
    "worktreePath": "/repos/worktree/issue-123"
  },

  "tasks": [
    {
      "id": "uuid-1",
      "status": "completed",
      "retryCount": 0
    },
    {
      "id": "uuid-2",
      "status": "completed",
      "retryCount": 0
    },
    {
      "id": "uuid-3",
      "status": "in_progress",
      "retryCount": 0
    },
    {
      "id": "uuid-4",
      "status": "pending",
      "retryCount": 0
    }
  ],

  "createdAt": "2026-04-03T10:00:00Z",
  "updatedAt": "2026-04-03T12:30:00Z"
}
```

### 6.2 状态更新时机

| 时机 | 更新内容 |
|------|---------|
| 阶段转换 | `currentStage`、`updatedAt` |
| 任务开始 | `execution.currentTaskId`、对应任务 `status` |
| 任务完成 | `execution.completedTaskIds`、对应任务 `status`、`updatedAt` |
| 任务失败 | `execution.failedTaskId`、对应任务 `status`、`failureReason` |
| 暂停 | `status`、`pauseReason`、`updatedAt` |
| 恢复 | 从数据库读取状态，验证 `context.json` 一致性 |
| 后置行为 | 从断点继续执行 |
| 回退 | `currentStage`、相关子状态重置、`updatedAt` |

### 6.3 暂停恢复超时配置

| 暂停原因 | 自动恢复超时 | 最大等待时间 | 超时后处理 |
|---------|-------------|-------------|-----------|
| `rate_limited` | API 重置时间 | 30分钟 | 通知管理员 |
| `resource_exhausted` | 队列预估时间 | 1小时 | 通知管理员 |
| `test_env_unavailable` | 5分钟检测周期 | 30分钟 | 通知管理员 |
| `agent_crashed` | 不自动恢复 | - | 等待管理员指令 |
| `task_failed` | 不自动恢复 | - | 等待管理员指令 |
| `timeout` | 不自动恢复 | - | 等待管理员指令 |

---

## 7. 状态转换事件

### 7.1 事件与状态转换映射

| 事件 | 状态转换 |
|------|---------|
| `WorkSessionStarted` | 初始化 → `clarification` |
| `ClarificationCompleted` | `clarification` → `design` |
| `DesignApproved` | `design` → `task_breakdown` |
| `TaskListCreated` | `task_breakdown` → `execution` |
| `TaskStarted` | Task: `pending` → `in_progress` |
| `TaskCompleted` | Task: `in_progress` → `completed` |
| `TaskFailed` | Task: `in_progress` → `failed` |
| `TaskSkipped` | Task: `failed` → `skipped` |
| `AllTasksCompleted` | `execution` → `pull_request` |
| `PullRequestCreated` | `execution` → `pull_request` |
| `PullRequestMerged` | `pull_request` → `completed` |
| `StageRolledBack` | 回退到指定阶段 |
| `PRRolledBackToExecution` | `pull_request` → `execution`（R4 浅层回退） |
| `PRRolledBackToDesign` | `pull_request` → `design`（R5 深层回退） |
| `PRRolledBackToClarification` | `pull_request` → `clarification`（R6 最深层回退） |
| `WorkSessionPaused` | 任意 → `paused` |
| `WorkSessionResumed` | `paused` → 原阶段 |
| `WorkSessionTerminated` | 任意 → `terminated` |

### 7.2 事件发布时机

```
领域对象执行业务方法
       │
       │ 内部状态变更
       │
       ▼
添加领域事件到 events 数组
       │
       │ 方法返回后
       │
       ▼
Application Service 提取事件
       │
       ▼
Event Dispatcher 分发事件
       │
       ├──────────────────────┬──────────────────────┐
       │                      │                      │
       ▼                      ▼                      ▼
持久化事件          发布 WebSocket            执行后续操作
(可选)              推送给前端                (如 GitHub Comment)
```

---

## 8. 状态查询与展示

### 8.1 前端展示状态

前端展示时需将内部状态转换为用户友好的文本：

| 内部状态 | 前端展示 |
|---------|---------|
| `clarification` | 需求澄清中 |
| `design` | 设计方案中 |
| `task_breakdown` | 任务拆解中 |
| `execution` | 任务执行中 |
| `pull_request` | 创建 PR |
| `completed` | 已完成 |
| `paused` | 已暂停 |
| `terminated` | 已终止 |

### 8.2 进度计算

```go
// 任务完成进度
func CalculateProgress(session *WorkSession) float64 {
    if len(session.tasks) == 0 {
        return 0
    }
    
    completed := 0
    for _, task := range session.tasks {
        if task.status == TaskStatusCompleted || task.status == TaskStatusSkipped {
            completed++
        }
    }
    
    return float64(completed) / float64(len(session.tasks)) * 100
}
```

---

## 9. 异常状态处理

### 9.1 状态不一致检测

| 场景 | 检测方法 | 处理策略 |
|------|---------|---------|
| 任务状态与执行状态不一致 | `execution.currentTaskId` 对应任务状态非 `in_progress` | 自动修复状态 |
| 阶段状态与子实体状态不一致 | 如 `stage=execution` 但 `tasks` 为空 | 回退到 `task_breakdown` |
| 暂停后恢复失败 | 读取 `context.json` 失败 | 重新初始化，保留 Issue 信息 |
| **Agent 进程异常终止** | 进程状态检测 | 尝试恢复会话，失败则暂停 |
| **GitHub API 持续限流** | 连续限流次数超过阈值 | 暂停并通知管理员 |
| **测试环境持续不可用** | 环境检测失败超过阈值 | 暂停并通知管理员 |
| **会话上下文文件损坏** | JSON 解析失败 | 从数据库恢复，重建缓存 |
| **预算持续超限** | 多次切换备用模型仍超限 | 暂停并通知管理员 |

### 9.2 状态修复策略

```go
func (s *WorkSession) RepairState() error {
    // 检查执行阶段任务状态一致性
    if s.currentStage == StageExecution {
        if s.execution.currentTaskId != nil {
            task := s.findTask(*s.execution.currentTaskId)
            if task.status != TaskStatusInProgress {
                // 修复：调用领域方法修改状态
                task.Start()
            }
        }
    }
    
    // 检查阶段前置条件
    if s.currentStage == StageExecution && len(s.tasks) == 0 {
        // 修复：回退到任务拆解
        s.RollbackTo(StageTaskBreakdown, "任务列表丢失")
    }
    
    return nil
}
```

**修复时机**：服务启动时自动检测并修复所有进行中的 WorkSession。
```

---

## 10. 并发状态控制

### 10.1 并发场景

| 场景 | 风险 | 控制策略 |
|------|------|---------|
| 多个 Issue 同时处理 | WorkSession 状态独立，无冲突 | 每个 Issue 独立 Worktree |
| 用户同时发送多个指令 | 状态竞争 | 指令队列串行处理 |
| WebSocket 推送与状态更新 | 状态不一致 | 事件驱动，状态更新后推送 |
| 服务重启恢复 | 状态丢失 | 从持久化恢复 + 状态修复 |
| PR 回退与合并同时发生 | 状态竞争 | 检查 PR 状态后再执行回退 |
| PR 回退中用户发送新指令 | 操作冲突 | 回退过程中锁定 WorkSession |
| CI 回调与用户指令冲突 | 状态不一致 | 乐观锁 + 事件版本控制 |
| 多次 PR 回退 | 分支混乱 | 记录回退历史，限制回退次数 |

### 10.2 PR 回退并发控制

**回退锁定机制**：

当执行 PR 回退操作时，WorkSession 进入锁定状态，防止并发操作：

```go
type RollbackLock struct {
    IsLocked      bool      `json:"isLocked"`
    LockReason    string    `json:"lockReason"`
    LockedAt      time.Time `json:"lockedAt"`
    TargetStage   Stage     `json:"targetStage"`
}
```

**锁定规则**：
1. 回退开始前检查 `RollbackLock.IsLocked`
2. 若已锁定，拒绝新的回退请求，返回当前回退进度
3. 回退完成后释放锁

**PR 状态检查**：

执行 PR 回退前需检查 GitHub PR 状态：

| PR 状态 | 可执行操作 |
|---------|-----------|
| `open` | 可执行 R4/R5/R6 回退 |
| `merged` | 不可回退，返回错误提示 |
| `closed` | 不可回退，返回错误提示 |

**回退次数限制**：

为防止无限回退，限制 PR 阶段的最大回退次数：

```go
type PRRollbackHistory struct {
    TotalCount   int                `json:"totalCount"`
    Records      []RollbackRecord   `json:"records"`
    MaxAllowed   int                `json:"maxAllowed"` // 默认 3
}

type RollbackRecord struct {
    FromStage    Stage     `json:"fromStage"`
    ToStage      Stage     `json:"toStage"`
    Reason       string    `json:"reason"`
    Timestamp    time.Time `json:"timestamp"`
    BranchBefore string    `json:"branchBefore"`
    BranchAfter  string    `json:"branchAfter"`
}
```

**回退次数超限处理**：
- 当 `totalCount >= maxAllowed` 时，拒绝回退请求
- 提示用户考虑终止当前 WorkSession 并创建新的 Issue

### 10.2 乐观锁控制

```sql
-- 更新时检查版本
UPDATE work_sessions 
SET current_stage = 'execution', updated_at = NOW()
WHERE id = 'uuid' AND updated_at = '2026-04-03T12:00:00Z';

-- 如果 updated_at 不匹配，说明已被其他操作修改
-- 需要重新读取状态后再次尝试
```
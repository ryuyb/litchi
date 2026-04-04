# DDD 领域设计

## 1. 领域概述

**核心业务**：从 GitHub Issue 到 Pull Request 的全流程自动化开发

**核心理念**：
- 人机协作：Agent 负责执行，人类负责决策和审核
- 可中断恢复：每个阶段都支持中断后从断点继续
- 透明可控：用户可随时介入，Agent 可回退到任意阶段

---

## 2. 限界上下文

### 2.1 Automation Context（核心上下文）

自动化开发流程的核心业务逻辑。

**职责**：
- Issue 处理与需求澄清
- 阶段流转管理
- 设计方案生成与版本管理
- 任务拆解与执行
- PR 创建

### 2.2 Integration Context（集成上下文）

外部系统集成适配。

**职责**：
- GitHub API 交互（Issue、Comment、PR）
- Git 操作（分支、提交、Worktree）
- Agent 调用（Claude Code 等）

### 2.3 上下文映射

```
┌─────────────────────┐         ┌─────────────────────┐
│  Automation        │         │   Integration       │
│  Context           │ ──────▶ │   Context           │
│                    │         │                     │
│  - WorkSession     │         │  - GitHubClient     │
│  - Issue           │         │  - GitOperator      │
│  - Design          │         │  - AgentRunner      │
│  - Task            │         │                     │
└─────────────────────┘         └─────────────────────┘
        │                                │
        │         领域事件驱动            │
        │                                │
        └────────────────────────────────┘
```

---

## 3. 聚合设计

### 3.1 WorkSession 聚合（核心聚合根）

管理一个 Issue 从触发到 PR 创建的完整生命周期，是系统的核心聚合。

**聚合边界**：

```
WorkSession（聚合根）
├── Issue（实体）
├── Clarification（实体）
│   └── ConversationTurn[]（值对象）
├── Design（实体）
│   └── DesignVersion[]（值对象）
├── Task[]（实体）
└── Execution（实体）
    └── FailedTask（值对象）
```

**不变性约束**：
- 一个 WorkSession 对应一个 Issue
- 阶段只能按顺序前进或逐级回退
- 任务执行前必须完成设计
- PR 创建前必须完成所有任务

---

## 4. 实体

### 4.1 Issue

GitHub Issue 的领域表示。

**属性**：
| 属性 | 说明 |
|------|------|
| id | 唯一标识 |
| number | Issue 编号 |
| title | 标题 |
| body | 内容 |
| repository | 所属仓库 |
| author | 创建者（Issue 作者，有权回答澄清问题） |

### 4.2 Clarification

需求澄清过程的状态。

**属性**：
| 属性 | 说明 |
|------|------|
| confirmedPoints | 已确认的需求点 |
| pendingQuestions | 待回答问题列表 |
| history | 对话历史 |
| status | 澄清状态（进行中/已完成） |
| clarityScore | 需求清晰度评分（0-100） |
| clarityDimensions | 各维度清晰度评分详情 |

**业务规则**：
- 所有问题回答完毕才能完成澄清
- 至少确认一个需求点才能进入设计阶段
- 清晰度评分 ≥ 阈值（默认 60）才可自动进入设计阶段
- 清晰度评分 < 阈值需人工确认是否开始设计

---

### 需求清晰度评分体系

#### 评分维度

| 维度 | 权重 | 评分标准 | 检查项 |
|------|------|----------|--------|
| **完整性** | 30% | 需求要素是否齐全 | 见下方检查清单 |
| **明确性** | 25% | 描述是否具体可执行 | 见下方检查清单 |
| **一致性** | 20% | 需求之间是否无冲突 | 见下方检查清单 |
| **可行性** | 15% | 技术实现是否可行 | 见下方检查清单 |
| **可测试性** | 10% | 验收标准是否明确 | 见下方检查清单 |

#### 各维度检查清单

**完整性检查项（30分）**：

| 检查项 | 分值 | 判断标准 |
|--------|------|----------|
| 功能目标明确 | 8分 | 明确要实现什么功能，有具体行为描述 |
| 输入/输出定义 | 6分 | 明确输入数据格式和输出结果格式 |
| 技术约束定义 | 6分 | 明确技术栈、框架、性能要求等约束 |
| 边界条件覆盖 | 5分 | 覆盖异常情况、边界值、错误处理 |
| 依赖关系明确 | 5分 | 明确依赖的模块、服务、外部系统 |

**明确性检查项（25分）**：

| 检查项 | 分值 | 判断标准 |
|--------|------|----------|
| 无模糊词汇 | 8分 | 无"可能"、"大概"、"某种"等模糊词 |
| 具体数据类型 | 6分 | 字段类型、数据结构具体定义 |
| 具体操作行为 | 6分 | 操作步骤、触发条件具体描述 |
| UI/交互明确 | 5分 | 如涉及UI，有具体的交互描述或原型 |

**一致性检查项（20分）**：

| 检查项 | 分值 | 判断标准 |
|--------|------|----------|
| 需求点无冲突 | 8分 | 各需求点之间逻辑一致，无矛盾 |
| 与现有代码兼容 | 7分 | 与现有架构、接口风格一致 |
| 命名/术语一致 | 5分 | 使用统一的术语和命名规范 |

**可行性检查项（15分）**：

| 检查项 | 分值 | 判断标准 |
|--------|------|----------|
| 技术方案可行 | 6分 | 现有技术栈可实现，无未知技术依赖 |
| 资源约束可接受 | 5分 | 预估工作量在可接受范围内 |
| 无阻塞性依赖 | 4分 | 无等待外部完成的阻塞性依赖 |

**可测试性检查项（10分）**：

| 检查项 | 分值 | 判断标准 |
|--------|------|----------|
| 验收标准明确 | 5分 | 有明确的验收条件和成功标准 |
| 测试场景覆盖 | 5分 | 覆盖正常、异常、边界测试场景 |

#### 评分计算公式

```
清晰度总分 = 完整性得分 × 0.30 
           + 明确性得分 × 0.25 
           + 一致性得分 × 0.20 
           + 可行性得分 × 0.15 
           + 可测试性得分 × 0.10
```

#### 评分等级划分

| 分数范围 | 等级 | 处理策略 |
|----------|------|----------|
| 80-100 | 高清晰度 | 自动进入设计阶段，无需确认 |
| 60-79 | 中清晰度 | 自动进入设计阶段，但设计需确认 |
| 40-59 | 低清晰度 | 需人工确认是否开始设计 |
| 0-39 | 不清晰 | 必须继续澄清，不能进入设计 |

#### ClarityDimensions 值对象结构

```json
{
  "completeness": {
    "score": 24,
    "maxScore": 30,
    "checks": {
      "functionalGoal": { "score": 8, "passed": true, "detail": "功能目标明确" },
      "inputOutput": { "score": 6, "passed": true, "detail": "输入输出定义完整" },
      "techConstraints": { "score": 4, "passed": false, "detail": "缺少性能要求" },
      "boundaryConditions": { "score": 3, "passed": false, "detail": "仅覆盖部分边界" },
      "dependencies": { "score": 5, "passed": true, "detail": "依赖关系明确" }
    }
  },
  "clarity": {
    "score": 20,
    "maxScore": 25,
    "checks": {
      "noAmbiguousWords": { "score": 8, "passed": true },
      "specificDataTypes": { "score": 6, "passed": true },
      "specificOperations": { "score": 4, "passed": false, "detail": "部分操作描述模糊" },
      "uiInteraction": { "score": 2, "passed": false, "detail": "UI交互未明确" }
    }
  },
  "consistency": {
    "score": 18,
    "maxScore": 20,
    "checks": { ... }
  },
  "feasibility": {
    "score": 12,
    "maxScore": 15,
    "checks": { ... }
  },
  "testability": {
    "score": 8,
    "maxScore": 10,
    "checks": { ... }
  },
  "totalScore": 72,
  "grade": "中清晰度",
  "canAutoProceed": true
}
```

#### Agent 评分提示模板

```markdown
## 需求清晰度评估任务

请对以下需求进行清晰度评分，按各维度检查项逐项评估。

### 已确认需求点
{confirmedPoints}

### 评估要求
1. 逐项检查每个维度的检查项
2. 给出每项得分（满分/部分分/零分）
3. 未通过的项需说明原因
4. 计算加权总分
5. 给出等级判定和处理建议

### 输出格式（JSON）
{
  "dimensions": { ... },
  "totalScore": 72,
  "grade": "中清晰度",
  "canAutoProceed": true,
  "missingItems": ["性能要求未明确", "UI交互未描述"],
  "suggestedQuestions": ["请明确性能指标要求", "请描述用户交互流程"]
}
```

### 4.3 Design

设计方案及其版本管理。

**属性**：
| 属性 | 说明 |
|------|------|
| versions | 设计版本列表 |
| currentVersion | 当前版本号 |
| complexityScore | 复杂度评分（0-100） |
| requireConfirmation | 是否需要人工确认 |
| confirmed | 是否已确认 |

**业务规则**：
- 每次修改生成新版本，保留历史
- 复杂度超过阈值需人工确认
- 回退到设计阶段时版本号+1

### 4.4 Task

可独立执行的任务单元。

**属性**：
| 属性 | 说明 |
|------|------|
| id | 唯一标识 |
| description | 任务描述 |
| status | 状态（待执行/进行中/完成/失败/跳过） |
| dependencies | 依赖的任务 ID 列表 |
| executionResult | 执行结果 |
| retryCount | 重试次数 |
| failureReason | 失败原因 |
| suggestion | 解决建议 |

**业务规则**：
- 依赖任务完成后才能开始执行
- 失败任务可重试，有最大重试次数限制
- 可跳过失败任务（需用户指令）

### 4.5 Execution

执行阶段的状态追踪。

**属性**：
| 属性 | 说明 |
|------|------|
| worktreePath | Git Worktree 路径 |
| branch | 当前分支 |
| completedTasks | 已完成任务列表 |
| currentTask | 当前执行的任务 |
| failedTask | 失败任务信息 |
| fixTasks | 修复任务列表（PR 回退时追加） |
| rollbackHistory | 回退历史记录 |

**业务规则**：
- 依赖任务完成后才能开始执行
- 失败任务可重试，有最大重试次数限制
- 可跳过失败任务（需用户指令）
- PR 阶段回退到 Execution 时，修复任务追加到 fixTasks

### 4.6 AuditLog

审计日志实体，记录系统中的所有关键操作。

**属性**：
| 属性 | 说明 |
|------|------|
| id | 唯一标识 |
| timestamp | 操作时间 |
| sessionId | 关联的会话 ID |
| repository | 仓库名称 |
| issueNumber | Issue 编号 |
| actor | 操作者 |
| actorRole | 操作者角色（admin / issue_author） |
| operation | 操作类型 |
| resourceType | 资源类型 |
| resourceId | 资源标识 |
| parameters | 操作参数 |
| result | 操作结果（success / failed / denied） |
| duration | 操作耗时 |
| output | 输出摘要 |
| error | 错误信息 |

**角色权限范围**：
| 角色 | 可执行操作 |
|------|-----------|
| issue_author | 回答澄清问题、触发流程 |
| admin | 所有操作（含澄清回答、设计确认、审批、合并等） |

**操作类型**：
| 操作类型 | 说明 |
|---------|------|
| session_start | 会话启动 |
| session_pause | 会话暂停 |
| session_resume | 会话恢复 |
| session_terminate | 会话终止 |
| stage_transition | 阶段转换 |
| agent_call | Agent 调用 |
| tool_use | 工具使用 |
| file_read | 文件读取 |
| file_write | 文件写入 |
| bash_execute | Bash 命令执行 |
| git_operation | Git 操作 |
| pr_create | PR 创建 |
| approval_request | 审批请求 |
| approval_decision | 审批决策 |

**业务规则**：
- 审计日志不可修改、不可删除
- 审计日志保留 90 天（可配置）
- 敏感操作必须记录审计日志
- 输出内容超过配置长度时自动截断

### 4.7 Repository

代码仓库的配置实体。

**属性**：
| 属性 | 说明 |
|------|------|
| id | 唯一标识 |
| name | 仓库名称（格式：owner/repo） |
| enabled | 是否启用 |
| config | 仓库级配置覆盖（JSON） |

**业务规则**：
- 仓库名称必须唯一
- 仓库级配置优先于全局配置
- 禁用仓库后不再处理该仓库的 Webhook

**配置覆盖规则**：
| 配置项 | 说明 | 继承规则 |
|-------|------|---------|
| maxConcurrency | 仓库最大并发数 | 未配置则使用全局配置 |
| complexityThreshold | 复杂度阈值 | 未配置则使用全局配置 |
| forceDesignConfirm | 强制设计确认 | 未配置则使用全局配置 |
| defaultModel | 默认模型 | 未配置则使用全局配置 |
| taskRetryLimit | 任务重试次数限制 | 未配置则使用全局配置 |

**Webhook 路由逻辑**：
1. Webhook 接收时从 payload 提取 `repository.full_name`
2. 查询 repositories 表检查是否已配置且启用
3. 如果不存在或未启用，返回 404 忽略
4. 如果存在，合并仓库级配置与全局配置（仓库级优先）

---

## 5. 值对象

### 5.1 Stage（阶段）

阶段枚举，表示工作流的当前阶段。

```
Clarification → Design → TaskBreakdown → Execution → PullRequest → Completed
```

### 5.2 ComplexityScore（复杂度评分）

0-100 的评分，用于判断是否需要人工确认设计。

**评分维度**：
| 因素 | 权重 |
|------|------|
| 预估代码量 | 30% |
| 涉及模块数 | 25% |
| 破坏性变更 | 25% |
| 测试覆盖难度 | 20% |

### 5.3 TaskStatus（任务状态）

```
Pending → InProgress → Completed / Failed / Skipped
```

### 5.4 DesignVersion（设计版本）

设计方案的单个版本。

**属性**：
| 属性 | 说明 |
|------|------|
| version | 版本号 |
| content | 设计内容 |
| reason | 版本变更原因 |
| createdAt | 创建时间 |

### 5.5 Branch（分支）

Git 分支信息。

**属性**：
| 属性 | 说明 |
|------|------|
| name | 分支名 |
| isDeprecated | 是否已废弃 |

### 5.6 FailedTask（失败任务）

失败任务的详细信息。

**属性**：
| 属性 | 说明 |
|------|------|
| taskId | 任务 ID |
| reason | 失败原因 |
| suggestion | 解决建议 |

### 5.7 DeprecatedBranch（废弃分支）

记录被废弃的分支信息，用于回退操作后的分支管理。

**属性**：
| 属性 | 说明 |
|------|------|
| name | 分支名称 |
| deprecatedAt | 废弃时间 |
| reason | 废弃原因（如：回退到设计阶段） |
| prNumber | 关联的 PR 编号（如有） |
| rollbackToStage | 回退到的目标阶段 |

**业务规则**：
- PR 阶段深层回退（R5、R6）时标记当前分支为废弃
- 浅层回退（R4）不废弃分支，继续在原分支修复
- 废弃分支不删除，保留历史记录供查询

### 5.8 ExecutionResult（执行结果）

任务执行的结果。

**属性**：
| 属性 | 说明 |
|------|------|
| output | 执行输出 |
| testResults | 测试结果列表 |

### 5.9 IdempotencyKey（幂等性键）

用于保证操作幂等性的值对象。

**属性**：
| 属性 | 说明 |
|------|------|
| deliveryId | GitHub Webhook 投递 ID |
| eventType | 事件类型 |
| repository | 仓库名称 |
| createdAt | 创建时间 |
| expiresAt | 过期时间 |

**业务规则**：
- 同一 deliveryId 的 Webhook 只处理一次
- 过期后自动清理（默认 24 小时）
- GitHub 可能重发 Webhook，必须去重

---

### 幂等性保证策略

| 场景 | 幂等性策略 | 实现方式 |
|------|-----------|---------|
| Webhook 重复投递 | delivery_id 去重 | 数据库唯一索引 |
| 重复 @bot 触发 | 检查活跃 Session | 查询 work_sessions 表 |
| 重复阶段转换 | 乐观锁版本控制 | updated_at 版本检查 |
| Agent 调用重复 | Session ID 关联 | session_id 绑定 |

**幂等性设计原则**：
- 所有外部输入（Webhook、用户指令）都必须有唯一标识
- 数据库层面使用唯一索引保证去重
- 业务层面检查当前状态，避免重复操作
- 返回成功状态码，避免外部系统重试

## 6. 领域事件

### 6.1 生命周期事件

| 事件 | 触发时机 |
|------|----------|
| WorkSessionStarted | 创建新的工作会话 |
| WorkSessionPaused | 暂停工作会话 |
| WorkSessionResumed | 恢复工作会话 |
| WorkSessionTerminated | 终止工作会话 |

### 6.2 阶段流转事件

| 事件 | 触发时机 |
|------|----------|
| StageTransitioned | 阶段正常转换 |
| StageRolledBack | 阶段回退 |

### 6.3 澄清阶段事件

| 事件 | 触发时机 |
|------|----------|
| QuestionAsked | Agent 提出问题 |
| QuestionAnswered | Issue 作者或管理员回答问题 |
| ClarificationCompleted | 澄清阶段完成 |

### 6.4 设计阶段事件

| 事件 | 触发时机 |
|------|----------|
| DesignCreated | 创建设计方案版本 |
| DesignApproved | 设计方案获批 |
| DesignRejected | 设计方案被拒绝 |

### 6.5 任务阶段事件

| 事件 | 触发时机 |
|------|----------|
| TaskListCreated | 任务列表创建 |
| TaskStarted | 任务开始执行 |
| TaskCompleted | 任务执行完成 |
| TaskFailed | 任务执行失败 |
| TaskSkipped | 任务被跳过 |

### 6.6 PR 阶段事件

| 事件 | 触发时机 |
|------|----------|
| PullRequestCreated | PR 创建成功 |

### 6.7 用户指令事件

| 事件 | 触发时机 |
|------|----------|
| UserCommandReceived | 收到用户指令（继续/跳过/回退/终止） |

### 6.8 仓库管理事件

| 事件 | 触发时机 |
|------|----------|
| RepositoryAdded | 添加新仓库 |
| RepositoryUpdated | 更新仓库配置 |
| RepositoryEnabled | 启用仓库 |
| RepositoryDisabled | 禁用仓库 |
| RepositoryDeleted | 删除仓库 |

---

## 7. 状态机

### 7.1 阶段状态机

```
                    ┌──────────────┐
                    │              ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│ Clarification│───▶│    Design    │───▶│ TaskBreakdown│
└──────────────┘    └──────────────┘    └──────────────┘
       ▲                   │                   │
       │                   ▼                   ▼
       │            ┌──────────────┐    ┌──────────────┐
       └────────────│   Execution  │◀───│              │
                    └──────────────┘    └──────────────┘
                           │                   │
                           ▼                   │
                    ┌──────────────┐           │
                    │  PullRequest │◀──────────┘
                    └──────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │   Completed  │
                    └──────────────┘
```

### 7.2 任务状态机

```
┌─────────┐    开始    ┌────────────┐
│ Pending │ ─────────▶ │ InProgress │
└─────────┘            └────────────┘
                            │
              ┌─────────────┼─────────────┐
              ▼             ▼             ▼
       ┌──────────┐  ┌──────────┐  ┌──────────┐
       │Completed │  │  Failed  │  │ Skipped  │
       └──────────┘  └──────────┘  └──────────┘
                          │
                          │ 重试
                          ▼
                    ┌────────────┐
                    │ InProgress │
                    └────────────┘
```

### 7.3 回退规则

| 当前阶段 | 可回退到 | 说明 |
|---------|---------|------|
| Execution | Design、Clarification | 执行阶段可回退到设计或澄清 |
| Design | Clarification | 设计阶段可回退到澄清 |
| Clarification | 无 | 澄清阶段不可回退 |
| TaskBreakdown | Design、Clarification | 任务拆解阶段可回退到设计或澄清 |
| PullRequest | Execution、Design、Clarification | PR 阶段支持三级回退 |

**PR 阶段回退详情**：

| 回退类型 | 目标阶段 | 触发指令 | 分支处理 | PR 处理 |
|---------|---------|---------|---------|---------|
| 浅层回退 (R4) | Execution | `@bot 修改代码` | 保留分支 | 保持 open |
| 深层回退 (R5) | Design | `@bot 回退设计` | 废弃分支 | 关闭 PR |
| 最深层回退 (R6) | Clarification | `@bot 回退澄清` | 废弃分支 | 关闭 PR |

**回退行为**：
- 回退到 Design：创建新版本设计方案（v{n+1}），原分支标记为废弃
- 回退到 Clarification：保留已确认的需求点，清空待回答问题
- PR 阶段回退：根据回退深度决定分支和 PR 处理方式

---

## 8. 领域服务

### 8.1 ComplexityEvaluator（复杂度评估服务）

**职责**：评估设计方案的复杂度

**接口**：
```
evaluate(design, codebase) -> ComplexityScore
```

**评估维度**：
- 预估代码量
- 涉及模块数
- 破坏性变更风险
- 测试覆盖难度

### 8.2 StageTransitionService（阶段转换服务）

**职责**：管理阶段转换和回退

**接口**：
```
canTransition(session, toStage) -> bool
transition(session, toStage) -> error
rollback(session, toStage, reason) -> error
```

**业务规则**：
- 只能按顺序前进或逐级回退
- 转换前需检查前置条件
- 回退时保留已确认的上下文

### 8.3 TaskScheduler（任务调度服务）

**职责**：管理任务执行顺序

**接口**：
```
getNextExecutable(tasks, completedTasks) -> Task
markCompleted(task)
markFailed(task, reason, suggestion)
canRetry(task, maxRetry) -> bool
```

**业务规则**：
- 依赖任务完成后才能执行
- 失败任务支持重试
- 超过最大重试次数需用户干预

---

## 9. 仓库接口

### 9.1 WorkSessionRepository

```
save(session) -> error
findById(id) -> WorkSession, error
findByIssueNumber(repo, issueNumber) -> WorkSession, error
findInProgress() -> []WorkSession, error  // 用于恢复
update(session) -> error
```

### 9.2 DesignRepository

```
save(sessionId, design) -> error
findVersions(sessionId) -> []DesignVersion, error
findVersion(sessionId, version) -> DesignVersion, error
```

---

## 10. 持久化策略

### 10.1 状态存储

#### 数据库存储（主存储）

PostgreSQL 数据库存储 WorkSession 及其聚合内所有实体的持久化状态，详见架构设计文档的数据库设计部分。

#### 文件存储（Agent 输入输出缓存）

位置：`.litchi/issues/{issue-id}/context.json`

**用途**：作为 Agent 执行时的输入输出缓存，便于 Agent 快速恢复上下文，减少重复分析。

```json
{
  "sessionId": "uuid",
  "issueId": 123,
  "currentStage": "execution",
  "agentHistory": [...],
  "clarification": {
    "confirmed": ["需求点1", "需求点2"],
    "pending": []
  },
  "design": {
    "currentVersion": 2,
    "requireConfirmation": false,
    "confirmed": true
  },
  "tasks": {
    "total": 4,
    "completed": [1, 2],
    "current": 3,
    "failed": null
  },
  "worktree": "/path/to/worktree",
  "branch": "issue-123-feature",
  "createdAt": "2026-04-03T10:00:00Z",
  "updatedAt": "2026-04-03T12:30:00Z"
}
```

#### 存储关系说明

| 存储类型 | 位置 | 内容 | 用途 |
|---------|------|------|------|
| 数据库 | PostgreSQL | WorkSession 聚合完整状态 | 系统持久化、查询、恢复 |
| 文件 | `.litchi/issues/{issue-id}/` | Agent 输入输出缓存 | Agent 执行上下文、设计方案版本 |

**一致性保证**：每次阶段转换、任务完成/失败时，先更新数据库，再同步更新文件缓存。

### 10.2 设计方案存储

位置：`.litchi/issues/{issue-id}/designs/v{n}.md`

版本化管理，不覆盖历史版本。设计方案文件供 Agent 执行时参考，以及用户查阅历史。

### 10.3 任务列表存储

位置：`.litchi/issues/{issue-id}/tasks.md`

任务列表文件供 Agent 执行时参考当前任务详情。

---

## 11. 并发控制

### 11.1 并发策略

- 多个 Issue 可同时处理
- 每个 Issue 独立的 Git Worktree
- 最大并发数可配置

### 11.2 锁策略

- WorkSession 级别的乐观锁
- 使用 `updatedAt` 版本控制
- 冲突时提示用户

---

## 12. 配置项

| 配置项 | 说明 | 默认值 |
|-------|------|--------|
| maxConcurrency | 最大并发 Issue 数 | 3 |
| complexityThreshold | 复杂度阈值（超过需人工确认设计） | 70 |
| forceDesignConfirm | 强制设计方案人工确认 | false |
| taskRetryLimit | 任务重试次数 | 3 |
| approvalTimeout | 审批等待超时时间 | 24h |
| clarityThreshold | 需求清晰度阈值（低于需人工确认） | 60 |
| clarityAutoProceedThreshold | 清晰度自动进入设计阈值（无需确认） | 80 |
| clarityForceClarifyThreshold | 清晰度强制继续澄清阈值 | 40 |
| allowPRRollback | 是否允许 PR 阶段回退 | true |
| autoFixOnCIFailure | CI 失败时是否自动回退修复 | false |
| maxPRRollbackCount | PR 阶段最大回退次数 | 3 |

### 12.1 PR 回退配置详解

| 配置项 | 类型 | 说明 |
|-------|------|------|
| allowPRRollback | bool | 控制是否允许在 PR 阶段执行回退操作。若设为 false，PR 阶段不可回退，需关闭 PR 后重新创建 |
| autoFixOnCIFailure | bool | 当 CI 检查失败时，是否自动触发 R4 回退（PullRequest → Execution）进行代码修复 |
| maxPRRollbackCount | int | 限制单个 WorkSession 在 PR 阶段的最大回退次数，防止无限循环修改 |

### 12.2 配置使用示例

```yaml
# litchi 配置示例
workflow:
  maxConcurrency: 3
  taskRetryLimit: 3
  
  clarity:
    threshold: 60
    autoProceedThreshold: 80
    forceClarifyThreshold: 40
  
  design:
    complexityThreshold: 70
    forceConfirm: false
  
  pr:
    allowRollback: true
    autoFixOnCIFailure: false
    maxRollbackCount: 3
```
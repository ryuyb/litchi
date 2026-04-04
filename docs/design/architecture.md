# 架构设计文档

## 1. 系统概述

### 1.1 项目名称

**Litchi** - 自动化开发 Agent 系统

### 1.2 系统目标

实现从 GitHub Issue 到 Pull Request 的全流程自动化开发，支持人机协作、可中断恢复、透明可控。

---

## 2. 系统架构

### 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Presentation Layer                              │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                         React + shadcn/ui                              │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │  │
│  │  │  仪表盘   │  │ Issue列表│  │ 进度监控 │  │ 配置管理 │  │ 日志查看 │  │  │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘  │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        │ HTTP/WebSocket
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Application Layer                               │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                           Go HTTP Server                               │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │  │
│  │  │ REST API │  │WebSocket │  │ Webhook  │  │ Scheduler│  │ Recovery │  │  │
│  │  │ Handler  │  │ Handler  │  │ Handler  │  │ Service  │  │ Service  │  │  │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘  │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                        │                                     │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                         Application Services                           │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐  │  │
│  │  │IssueService  │  │StageService  │  │ TaskService  │  │PRService   │  │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  └────────────┘  │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                                Domain Layer                                  │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                            Aggregates                                   │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐   │  │
│  │  │                      WorkSession                                  │   │  │
│  │  │  ┌────────┐ ┌─────────────┐ ┌────────┐ ┌────────┐ ┌───────────┐  │   │  │
│  │  │  │ Issue  │ │Clarification│ │ Design │ │ Task[] │ │ Execution │  │   │  │
│  │  │  └────────┘ └─────────────┘ └────────┘ └────────┘ └───────────┘  │   │  │
│  │  └──────────────────────────────────────────────────────────────────┘   │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                         Domain Services                                 │  │
│  │  ┌──────────────────┐  ┌────────────────────┐  ┌───────────────────┐   │  │
│  │  │ComplexityEvaluator│  │StageTransitionService│  │  TaskScheduler  │   │  │
│  │  └──────────────────┘  └────────────────────┘  └───────────────────┘   │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                        Domain Events                                    │  │
│  │  WorkSessionStarted │ StageTransitioned │ TaskCompleted │ ...          │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Infrastructure Layer                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │  GitHub     │  │    Git      │  │   Agent     │  │   Persistence       │  │
│  │  Client     │  │  Operator   │  │   Runner    │  │                     │  │
│  │             │  │             │  │             │  │  ┌───────────────┐  │  │
│  │ - Issue API │  │ - Branch    │  │ - Claude    │  │  │  PostgreSQL   │  │  │
│  │ - PR API    │  │ - Commit    │  │ - (extensible)│  │  │     + GORM    │  │  │
│  │ - Webhook   │  │ - Worktree  │  │             │  │  └───────────────┘  │  │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────────────┘  │
│                                        │                                     │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                           Cross-cutting                                 │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │  │
│  │  │    Viper    │  │     Zap     │  │  Metrics   │  │    Tracing     │  │  │
│  │  │  (Config)   │  │  (Logging)  │  │ (Prometheus)│  │  (OpenTelemetry)│  │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────────┘  │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 架构分层

| 层级 | 职责 | 依赖方向 |
|------|------|----------|
| Presentation | 用户界面展示 | → Application |
| Application | 用例编排、请求处理 | → Domain |
| Domain | 核心业务逻辑、领域模型 | 无外部依赖 |
| Infrastructure | 外部服务适配、持久化 | → Domain |

---

## 3. 技术栈选型

### 3.1 后端技术栈

| 组件 | 技术选型 | 说明 |
|------|---------|------|
| 语言 | Go 1.22+ | 高性能、并发友好 |
| Web 框架 | Gin / Echo | REST API 和 WebSocket 支持 |
| 配置管理 | Viper | 支持 YAML/JSON/ENV，热重载 |
| 日志 | Zap | 高性能结构化日志 |
| ORM | GORM | PostgreSQL 支持，迁移方便 |
| 数据库 | PostgreSQL 16 | JSON 支持、事务可靠 |
| 消息队列 | 内置 Channel / Redis Stream | 事件驱动（可扩展） |
| 定时任务 | cron | 恢复中断任务、超时处理 |

### 3.2 前端技术栈

| 组件 | 技术选型 | 说明 |
|------|---------|------|
| 框架 | React 18 | 组件化开发 |
| 构建工具 | Vite | 快速开发体验 |
| UI 组件库 | shadcn/ui | 基于 Radix UI，可定制 |
| 样式 | Tailwind CSS | 原子化 CSS |
| 状态管理 | Zustand | 轻量状态管理 |
| 数据请求 | TanStack Query | 缓存、自动刷新 |
| 路由 | React Router v6 | 声明式路由 |

### 3.3 外部依赖

| 组件 | 说明 |
|------|------|
| GitHub API | Issue/PR/Comment 操作 |
| Git | 分支管理、Worktree、提交 |
| Claude Code | AI Agent 执行 |

---

## 4. 模块设计

### 4.1 后端目录结构

```
cmd/
├── server/
│   └── main.go              # HTTP 服务入口
└── worker/
    └── main.go              # 后台任务 Worker（可选独立进程）

internal/
├── domain/                   # 领域层
│   ├── aggregate/            # 聚合根
│   │   └── work_session.go
│   ├── entity/               # 实体
│   │   ├── issue.go
│   │   ├── clarification.go
│   │   ├── design.go
│   │   ├── task.go
│   │   └── execution.go
│   ├── valueobject/          # 值对象
│   │   ├── stage.go
│   │   ├── task_status.go
│   │   ├── complexity_score.go
│   │   └── ...
│   ├── event/                # 领域事件
│   │   └── domain_event.go
│   ├── service/              # 领域服务
│   │   ├── complexity_evaluator.go
│   │   ├── stage_transition.go
│   │   └── task_scheduler.go
│   └── repository/           # 仓库接口
│       └── work_session_repository.go
│
├── application/              # 应用层
│   ├── service/               # 应用服务
│   │   ├── issue_service.go
│   │   ├── stage_service.go
│   │   ├── task_service.go
│   │   └── pr_service.go
│   ├── handler/               # HTTP 处理器
│   │   ├── rest/
│   │   │   ├── issue_handler.go
│   │   │   ├── session_handler.go
│   │   │   └── config_handler.go
│   │   └── websocket/
│   │       └── progress_handler.go
│   ├── dto/                   # 数据传输对象
│   │   ├── request/
│   │   └── response/
│   └── event/                 # 应用层事件处理
│       ├── dispatcher.go
│       └── handlers/
│
├── infrastructure/            # 基础设施层
│   ├── persistence/           # 持久化
│   │   ├── postgres/
│   │   │   ├── connection.go
│   │   │   ├── migrations/
│   │   │   └── repositories/
│   │   │       └── work_session_repo.go
│   │   └── models/            # GORM 模型
│   │       └── work_session_model.go
│   ├── github/                # GitHub 集成
│   │   ├── client.go
│   │   ├── issue.go
│   │   ├── pull_request.go
│   │   └── webhook.go
│   ├── git/                   # Git 操作
│   │   ├── operator.go
│   │   ├── branch.go
│   │   └── worktree.go
│   ├── agent/                 # Agent 集成
│   │   ├── runner.go
│   │   └── claude/
│   │       └── claude_runner.go
│   └── config/                # 配置
│       └── config.go
│
├── pkg/                       # 公共包
│   ├── logger/
│   │   └── zap.go
│   ├── errors/
│   │   └── errors.go
│   └── utils/
│
└── interfaces/                # 接口定义（可选）
    └── agent_interface.go
```

### 4.2 前端目录结构

```
web/
├── public/
├── src/
│   ├── main.tsx
│   ├── App.tsx
│   ├── index.css
│   │
│   ├── components/            # 组件
│   │   ├── ui/                # shadcn/ui 组件
│   │   │   ├── button.tsx
│   │   │   ├── card.tsx
│   │   │   └── ...
│   │   ├── layout/            # 布局组件
│   │   │   ├── Sidebar.tsx
│   │   │   ├── Header.tsx
│   │   │   └── MainLayout.tsx
│   │   └── features/          # 业务组件
│   │       ├── IssueCard/
│   │       ├── StageProgress/
│   │       ├── TaskList/
│   │       ├── LogViewer/
│   │       ├── AuditLogFilter/     # 审计日志筛选组件
│   │       ├── AuditLogTable/      # 审计日志表格组件
│   │       └── AuditLogDetail/     # 审计日志详情组件
│   │
│   ├── pages/                 # 页面
│   │   ├── Dashboard.tsx
│   │   ├── IssueList.tsx
│   │   ├── IssueDetail.tsx
│   │   ├── Settings.tsx
│   │   ├── Repositories.tsx       # 仓库列表页
│   │   ├── RepositoryConfig.tsx   # 仓库配置页
│   │   ├── AuditLogs.tsx          # 审计日志列表页
│   │   └── AuditLogDetail.tsx     # 审计日志详情页
│   │
│   ├── hooks/                 # 自定义 Hooks
│   │   ├── useSession.ts
│   │   ├── useWebSocket.ts
│   │   └── useConfig.ts
│   │
│   ├── stores/                # Zustand 状态
│   │   ├── sessionStore.ts
│   │   └── configStore.ts
│   │
│   ├── services/              # API 服务
│   │   ├── api.ts
│   │   ├── issueService.ts
│   │   └── sessionService.ts
│   │
│   ├── types/                 # TypeScript 类型
│   │   ├── domain.ts
│   │   └── api.ts
│   │
│   └── lib/                   # 工具函数
│       └── utils.ts
│
├── components.json            # shadcn/ui 配置
├── tailwind.config.js
├── vite.config.ts
├── tsconfig.json
└── package.json
```

---

## 5. 核心流程设计

### 5.1 Issue 触发流程

#### Webhook 处理流程（含去重）

```
GitHub Webhook 请求
    │
    ▼
┌─────────────────┐
│ 1. 签名验证     │ ← 验证 X-Hub-Signature-256
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 2. 去重检查     │ ← 检查 X-GitHub-Delivery 是否已处理
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
   已处理    未处理
    │         │
    ▼         ▼
 返回 200   继续处理
 (忽略)         │
                ▼
         ┌─────────────────┐
         │ 3. 仓库检查     │ ← 检查仓库是否已配置且启用
         └────────┬────────┘
                  │
             ┌────┴────┐
             │         │
          未配置      已配置
             │         │
             ▼         ▼
          返回 404   继续处理
          (忽略)         │
                         ▼
                  ┌─────────────────┐
                  │ 4. 权限检查     │ ← 检查触发者权限
                  └────────┬────────┘
                           │
                          ...
```

#### 触发流程详细步骤

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   GitHub     │────▶│   Webhook    │────▶│   去重       │
│   Webhook    │     │   Handler    │     │   检查       │
└──────────────┘     └──────────────┘     └──────────────┘
                                                 │
                                                 ▼
                      ┌──────────────┐     ┌──────────────┐
                      │   记录       │◀────│   签名       │
                      │   Delivery   │     │   验证       │
                      └──────────────┘     └──────────────┘
                                                 │
                                                 ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  WorkSession │◀────│   Create     │◀────│   Validate   │
│  Repository  │     │   Session    │     │   Trigger    │
└──────────────┘     └──────────────┘     └──────────────┘
                                                 │
                                                 ▼
                                         ┌──────────────┐
                                         │   Publish    │
                                         │   Event      │
                                         └──────────────┘
```

### 5.2 阶段流转流程

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Event-Driven Flow                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐         │
│  │ Command │───▶│ Service │───▶│ Domain  │───▶│ Event   │         │
│  │         │    │         │    │ Object  │    │         │         │
│  └─────────┘    └─────────┘    └─────────┘    └────┬────┘         │
│                                                     │               │
│                                                     ▼               │
│                                              ┌─────────────┐       │
│                                              │   Event     │       │
│                                              │   Dispatcher│       │
│                                              └─────────────┘       │
│                                                     │               │
│                    ┌────────────────────────────────┼───────────┐   │
│                    │                                │           │   │
│                    ▼                                ▼           ▼   │
│           ┌─────────────┐                 ┌─────────────┐ ┌───────┐│
│           │ GitHub      │                 │  WebSocket  │ │Logging││
│           │ Notifier    │                 │  Push       │ │       ││
│           └─────────────┘                 └─────────────┘ └───────┘│
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 5.3 任务执行流程

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│ TaskService  │────▶│ TaskScheduler│────▶│  Get Next    │
│              │     │              │     │  Task        │
└──────────────┘     └──────────────┘     └──────────────┘
                                                   │
                                                   ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Update    │◀────│   Execute    │◀────│ AgentRunner  │
│   Status    │     │   Task       │     │              │
└──────────────┘     └──────────────┘     └──────────────┘
       │                   │
       │            ┌──────┴──────┐
       │            ▼             ▼
       │     ┌──────────┐  ┌──────────┐
       │     │ Success  │  │  Failed  │
       │     └──────────┘  └──────────┘
       │            │             │
       │            ▼             ▼
       │     ┌──────────┐  ┌──────────┐
       │     │ Run Test │  │  Retry/  │
       │     └──────────┘  │  Notify  │
       │            │      └──────────┘
       │            ▼
       │     ┌──────────┐
       └────▶│ Complete │
             └──────────┘
```

---

## 6. API 设计

### 6.1 REST API

#### 会话管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/sessions` | 获取会话列表 |
| GET | `/api/v1/sessions/:id` | 获取会话详情 |
| POST | `/api/v1/sessions` | 创建会话（手动触发） |
| POST | `/api/v1/sessions/:id/pause` | 暂停会话 |
| POST | `/api/v1/sessions/:id/resume` | 恢复会话 |
| POST | `/api/v1/sessions/:id/rollback` | 回退到指定阶段 |
| POST | `/api/v1/sessions/:id/terminate` | 终止会话 |

#### 任务管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/sessions/:id/tasks` | 获取任务列表 |
| POST | `/api/v1/sessions/:id/tasks/:taskId/skip` | 跳过任务 |
| POST | `/api/v1/sessions/:id/tasks/:taskId/retry` | 重试任务 |

#### 配置管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/config` | 获取配置 |
| PUT | `/api/v1/config` | 更新配置 |

#### 仓库管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/repositories` | 获取仓库列表 |
| POST | `/api/v1/repositories` | 添加仓库 |
| GET | `/api/v1/repositories/:id` | 获取仓库详情 |
| PUT | `/api/v1/repositories/:id` | 更新仓库配置 |
| DELETE | `/api/v1/repositories/:id` | 删除仓库 |
| GET | `/api/v1/repositories/:id/validation-config` | 获取执行验证配置 |
| PUT | `/api/v1/repositories/:id/validation-config` | 更新执行验证配置 |
| POST | `/api/v1/repositories/:id/detect-project` | 触发项目检测 |
| GET | `/api/v1/repositories/:id/detection-result` | 获取检测结果 |

#### Webhook

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/webhooks/github` | GitHub Webhook 接收 |

#### 审计日志

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/audit-logs` | 查询审计日志列表 |
| GET | `/api/v1/audit-logs/:id` | 获取审计日志详情 |
| GET | `/api/v1/sessions/:id/audit-logs` | 获取指定会话的审计日志 |
| GET | `/api/v1/repositories/:id/audit-logs` | 获取指定仓库的审计日志 |

**查询参数**：
- `startTime` / `endTime`：时间范围
- `repository`：仓库名称过滤
- `operation`：操作类型过滤
- `actor`：操作者过滤
- `result`：结果过滤（success/failed/denied）
- `page` / `pageSize`：分页

**Webhook 去重处理逻辑**：

```go
// WebhookHandler 处理 Webhook
func (h *WebhookHandler) HandleGitHubWebhook(c *gin.Context) {
    // 1. 签名验证
    if !h.verifySignature(c) {
        c.JSON(401, gin.H{"error": "invalid signature"})
        return
    }

    // 2. 去重检查
    deliveryID := c.GetHeader("X-GitHub-Delivery")
    if h.isProcessed(deliveryID) {
        // 已处理，直接返回成功
        c.JSON(200, gin.H{"status": "already_processed"})
        return
    }

    // 3. 记录 delivery_id
    eventType := c.GetHeader("X-GitHub-Event")
    repository := h.extractRepository(c)
    h.recordDelivery(deliveryID, eventType, repository)

    // 4. 继续处理...
    h.processWebhook(c, deliveryID, eventType)
}

// isProcessed 检查是否已处理
func (h *WebhookHandler) isProcessed(deliveryID string) bool {
    var delivery WebhookDelivery
    err := h.db.Where("delivery_id = ? AND processed = ?", deliveryID, true).First(&delivery).Error
    return err == nil
}

// recordDelivery 记录投递
func (h *WebhookHandler) recordDelivery(deliveryID, eventType, repository string) {
    h.db.Create(&WebhookDelivery{
        DeliveryID:  deliveryID,
        EventType:   eventType,
        Repository:  repository,
        Processed:   true,
        ExpiresAt:   time.Now().Add(h.config.IdempotencyTTL),
    })
}
```

**去重保证机制**：
- 使用数据库唯一索引保证 `delivery_id` 不重复
- 去重检查在签名验证之后、业务处理之前
- 已处理的 Webhook 返回 200 状态码（避免 GitHub 重试）
- 过期记录自动清理，避免表无限增长

### 6.2 WebSocket

| 路径 | 说明 |
|------|------|
| `/ws/sessions/:id` | 会话实时进度推送 |

**消息格式**：

```json
{
  "type": "stage_transitioned",
  "payload": {
    "sessionId": "uuid",
    "from": "clarification",
    "to": "design",
    "timestamp": "2026-04-03T10:00:00Z"
  }
}
```

**事件类型**：
- `stage_transitioned` - 阶段转换
- `task_started` - 任务开始
- `task_completed` - 任务完成
- `task_failed` - 任务失败
- `question_asked` - 提出问题
- `design_created` - 设计创建
- `pr_created` - PR 创建

---

## 7. 数据库设计

### 7.1 数据库与文件存储关系

| 存储类型 | 位置 | 内容 | 用途 |
|---------|------|------|------|
| **数据库** | PostgreSQL | WorkSession 聚合完整状态 | 系统持久化、查询统计、崩溃恢复 |
| **文件缓存** | `.litchi/issues/{issue-id}/` | Agent 输入输出、设计方案版本 | Agent 执行上下文、用户查阅 |

**目录结构**：
```
.litchi/
└── issues/
    └── {issue-id}/
        ├── designs/
        │   ├── v1.md
        │   ├── v2.md
        │   └── ...
        ├── tasks.md
        └── context.json
```

**一致性保证**：
- 状态变更时先更新数据库，再同步更新文件缓存
- 服务启动时从数据库恢复状态，验证文件缓存一致性

### 7.2 ER 图

```
┌──────────────────┐       ┌──────────────────┐       ┌──────────────────┐
│   repositories   │       │   work_sessions  │       │      issues      │
├──────────────────┤       ├──────────────────┤       ├──────────────────┤
│ id (PK)          │       │ id (PK)          │───┐   │ id (PK)          │
│ name             │       │ issue_id (FK)    │   │   │ number           │
│ enabled          │       │ current_stage    │   │   │ title            │
│ config (JSONB)   │       │ status           │   │   │ body             │
│ created_at       │       │ created_at       │   │   │ repository (FK)  │───┐
│ updated_at       │       │ updated_at       │   │   │ author           │   │
└──────────────────┘       └──────────────────┘   │   │ created_at       │   │
         │                          │             │   └──────────────────┘   │
         │                          │             │                          │
         │                          │             └──────────────────────────┘
         │                          │                                      │
         │                          ▼                                      │
         │                  ┌──────────────────┐                           │
         │                  │ clarifications   │                           │
         │                  ├──────────────────┤                           │
         │                  │ id (PK)          │                           │
         │                  │ session_id (FK)  │                           │
         │                  │ confirmed_points │                           │
         │                  │ pending_questions│                           │
         │                  │ status           │                           │
         │                  │ created_at       │                           │
         │                  └──────────────────┘                           │
         │                          │                                      │
         │                          │                                      │
         │                          ▼                                      │
         │                  ┌──────────────────┐       ┌──────────────────┐│
         │                  │     designs      │       │ design_versions  ││
         │                  ├──────────────────┤       ├──────────────────┤│
         │                  │ id (PK)          │───┐   │ id (PK)          ││
         │                  │ session_id (FK)  │   │   │ design_id (FK)   ││
         │                  │ current_version  │   │   │ version          ││
         │                  │ complexity_score │   │   │ content          ││
         │                  │ require_confirm  │   │   │ reason           ││
         │                  │ confirmed        │   │   │ created_at       ││
         │                  │ created_at       │   │   └──────────────────┘│
         │                  └──────────────────┘   │                       │
         │                                         │                       │
         │                  ┌─────────────────────┴─────────────────────┐  │
         │                  │                                           │  │
         │                  ▼                                           ▼  │
         │          ┌──────────────────┐                       ┌──────────────────┐
         │          │      tasks       │                       │   executions     │
         │          ├──────────────────┤                       ├──────────────────┤
         │          │ id (PK)          │                       │ id (PK)          │
         │          │ session_id (FK)  │                       │ session_id (FK)  │
         │          │ description      │                       │ worktree_path    │
         │          │ status           │                       │ branch_name      │
         │          │ dependencies     │                       │ branch_deprecated│
         │          │ retry_count      │                       │ current_task_id  │
         │          │ failure_reason   │                       │ failed_task_id   │
         │          │ suggestion       │                       │ created_at       │
         │          │ created_at       │                       └──────────────────┘
         │          └──────────────────┘                                │
         │                  │                                           │
         │                  │                                           │
         │                  ▼                                           │
         │          ┌──────────────────┐                                │
         │          │  task_results    │                                │
         │          ├──────────────────┤                                │
         │          │ id (PK)          │                                │
         │          │ task_id (FK)     │                                │
         │          │ output           │                                │
         │          │ test_results     │                                │
         │          │ created_at       │                                │
         │          └──────────────────┘                                │
         │                                                          │   │
         └──────────────────────────────────────────────────────────┘───┘

         注：issues.repository 外键关联 repositories.name
```

### 7.3 表结构

#### repositories 表

```sql
CREATE TABLE repositories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,  -- 如 "org/repo"
    enabled BOOLEAN DEFAULT true,
    config JSONB DEFAULT '{}',           -- 仓库级配置覆盖
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_repositories_name ON repositories(name);
CREATE INDEX idx_repositories_enabled ON repositories(enabled);
```

**config 字段结构**：

```json
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
        "detectedTools": [...]
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

**配置覆盖规则**：
- 仓库级配置优先于全局配置
- 未配置的项使用全局配置默认值
- 支持的配置项：
  - `maxConcurrency`: 仓库最大并发数
  - `complexityThreshold`: 复杂度阈值
  - `forceDesignConfirm`: 强制设计确认
  - `defaultModel`: 默认模型
  - `taskRetryLimit`: 任务重试次数限制
  - `executionValidation`: 执行验证配置（详见 `execution-validation-design.md`）

#### work_sessions 表

```sql
CREATE TABLE work_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    issue_id UUID NOT NULL REFERENCES issues(id),
    current_stage VARCHAR(50) NOT NULL DEFAULT 'clarification',
    status VARCHAR(50) NOT NULL DEFAULT 'active',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_issue FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE
);

CREATE INDEX idx_work_sessions_issue_id ON work_sessions(issue_id);
CREATE INDEX idx_work_sessions_status ON work_sessions(status);
```

#### issues 表

```sql
CREATE TABLE issues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    number BIGINT NOT NULL,
    title VARCHAR(500) NOT NULL,
    body TEXT,
    repository VARCHAR(255) NOT NULL REFERENCES repositories(name),
    author VARCHAR(255) NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(repository, number)
);

CREATE INDEX idx_issues_repository ON issues(repository);
CREATE INDEX idx_issues_number ON issues(number);
```

#### clarifications 表

```sql
CREATE TABLE clarifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES work_sessions(id) ON DELETE CASCADE,
    confirmed_points JSONB DEFAULT '[]',
    pending_questions JSONB DEFAULT '[]',
    conversation_history JSONB DEFAULT '[]',
    status VARCHAR(50) NOT NULL DEFAULT 'in_progress',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_session FOREIGN KEY (session_id) REFERENCES work_sessions(id) ON DELETE CASCADE
);
```

#### designs 表

```sql
CREATE TABLE designs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES work_sessions(id) ON DELETE CASCADE,
    current_version INT NOT NULL DEFAULT 0,
    complexity_score INT,
    require_confirmation BOOLEAN DEFAULT FALSE,
    confirmed BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_session FOREIGN KEY (session_id) REFERENCES work_sessions(id) ON DELETE CASCADE
);
```

#### design_versions 表

```sql
CREATE TABLE design_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    design_id UUID NOT NULL REFERENCES designs(id) ON DELETE CASCADE,
    version INT NOT NULL,
    content TEXT NOT NULL,
    reason VARCHAR(500),

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_design FOREIGN KEY (design_id) REFERENCES designs(id) ON DELETE CASCADE,
    UNIQUE(design_id, version)
);
```

#### tasks 表

```sql
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES work_sessions(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    dependencies UUID[] DEFAULT '{}',
    retry_count INT DEFAULT 0,
    failure_reason TEXT,
    suggestion TEXT,
    "order" INT NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_session FOREIGN KEY (session_id) REFERENCES work_sessions(id) ON DELETE CASCADE
);

CREATE INDEX idx_tasks_session_id ON tasks(session_id);
CREATE INDEX idx_tasks_status ON tasks(status);
```

#### executions 表

```sql
CREATE TABLE executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES work_sessions(id) ON DELETE CASCADE,
    worktree_path VARCHAR(500),
    branch_name VARCHAR(255),
    branch_deprecated BOOLEAN DEFAULT FALSE,
    current_task_id UUID,
    failed_task_id UUID,
    completed_task_ids UUID[] DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_session FOREIGN KEY (session_id) REFERENCES work_sessions(id) ON DELETE CASCADE
);
```

#### task_results 表

```sql
CREATE TABLE task_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    output TEXT,
    test_results JSONB DEFAULT '[]',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT fk_task FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);
```

#### execution_validation_results 表（执行验证结果）

记录每次任务执行后的验证结果。

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

#### domain_events 表（领域事件存储）

```sql
CREATE TABLE domain_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id UUID NOT NULL,
    aggregate_type VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,

    occurred_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_domain_events_aggregate ON domain_events(aggregate_id, aggregate_type);
CREATE INDEX idx_domain_events_type ON domain_events(event_type);
CREATE INDEX idx_domain_events_occurred_at ON domain_events(occurred_at);
```

#### audit_logs 表（审计日志）

```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    session_id UUID REFERENCES work_sessions(id) ON DELETE SET NULL,
    repository VARCHAR(255) NOT NULL,
    issue_number BIGINT,

    -- 操作者
    actor VARCHAR(255) NOT NULL,           -- 触发者（GitHub 用户名）
    actor_role VARCHAR(50),                 -- 角色：admin / issue_author

    -- 操作详情
    operation VARCHAR(100) NOT NULL,        -- 操作类型
    resource_type VARCHAR(100),             -- 资源类型
    resource_id VARCHAR(255),               -- 资源标识

    -- 结果
    result VARCHAR(50) NOT NULL,            -- success / failed / denied
    duration_ms BIGINT,                     -- 操作耗时（毫秒）

    -- 详细信息
    parameters JSONB,                       -- 操作参数
    output TEXT,                            -- 输出摘要（截断）
    error_message TEXT,                     -- 错误信息

    -- 索引优化字段
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_session ON audit_logs(session_id);
CREATE INDEX idx_audit_logs_repository ON audit_logs(repository);
CREATE INDEX idx_audit_logs_operation ON audit_logs(operation);
CREATE INDEX idx_audit_logs_actor ON audit_logs(actor);
CREATE INDEX idx_audit_logs_result ON audit_logs(result);

-- 分区（可选，数据量大时）
-- CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
```

#### webhook_deliveries 表（Webhook 投递记录）

用于 Webhook 去重，保证幂等性。

```sql
-- Webhook 投递记录表（用于去重）
CREATE TABLE webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id VARCHAR(255) NOT NULL UNIQUE,  -- GitHub delivery ID
    event_type VARCHAR(100) NOT NULL,          -- 事件类型：issues, issue_comment, etc.
    repository VARCHAR(255) NOT NULL,          -- 仓库名称
    payload_hash VARCHAR(64),                  -- payload SHA256 哈希（可选校验）
    processed BOOLEAN DEFAULT false,           -- 是否已处理
    process_result VARCHAR(50),                -- 处理结果：success / ignored / error
    process_message TEXT,                      -- 处理消息
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE        -- 过期时间
);

-- 索引
CREATE INDEX idx_webhook_deliveries_delivery_id ON webhook_deliveries(delivery_id);
CREATE INDEX idx_webhook_deliveries_created_at ON webhook_deliveries(created_at);
CREATE INDEX idx_webhook_deliveries_expires_at ON webhook_deliveries(expires_at);

-- 自动清理过期记录（可选）
-- CREATE INDEX idx_webhook_deliveries_processed ON webhook_deliveries(processed, expires_at);
```

**说明**：
- `delivery_id` 字段有唯一索引，保证同一 Webhook 不会重复处理
- `expires_at` 设置过期时间（默认 24 小时），支持自动清理
- `payload_hash` 可用于校验 payload 内容一致性
- `process_result` 记录处理结果，便于排查问题

---

## 8. 配置管理

### 8.1 配置文件结构

```yaml
# config.yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "debug"  # debug, release, test

database:
  host: "localhost"
  port: 5432
  name: "litchi"
  user: "postgres"
  password: "${DB_PASSWORD}"
  sslmode: "disable"
  max_open_conns: 100
  max_idle_conns: 10

github:
  token: "${GITHUB_TOKEN}"
  webhook_secret: "${GITHUB_WEBHOOK_SECRET}"
  app_id: "${GITHUB_APP_ID}"
  private_key_path: "${GITHUB_PRIVATE_KEY_PATH}"

webhook:
  idempotency:
    enabled: true                    # 是否启用去重
    ttl: 24h                         # delivery ID 保留时间
    autoCleanup: true                # 是否自动清理过期记录
    cleanupInterval: 1h              # 清理间隔

agent:
  type: "claude-code"
  maxConcurrency: 3
  taskRetryLimit: 3
  approvalTimeout: "24h"

clarity:
  threshold: 60                # 清晰度阈值（低于需人工确认）
  autoProceedThreshold: 80     # 清晰度自动进入设计阈值（无需确认）
  forceClarifyThreshold: 40    # 清晰度强制继续澄清阈值

complexity:
  threshold: 70                # 复杂度阈值（超过需人工确认设计）
  forceDesignConfirm: false    # 强制设计方案人工确认

# 审计日志配置
audit:
  enabled: true                      # 是否启用审计日志
  retentionDays: 90                  # 审计日志保留天数
  maxOutputLength: 1000              # 输出截断长度
  sensitiveOperations:               # 必须审计的操作
    - agent_call
    - tool_use
    - file_read
    - file_write
    - bash_execute
    - git_operation
    - pr_create
    - approval_request
    - approval_decision
    - stage_transition

# 失败处理配置
failure:
  retry:
    maxRetries: 3
    initialBackoff: 5s
    maxBackoff: 60s
    backoffMultiplier: 2.0
    retryableErrors:
      - test_failed
      - build_failed
      - timeout
      - network_error
      - github_rate_limit

  rateLimit:
    enabled: true
    waitEnabled: true
    maxWaitDuration: 30m
    notifyThreshold: 10

  timeout:
    clarificationAgent: 5m
    designAnalysis: 10m
    designGeneration: 15m
    taskBreakdown: 10m
    taskExecution: 30m
    testRun: 10m
    prCreation: 5m
    approvalWait: 24h
    sessionMaxDuration: 72h

  queue:
    maxLength: 10
    priorityEnabled: true
    timeoutOnQueue: 1h

  testEnvironment:
    skipIfNoTests: true
    skipIfUnavailable: false
    checkInterval: 5m

logging:
  level: "info"
  format: "json"
  output: "stdout"

redis:
  enabled: false
  addr: "localhost:6379"
  password: ""
  db: 0
```

### 8.2 配置加载

```go
// internal/infrastructure/config/config.go
type Config struct {
    Server     ServerConfig     `mapstructure:"server"`
    Database   DatabaseConfig   `mapstructure:"database"`
    GitHub     GitHubConfig     `mapstructure:"github"`
    Agent      AgentConfig      `mapstructure:"agent"`
    Complexity ComplexityConfig `mapstructure:"complexity"`
    Logging    LoggingConfig    `mapstructure:"logging"`
    Redis      RedisConfig      `mapstructure:"redis"`
}

func Load() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath(".")
    viper.AddConfigPath("./config")
    viper.AutomaticEnv()

    if err := viper.ReadInConfig(); err != nil {
        return nil, err
    }

    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }

    return &cfg, nil
}
```

---

## 9. 日志设计

### 9.1 日志规范

```go
// 使用 Zap 结构化日志
logger.Info("task completed",
    zap.String("session_id", sessionID),
    zap.String("task_id", taskID),
    zap.Duration("duration", duration),
)

logger.Error("task failed",
    zap.String("session_id", sessionID),
    zap.String("task_id", taskID),
    zap.Error(err),
    zap.String("suggestion", suggestion),
)
```

### 9.2 日志级别使用

| 级别 | 使用场景 |
|------|---------|
| DEBUG | 详细调试信息（开发环境） |
| INFO | 正常业务流程（阶段转换、任务完成） |
| WARN | 可恢复的异常（重试、回退） |
| ERROR | 业务错误（任务失败、Agent 错误） |
| FATAL | 系统级错误（数据库连接失败） |

---

## 10. 部署架构

### 10.1 单机部署

```
┌─────────────────────────────────────────────────────────┐
│                      Docker Compose                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │
│  │   Nginx     │  │   Litchi    │  │   PostgreSQL   │  │
│  │   (Proxy)   │  │   Server    │  │                 │  │
│  └─────────────┘  └─────────────┘  └─────────────────┘  │
│         │                │                    │         │
│         └────────────────┼────────────────────┘         │
│                          │                              │
│                    ┌─────┴─────┐                       │
│                    │  Volumes  │                       │
│                    └───────────┘                       │
└─────────────────────────────────────────────────────────┘
```

### 10.2 Kubernetes 部署（可选）

```
┌─────────────────────────────────────────────────────────────────┐
│                        Kubernetes Cluster                        │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                        Ingress                              │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                              │                                   │
│              ┌───────────────┼───────────────┐                   │
│              ▼               ▼               ▼                   │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │  Litchi     │  │  Litchi     │  │  Litchi     │              │
│  │  Pod 1      │  │  Pod 2      │  │  Pod 3      │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
│              │               │               │                   │
│              └───────────────┼───────────────┘                   │
│                              │                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                    PostgreSQL (StatefulSet)                 │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### 10.3 Docker Compose 示例

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: litchi
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  litchi:
    build: .
    environment:
      - DB_PASSWORD=postgres
      - GITHUB_TOKEN=${GITHUB_TOKEN}
      - GITHUB_WEBHOOK_SECRET=${GITHUB_WEBHOOK_SECRET}
    ports:
      - "8080:8080"
    depends_on:
      - postgres
    volumes:
      - ./config:/app/config
      - repos_data:/repos

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - litchi

volumes:
  postgres_data:
  repos_data:
```

---

## 11. 监控与可观测性

### 11.1 监控指标

| 指标类型 | 指标名称 | 说明 |
|---------|---------|------|
| Counter | `litchi_sessions_total` | 会话总数 |
| Counter | `litchi_sessions_completed_total` | 完成会话数 |
| Counter | `litchi_tasks_total` | 任务总数 |
| Counter | `litchi_tasks_failed_total` | 失败任务数 |
| Gauge | `litchi_sessions_active` | 活跃会话数 |
| Gauge | `litchi_tasks_in_progress` | 执行中任务数 |
| Histogram | `litchi_session_duration_seconds` | 会话耗时分布 |
| Histogram | `litchi_task_duration_seconds` | 任务耗时分布 |
| Histogram | `litchi_stage_duration_seconds` | 阶段耗时分布 |

### 11.2 健康检查

```
GET /health
```

```json
{
  "status": "healthy",
  "checks": {
    "database": "ok",
    "github_api": "ok",
    "git": "ok"
  },
  "version": "1.0.0"
}
```

---

## 12. 安全设计

### 12.1 用户角色与权限

#### 角色定义

| 角色 | 说明 | GitHub 权限要求 |
|------|------|----------------|
| **Issue 作者** | 创建 Issue 的用户 | 无特殊要求 |
| **仓库管理员** | 负责与 Agent 交互的授权用户 | `admin` 或 `maintain` 权限 |

#### 权限矩阵

| 操作 | Issue 作者 | 仓库管理员 |
|------|-----------|-----------|
| 创建 Issue | ✅ | ✅ |
| 触发 Agent（@bot） | ✅ | ✅ |
| 回答澄清问题 | ❌ | ✅ |
| 确认/拒绝设计方案 | ❌ | ✅ |
| 执行用户指令（继续/跳过/回退/终止） | ❌ | ✅ |
| 审批危险操作 | ❌ | ✅ |
| 合并 PR | ❌ | ✅ |
| 查看进度 | ✅ | ✅ |

#### 权限检查流程

```go
// internal/application/service/auth_service.go

type AuthService struct {
    githubClient *github.Client
}

// CheckRepoPermission 检查用户是否有仓库管理权限
func (s *AuthService) CheckRepoPermission(ctx context.Context, repo, username string) (bool, error) {
    permission, _, err := s.githubClient.Repositories.GetPermissionLevel(ctx, repo, username)
    if err != nil {
        return false, err
    }
    
    // admin 或 maintain 权限才视为仓库管理员
    return permission.GetPermission() == "admin" || 
           permission.GetPermission() == "maintain", nil
}

// ValidateActor 验证操作者是否有权限执行指令
func (s *AuthService) ValidateActor(ctx context.Context, repo, username, operation string) error {
    // 查询操作所需权限
    requiredRole := getRequiredRole(operation)
    
    if requiredRole == "repo_admin" {
        hasPermission, err := s.CheckRepoPermission(ctx, repo, username)
        if err != nil {
            return err
        }
        if !hasPermission {
            return errors.New("需要仓库管理员权限")
        }
    }
    
    return nil
}

// 操作权限映射
func getRequiredRole(operation string) string {
    viewerOperations := []string{"view_progress", "create_issue"}
    for _, op := range viewerOperations {
        if op == operation {
            return "viewer" // 任何人都可以
        }
    }
    return "repo_admin" // 默认需要仓库管理员权限
}
```

#### 触发后通知机制

当非管理员用户触发 Agent 时：

```go
func (s *IssueService) HandleTrigger(ctx context.Context, triggerUser, repo string) error {
    hasPermission, err := s.authService.CheckRepoPermission(ctx, repo, triggerUser)
    if err != nil {
        return err
    }
    
    if !hasPermission {
        // 通知仓库管理员接管
        s.notifyRepoAdmins(ctx, repo, fmt.Sprintf(
            "Issue 已由 %s 触发，请仓库管理员在 Comment 中回复指令接管流程。",
            triggerUser,
        ))
        
        // 进入等待管理员状态
        return s.setWaitingForAdmin(ctx, repo)
    }
    
    // 管理员触发，正常开始
    return s.startWorkSession(ctx, repo)
}
```

### 12.2 认证授权

| 组件 | 方案 |
|------|------|
| Webhook | GitHub 签名验证 |
| API | JWT Token（可选） |
| 前端 | Session Cookie（可选） |

### 12.3 敏感信息保护

- GitHub Token 存储在环境变量
- 数据库密码使用密钥管理服务
- 日志脱敏（不记录敏感信息）

### 12.4 危险操作审批

```go
type DangerousOperation string

const (
    OperationForcePush   DangerousOperation = "force-push"
    OperationDeleteBranch DangerousOperation = "delete-branch"
    OperationResetHard   DangerousOperation = "reset-hard"
)

func (s *SessionService) ExecuteDangerousOperation(
    ctx context.Context,
    sessionID uuid.UUID,
    operation DangerousOperation,
    reason string,
) error {
    // 1. 创建审批请求
    approval := s.createApproval(sessionID, operation, reason)

    // 2. 发布审批请求到 GitHub Comment
    s.notifyApprovalRequest(approval)

    // 3. 等待用户审批（超时 24h）
    <-approval.Done()

    if !approval.Approved() {
        return errors.New("operation not approved")
    }

    // 4. 执行操作
    return s.executeOperation(operation)
}
```

---

## 13. 扩展性设计

### 13.1 Agent 抽象层

```go
// internal/domain/service/agent_runner.go
// 统一的 AgentRunner 接口（领域服务）

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

// 实现示例
type ClaudeCodeAgent struct { ... }  // Claude Code CLI 实现
// type OpenAIAgent struct { ... }    // 未来扩展预留
```

### 13.2 事件驱动扩展

```go
// 领域事件订阅
type EventHandler interface {
    Handle(ctx context.Context, event DomainEvent) error
}

// 注册处理器
eventDispatcher.Register(WorkSessionStarted{}, &NotificationHandler{})
eventDispatcher.Register(TaskCompleted{}, &MetricsHandler{})
eventDispatcher.Register(StageTransitioned{}, &WebSocketPushHandler{})
```

---

## 14. 失败处理模块

### 14.1 模块架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Failure Handling Module                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      Error Detection Layer                          │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │   │
│  │  │ AgentError │  │ GitHubError │  │ NetworkError│  │ SystemError │ │   │
│  │  │ Detector   │  │ Detector    │  │ Detector    │  │ Detector    │ │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                      │                                     │
│                                      ▼                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      Error Classification                           │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                  │   │
│  │  │SeverityEval│  │RecoveryEval │  │TypeMapping  │                  │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘                  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                      │                                     │
│                                      ▼                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      Recovery Strategy Executor                     │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │   │
│  │  │ RetryHandler│  │ FallbackHdlr│  │ QueueHandler│  │ NotifyHandler│ │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 14.2 核心接口定义

```go
// internal/domain/service/error_handler.go

type ErrorHandler interface {
    // 检测并分类错误
    DetectAndClassify(err error) (*ClassifiedError, error)
    
    // 执行恢复策略
    ExecuteRecovery(ctx context.Context, classifiedErr *ClassifiedError) (*RecoveryResult, error)
    
    // 判断是否需要人工干预
    RequiresHumanIntervention(classifiedErr *ClassifiedError) bool
}

type ClassifiedError struct {
    OriginalError    error
    Type             ErrorType
    Severity         SeverityLevel
    RecoveryCategory RecoveryCategory
    Context          ErrorContext
    Timestamp        time.Time
}

type RecoveryResult struct {
    Success       bool
    ActionTaken   RecoveryAction
    NewState      SessionStatus
    NotifyMessage string
}
```

### 14.3 目录结构补充

```
internal/
├── domain/
│   └── service/
│       ├── error_handler.go          # 失败处理领域服务接口
│       └── retry_strategy.go         # 重试策略定义
│
├── infrastructure/
│   └── failure/
│       ├── detector/
│       │   ├── agent_error_detector.go
│       │   ├── github_error_detector.go
│       │   └── network_error_detector.go
│       ├── classifier/
│       │   └── error_classifier.go
│       └── recovery/
│           ├── retry_handler.go
│           ├── fallback_handler.go
│           └── queue_handler.go
```

### 14.4 错误码定义（部分）

| 错误码 | 类型 | 严重程度 | 说明 | 建议处理 |
|--------|------|---------|------|---------|
| L1SYS0001 | database | Critical | 数据库连接失败 | 立即告警，自动重连 |
| L2AGE0001 | agent | High | Agent 进程崩溃 | 尝试恢复会话 |
| L2AGE0002 | agent | High | 会话上下文丢失 | 通知重新触发 |
| L3GIT0001 | github | Medium | API 限流 | 等待重置 |
| L3NET0001 | network | Medium | 网络超时 | 重试 |
| L3ENV0001 | environment | Medium | 测试环境不可用 | 等待或跳过 |


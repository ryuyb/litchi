## Project Overview

**Litchi** - 自动化开发 Agent 系统，实现从 GitHub Issue 到 Pull Request 的全流程自动化。

当前状态：**设计阶段**，仅有文档，尚无代码实现。

## Architecture

### Backend (Go)
- **Web Framework**: Fiber v3
- **ORM**: GORM + PostgreSQL
- **DI**: Uber Fx (Provider/Invoke pattern)
- **Config**: Viper (YAML + env vars)
- **Logging**: Zap (structured JSON)
- **Migrations**: golang-migrate

### Frontend (React)
- **Framework**: TanStack Start (SSR/SSG)
- **Routing**: TanStack Router
- **Data Fetching**: TanStack Query (via Orval-generated hooks)
- **Tables/Forms**: TanStack Table + Form
- **Styling**: Tailwind CSS + shadcn/ui
- **State**: Zustand (client) + TanStack Query (server)
- **API Generation**: Orval (from Swagger/OpenAPI)

### Layered Architecture (DDD)
```
Presentation (React) → Application (Services) → Domain (Aggregates/Entities) → Infrastructure (GitHub/Git/Agent)
```

**Core Aggregate**: `WorkSession` (Issue + Clarification + Design + Task[] + Execution)

## Workflow Stages

```
Clarification → Design → TaskBreakdown → Execution → PullRequest → Completed
```

Each stage supports pause/resume/rollback.

## Key Documents

| Document | Purpose |
|----------|---------|
| `docs/index.md` | Documentation navigation |
| `docs/requirements.md` | Requirements specification |
| `docs/design/architecture.md` | System architecture |
| `docs/design/ddd.md` | Domain model design |
| `docs/design/state-machine.md` | State transition rules |
| `docs/tasks/index.md` | Task index (dependencies, progress) |
| `docs/tasks/phases/phase-*.md` | Detailed task specs per phase |

## Implementation Rules

实施开发时必须遵循以下规则：

1. **遵循任务规划**：所有实施必须按照 `docs/tasks/index.md` 中的任务顺序进行，尊重任务依赖关系
2. **及时更新进度**：完成任务后立即在对应的任务文档中勾选标记，保持进度同步
3. **确认需求清晰**：实施前必须确认需求已完全理解，如有任何不清晰之处，立即询问用户，不要假设或猜测
4. **询问优先**：宁可多问也不要做错，模糊的需求澄清比返工成本更低
5. **代码注释使用英文**：所有代码中的注释必须使用英文编写

## Compact Instructions

When compressing, preserve in priority order:

1. Architecture decisions (NEVER summarize)
2. Modified files and their key changes
3. Current verification status (pass/fail)
4. Open TODOs and rollback notes
5. Tool outputs (can delete, keep pass/fail only)

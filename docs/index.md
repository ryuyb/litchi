# Litchi 项目文档

> 自动化开发 Agent 系统 - 从 GitHub Issue 到 Pull Request 的全流程自动化

---

## 文档导航

### 快速开始

| 文档 | 说明 |
|------|------|
| [Setup Guide](./setup.md) | 安装指南：GitHub App 创建、配置文件、本地运行 |

### 需求与设计

| 文档 | 说明 |
|------|------|
| [需求文档](./requirements.md) | 项目需求规格说明 |
| [架构设计](./design/architecture.md) | 系统架构和技术选型 |
| [DDD 领域设计](./design/ddd.md) | 领域模型、聚合、实体设计 |
| [状态机设计](./design/state-machine.md) | WorkSession 和 Task 状态转换规则 |
| [Agent 调用层设计](./design/agent-runner.md) | Claude Code Agent 执行器设计 |
| [执行验证设计](./design/execution-validation.md) | 代码格式化、Lint、测试验证设计 |

### 任务清单

| 文档 | 说明 | 用途 |
|------|------|------|
| [任务索引](./tasks/index.md) | 精简版任务清单 | 查看依赖、进度、并行任务 |
| [完整任务清单](./tasks/full.md) | 详细版任务清单 | 完整参考（含验收标准） |

#### 各阶段任务

| 阶段 | 文档 | 工时 |
|------|------|------|
| 阶段一：项目初始化 | [phase-1-init.md](./tasks/phases/phase-1-init.md) | 9.5d |
| 阶段二：领域模型实现 | [phase-2-domain.md](./tasks/phases/phase-2-domain.md) | 10.5d |
| 阶段三：状态机实现 | [phase-3-state-machine.md](./tasks/phases/phase-3-state-machine.md) | 8d |
| 阶段四：外部集成层 | [phase-4-external.md](./tasks/phases/phase-4-external.md) | 13.5d |
| 阶段五：应用层实现 | [phase-5-app.md](./tasks/phases/phase-5-app.md) | 10d |
| 阶段六：HTTP API 实现 | [phase-6-api.md](./tasks/phases/phase-6-api.md) | 6.5d |
| 阶段七：前端实现 | [phase-7-frontend.md](./tasks/phases/phase-7-frontend.md) | 13d |
| 阶段八：集成测试与部署 | [phase-8-deploy.md](./tasks/phases/phase-8-deploy.md) | 11d |

---

## 文档结构

```
docs/
├── index.md                    # 文档导航入口
├── requirements.md             # 需求文档
├── design/                     # 设计文档
│   ├── architecture.md         # 架构设计
│   ├── ddd.md                  # DDD 领域设计
│   ├── state-machine.md        # 状态机设计
│   ├── agent-runner.md         # Agent 调用层设计
│   └── execution-validation.md # 执行验证设计
└── tasks/                      # 任务文档
    ├── index.md                # 任务索引（精简）
    ├── full.md                 # 完整任务清单
    └── phases/                 # 各阶段详细任务
        ├── phase-1-init.md
        ├── phase-2-domain.md
        ├── phase-3-state-machine.md
        ├── phase-4-external.md
        ├── phase-5-app.md
        ├── phase-6-api.md
        ├── phase-7-frontend.md
        └── phase-8-deploy.md
```

---

## 技术栈概览

### 后端
- **Web 框架**: [Fiber v3](https://github.com/gofiber/fiber) - Express 风格的高性能 Go Web 框架
- **ORM**: [GORM](https://gorm.io/) - Go 语言 ORM 库
- **依赖注入**: [Uber Fx](https://github.com/uber-go/fx) - 基于函数签名的依赖注入框架
- **配置管理**: [Viper](https://github.com/spf13/viper) - 配置管理
- **日志**: [Zap](https://github.com/uber-go/zap) - 结构化日志
- **数据库**: PostgreSQL + golang-migrate（迁移工具）

### 前端
- **框架**: [TanStack Start](https://tanstack.com/start) - 全栈 React 框架
- **路由**: [TanStack Router](https://tanstack.com/router) - 类型安全路由
- **数据获取**: [TanStack Query](https://tanstack.com/query) - 服务端状态管理
- **表格**: [TanStack Table](https://tanstack.com/table) - 表格组件
- **表单**: [TanStack Form](https://tanstack.com/form) - 表单管理
- **API 代码生成**: [Orval](https://orval.dev/) - 根据 Swagger/OpenAPI 生成 TypeScript API 客户端
- **样式**: [Tailwind CSS](https://tailwindcss.com/) + [shadcn/ui](https://ui.shadcn.com/)
- **客户端状态**: [Zustand](https://zustand-demo.pmnd.rs/) - 轻量状态管理
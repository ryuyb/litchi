# 阶段一：项目初始化 (Project Initialization)

> 技术栈：Fiber v3 + GORM + Fx + TanStack Start + Orval

---

## 1.1 Go 后端项目初始化

- [x] **T1.1.1** 初始化 Go 项目结构（Fiber v3 + Fx）
  - 验收标准：
    - [x] `go build ./...` 成功
    - [x] 目录结构包含 `cmd/`, `internal/`, `pkg/`
    - [x] go.mod 文件配置正确，依赖 Fiber v3 + Uber Fx
    - [x] Fx App 主框架可启动
    - [x] Fiber v3 通过 Fx Lifecycle 正确注册
  - 依赖：无
  - 风险：低
  - 预估：0.5d
  - 可并行：是

- [x] **T1.1.2** 配置 Viper 配置管理（Fx Provider）
  - 验收标准：
    - [x] 可读取 YAML 配置文件
    - [x] 支持环境变量覆盖
    - [x] 配置结构体定义完整
    - [x] 作为 Fx Provider 注册到依赖容器
  - 依赖：T1.1.1
  - 风险：低
  - 预估：0.5d
  - 可并行：否

- [x] **T1.1.3** 配置 Zap 结构化日志（Fx Provider）
  - 验收标准：
    - [x] 日志可按级别输出 JSON 格式
    - [x] 支持日志级别动态调整
    - [x] 作为 Fx Provider 注册到依赖容器
    - [x] 通过 Fx Invoke 初始化全局 logger
  - 依赖：T1.1.1
  - 风险：低
  - 预估：0.5d
  - 可并行：是（与 T1.1.2）

- [x] **T1.1.4** 定义全局错误码体系
  - 验收标准：
    - [x] 错误码定义完整（领域错误、基础设施错误、API 错误）
    - [x] 错误包装和解析方法正确
    - [x] 错误码文档生成
  - 依赖：T1.1.1
  - 风险：低
  - 预估：0.5d
  - 可并行：是（与 T1.1.2、T1.1.3）

- [x] **T1.1.5** 设计 Fx 模块化架构
  - 验收标准：
    - [x] 模块划分清晰
    - [x] 每个模块提供 Fx Module（包含 Provider/Invoke）
    - [x] 主程序通过 Fx Options 组合模块
    - [x] 依赖图可视化验证正确
  - 依赖：T1.1.1, T1.1.2, T1.1.3, T1.1.4
  - 风险：中
  - 预估：0.5d
  - 可并行：否

---

## 1.2 数据库层

- [x] **T1.2.1** 设计并创建 PostgreSQL 数据库表结构
  - 验收标准：
    - [x] 所有表创建成功
    - [x] 外键约束正确
    - [x] 索引覆盖常用查询
  - 依赖：T1.1.1
  - 风险：中
  - 预估：1d
  - 可并行：否

- [x] **T1.2.2** 集成数据库迁移工具
  - 验收标准：
    - [x] golang-migrate 集成成功
    - [x] 迁移脚本可执行
    - [x] 支持回滚操作
  - 依赖：T1.2.1
  - 风险：低
  - 预估：0.5d
  - 可并行：否

- [x] **T1.2.3** 实现 GORM 模型定义
  - 验收标准：
    - [x] 模型与表结构对应
    - [x] 关联关系正确配置
    - [x] 支持软删除
  - 依赖：T1.2.1
  - 风险：低
  - 预估：1d
  - 可并行：是（与 T1.2.2）

- [x] **T1.2.4** 实现数据库连接池和事务管理（Fx Provider）
  - 验收标准：
    - [x] 连接池配置正确
    - [x] 事务传播行为正确
    - [x] 连接泄漏检测
    - [x] GORM DB 作为 Fx Provider 注册
    - [x] 通过 Fx Lifecycle 管理连接（OnStart/OnStop）
  - 依赖：T1.2.1, T1.2.3
  - 风险：中
  - 预估：1d
  - 可并行：否

---

## 1.3 前端初始化

- [x] **T1.3.1** 初始化 TanStack Start 项目
  - 验收标准：
    - [x] 项目可启动（`pnpm dev` 或 `npm run dev`）
    - [x] TanStack Router 路由配置正确
    - [x] TypeScript 配置正确
    - [x] SSR/SSG 基础功能可用
  - 注：通过 add-on 同时安装了 Query、Table、Form、Shadcn、T3Env
  - 依赖：无
  - 风险：低
  - 预估：0.5d
  - 可并行：是（与 1.1.*）
  - 实际工时：0.5d
  - 完成日期：2026-04-04

- [x] **T1.3.2** 配置 shadcn/ui + Tailwind CSS
  - 验收标准：
    - [x] UI 组件库安装成功
    - [x] Tailwind 配置正确，与 TanStack Start 集成
    - [x] 暗色主题支持
  - 注：通过 add-on 安装，已有 components.json、radix-ui、Tailwind CSS v4
  - 依赖：T1.3.1
  - 风险：低
  - 预估：0.5d
  - 可并行：否
  - 实际工时：0d（add-on 包含）
  - 完成日期：2026-04-04

- [ ] **T1.3.3** 实现基础布局组件
  - 验收标准：
    - [ ] Sidebar 组件正确显示
    - [ ] Header 组件正确显示
    - [ ] MainLayout 响应式布局
    - [ ] TanStack Router 嵌套布局正确
  - 依赖：T1.3.2
  - 风险：低
  - 预估：1d
  - 可并行：否

- [ ] **T1.3.4** 配置 TanStack Query + TanStack Store
  - 验收标准：
    - [x] TanStack Query 配置正确（服务端状态）
    - [ ] TanStack Store 配置正确（客户端状态）
    - [ ] QueryClient 正确挂载到应用
    - [ ] Store 持久化配置
  - 注：TanStack Query 已通过 add-on 安装，TanStack Store 已通过 add-on 安装，需配置持久化
  - 依赖：T1.3.1
  - 风险：低
  - 预估：0.5d
  - 可并行：是（与 T1.3.2）
  - 进度：70%（Query 和 Store 已安装，待配置持久化）

- [ ] **T1.3.5** 配置 Orval API 代码生成
  - 验收标准：
    - [ ] Orval 配置文件正确（orval.config.ts）
    - [ ] 配置 Swagger/OpenAPI 文档源路径（指向后端生成的 swagger.json）
    - [ ] 配置生成 TanStack Query hooks
    - [ ] 配置生成 TypeScript 类型定义
    - [ ] 配置输出目录和代码格式
    - [ ] 配置 npm script：`orval:generate`
    - [ ] 注：实际生成在阶段六 T6.0.1 完成后执行
  - 依赖：T1.3.1
  - 风险：低
  - 预估：0.5d
  - 可并行：是（与 T1.3.2）

- [ ] **T1.3.6** 配置 TanStack Table + Form
  - 验收标准：
    - [x] TanStack Table 基础配置
    - [x] TanStack Form 基础配置
    - [ ] 与 TanStack Query 数据流集成
    - [ ] 类型安全验证（基础类型定义）
  - 注：Table 和 Form 已通过 add-on 安装，需后续集成
  - 依赖：T1.3.4
  - 风险：低
  - 预估：0.5d
  - 可并行：否
  - 进度：50%（安装完成，待集成）

---

## 阶段工时

**总计**: 9.5d

---

## 并行任务说明

- **T1.1.1, T1.3.1**: Go 后端（Fiber v3 + Fx）与前端（TanStack Start）初始化可并行
- **T1.1.2, T1.1.3, T1.1.4**: 配置管理、日志、错误码可并行（需 T1.1.1 完成）
- **T1.2.2, T1.2.3**: 迁移工具与 GORM 模型可并行（需 T1.2.1 完成）
- **T1.3.2, T1.3.4, T1.3.5**: 前端 UI 库、TanStack Query/Zustand、Orval 配置可并行（需 T1.3.1 完成）
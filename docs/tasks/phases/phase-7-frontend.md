# 阶段七：前端实现 (Frontend - TanStack Start)

> 技术栈：TanStack Start + Query + Table + Form + Orval

---

## 7.0 API 代码生成

- [x] **T7.0.1** 执行 Orval 生成 API 客户端代码
  - 验收标准：
    - [x] 执行 `pnpm orval:generate` 成功
    - [x] 生成的 TypeScript 类型定义正确
    - [x] 生成的 TanStack Query hooks 正确（useQuery/useMutation）
    - [x] 生成的 API 客户端函数类型安全
    - [x] 代码格式正确，无 TypeScript 错误
  - 依赖：T6.0.1, T1.3.5
  - 风险：低
  - 预估：0.5d
  - 可并行：否

---

## 7.1 核心页面（TanStack Router）

- [x] **T7.1.1** 实现仪表盘页面
  - 验收标准：
    - [x] TanStack Router 路由配置正确
    - [x] TanStack Query 数据获取正确（使用 Orval 生成的 hooks）
    - [x] 统计数据显示正确
    - [x] 活跃会话列表正确
    - [ ] 实时状态更新（WebSocket + TanStack Query）
  - 依赖：T6.1.1, T6.2.2, T1.3.3, T7.0.1
  - 风险：低
  - 预估：1.5d
  - 可并行：是（与 T7.1.2~T7.1.5）

- [x] **T7.1.2** 实现 Issue 列表页面
  - 验收标准：
    - [x] TanStack Table 配置正确
    - [x] 分页功能正确
    - [x] 筛选功能正确
    - [x] 状态显示正确
  - 依赖：T6.1.1, T7.0.1
  - 风险：低
  - 预估：1d
  - 可并行：是（与 T7.1.*）

- [x] **T7.1.3** 实现 Issue 详情页面
  - 验收标准：
    - [x] TanStack Router 动态路由正确
    - [x] 阶段进度显示正确
    - [x] TanStack Table 任务列表显示正确
    - [x] 操作按钮功能正确
  - 依赖：T6.1.1, T6.1.2, T7.0.1
  - 风险：中
  - 预估：2d
  - 可并行：是（与 T7.1.*）

- [x] **T7.1.4** 实现仓库列表页面
  - 验收标准：
    - [x] TanStack Table 仓库列表显示正确
    - [x] 启用/禁用操作正确
    - [x] 搜索功能正确
  - 依赖：T6.1.4, T7.0.1
  - 风险：低
  - 预估：1d
  - 可并行：是（与 T7.1.*）

- [x] **T7.1.5** 实现仓库配置页面
  - 验收标准：
    - [x] TanStack Form 配置表单正确（使用 React 状态管理）
    - [x] TanStack Query mutation 正确
    - [x] 验证配置编辑正确
    - [x] 保存功能正确
  - 依赖：T6.1.4, T7.0.1
  - 风险：中
  - 预估：1.5d
  - 可并行：是（与 T7.1.*）

---

## 7.2 业务组件

- [x] **T7.2.1** 实现阶段进度组件
  - 验收标准：
    - [x] 5 阶段可视化显示 (6 stages: Clarification -> Design -> TaskBreakdown -> Execution -> PullRequest -> Completed)
    - [x] 当前阶段高亮
    - [x] 支持点击查看详情
    - [x] TanStack Router 链接导航 (via onStageClick callback)
  - 依赖：T7.1.3
  - 风险：低
  - 预估：1d
  - 可并行：是（与 T7.2.2~T7.2.5）

- [x] **T7.2.2** 实现任务列表组件
  - 验收标准：
    - [x] TanStack Table 配置正确
    - [x] 任务状态显示正确
    - [x] 依赖关系显示正确
    - [x] TanStack Query mutation 支持任务操作
  - 依赖：T7.1.3, T7.0.1
  - 风险：低
  - 预估：1d
  - 可并行：是（与 T7.2.*）

- [x] **T7.2.3** 实现日志查看组件
  - 验收标准：
    - [x] WebSocket 实时日志显示
    - [x] TanStack Query 缓存正确
    - [x] 日志过滤功能
    - [x] 日志搜索功能
  - 依赖：T6.2.2
  - 风险：中
  - 预估：1.5d
  - 可并行：是（与 T7.2.*）

- [x] **T7.2.4** 实现验证配置表单组件
  - 验收标准：
    - [x] 表单状态管理正确（使用 React useState）
    - [x] 格式化配置表单正确
    - [x] Lint 配置表单正确
    - [x] 测试配置表单正确
  - 依赖：T7.1.5, T7.0.1
  - 风险：低
  - 预估：1d
  - 可并行：是（与 T7.2.*）

- [x] **T7.2.5** 实现项目检测结果展示
  - 验收标准：
    - [x] TanStack Query 数据获取正确
    - [x] 检测信息显示正确
    - [x] 置信度显示正确
    - [x] 支持重新触发检测
  - 依赖：T7.1.5
  - 风险：低
  - 预估：1d
  - 可并行：是（与 T7.2.*）

---

## 7.3 审计日志

- [x] **T7.3.1** 实现审计日志列表页面
  - 验收标准：
    - [x] TanStack Table 配置正确
    - [x] 多条件筛选正确
    - [x] TanStack Query 分页功能正确
    - [x] 导出功能支持
  - 依赖：T6.1.6, T7.0.1
  - 风险：低
  - 预估：1d
  - 可并行：是（与 T7.3.2）

- [x] **T7.3.2** 实现审计日志详情页面
  - 验收标准：
    - [x] TanStack Router 动态路由正确
    - [x] TanStack Query 数据获取正确
    - [x] 完整信息展示
    - [x] TanStack Router 链接到关联实体
    - [x] 操作上下文显示
  - 依赖：T6.1.6
  - 风险：低
  - 预估：0.5d
  - 可并行：是（与 T7.3.1）

---

## 阶段工时

**总计**: 13d

---

## 并行任务说明

- **T7.1.1 ~ T7.1.5**: 所有核心页面可并行（依赖 T7.0.1 Orval 生成完成后）
- **T7.2.1 ~ T7.2.5**: 所有业务组件可并行
- **T7.3.1, T7.3.2**: 审计日志页面可并行
# 阶段六：HTTP API 实现 (HTTP API - Fiber v3 + Fx)

> 技术栈：Fiber v3 + Fx + Swagger + WebSocket

---

## 6.0 OpenAPI/Swagger 文档

- [x] **T6.0.1** 配置 Swagger/OpenAPI 文档生成
  - 验收标准：
    - [x] Swagger/OpenAPI 3.0 规范文档生成正确 (OpenAPI 3.1)
    - [x] 使用 fiber-swagger 或 swaggo/swag 集成 (swaggo/swag/v2 + swaggerui)
    - [x] API 注释格式正确（路由、参数、响应）
    - [x] Swagger UI 可访问（/swagger/*）
    - [x] 文档自动更新（随 API 变化）
    - [x] 导出 swagger.json 供 Orval 使用 (docs/api/swagger.json)
    - [x] 替换 T1.3.5 手写的临时 swagger.json (等 T6.1 完成后切换)
    - [x] 更新前端 Orval 配置指向新文档路径 (暂保留 web/swagger.json 作为设计文档)
    - [x] 执行 `orval:generate` 重新生成 API 客户端代码
  - 依赖：T5.1.*
  - 风险：低
  - 预估：0.5d
  - 可并行：否
  - 备注：
    - 完成后需通知前端重新执行 Orval 生成
    - **Orval 切换时机**: T6.1.1~T6.1.7 完成后，删除 web/swagger.json，修改 orval.config.ts 指向 docs/api/swagger.json

---

## 6.1 REST API（Fiber v3 + Fx）

- [x] **T6.1.1** 实现会话管理 API（Fx Provider）
  - 验收标准：
    - [x] Fiber v3 路由组配置正确
    - [x] CRUD 操作正确
    - [x] 暂停/恢复/回退/终止操作正确
    - [x] Handler 作为 Fx Provider 注册
    - [x] 路由通过 Fx Invoke 注册到 Fiber App
    - [x] Swagger 注释完整，文档正确生成
  - 依赖：T6.0.1
  - 风险：低
  - 预估：1d
  - 可并行：是（与 T6.1.2~T6.1.7）

- [x] **T6.1.2** 实现任务管理 API（Fx Provider）
  - 验收标准：
    - [x] Fiber v3 路由配置正确
    - [x] 列表查询正确
    - [x] 跳过/重试操作正确
    - [x] 分页支持
    - [x] Handler 作为 Fx Provider 注册
    - [x] Swagger 注释完整
  - 依赖：T6.0.1, T5.1.4
  - 风险：低
  - 预估：0.5d
  - 可并行：是（与 T6.1.*）

- [x] **T6.1.3** 实现配置管理 API（Fx Provider）
  - 验收标准：
    - [x] 获取配置正确
    - [x] 更新配置正确
    - [x] 配置验证正确
    - [x] Handler 作为 Fx Provider 注册
    - [x] Swagger 注释完整
  - 依赖：T6.0.1
  - 风险：低
  - 预估：0.5d
  - 可并行：是（与 T6.1.*）

- [x] **T6.1.4** 实现仓库管理 API（Fx Provider）
  - 验收标准：
    - [x] 仓库 CRUD 正确
    - [x] 验证配置编辑正确
    - [x] 启用/禁用操作正确
    - [x] Handler 作为 Fx Provider 注册
    - [x] Swagger 注释完整
  - 依赖：T6.0.1
  - 风险：低
  - 预估：0.5d
  - 可并行：是（与 T6.1.*）

- [x] **T6.1.5** 实现 Webhook 接收 API（Fx Provider）
  - 验收标准：
    - [x] Fiber v3 Webhook 路由配置
    - [x] GitHub Webhook 正确处理
    - [x] 签名验证正确（使用 Fiber 中间件）
    - [x] 异步处理支持
    - [x] Handler 作为 Fx Provider 注册
    - [x] Swagger 注释完整
  - 依赖：T6.0.1, T4.1.5
  - 风险：中
  - 预估：1d
  - 可并行：是（与 T6.1.*）

- [x] **T6.1.6** 实现审计日志 API（Fx Provider）
  - 验收标准：
    - [x] 查询方法正确
    - [x] 多条件过滤支持
    - [x] 分页支持
    - [x] Handler 作为 Fx Provider 注册
    - [x] Swagger 注释完整
  - 依赖：T6.0.1
  - 风险：低
  - 预估：0.5d
  - 可并行：是（与 T6.1.*）

- [x] **T6.1.7** 实现健康检查 API（Fx Provider）
  - 验收标准：
    - [x] 数据库检查正确
    - [x] GitHub 连接检查
    - [x] Git 可用性检查
    - [x] Fiber v3 健康检查中间件配置
    - [x] Handler 作为 Fx Provider 注册
    - [x] Swagger 注释完整
  - 依赖：T6.0.1
  - 风险：低
  - 预估：0.5d
  - 可并行：是（与 T6.1.*）

---

## 6.2 WebSocket（Fiber v3 WebSocket + Fx）

- [x] **T6.2.1** 实现 WebSocket 连接管理（Fx Provider）
  - 验收标准：
    - [x] Fiber v3 WebSocket 配置正确
    - [x] 连接建立正确
    - [x] 断开处理正确
    - [x] 心跳机制正确
    - [x] WebSocket Handler 作为 Fx Provider 注册
  - 依赖：T6.1.*
  - 风险：中
  - 预估：1d
  - 可并行：否

- [x] **T6.2.2** 实现实时进度推送（Fx Invoke）
  - 验收标准：
    - [x] 阶段转换推送正确
    - [x] 任务状态推送正确
    - [x] 日志流推送正确
    - [x] 通过 Fx Invoke 注册事件订阅
  - 依赖：T6.2.1, T2.6.2
  - 风险：中
  - 预估：1d
  - 可并行：否

---

## 阶段工时

**总计**: 6.5d

---

## 并行任务说明

- **T6.1.1 ~ T6.1.7**: 所有 REST API 可并行（依赖 T6.0.1 Swagger 配置完成后）
- WebSocket 任务需串行：T6.2.1 → T6.2.2
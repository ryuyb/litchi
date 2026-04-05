# 阶段三：状态机实现 (State Machine)

> 技术栈：Go

---

## 3.1 WorkSession 状态机

- [x] **T3.1.1** 实现阶段正向转换逻辑 ⚠️ **高风险**
  - 验收标准：
    - [x] 5 个正向转换规则正确
    - [x] 前置条件验证完整
    - [x] 状态变更事件发布
  - 依赖：T2.4.2
  - 风险：**高**
  - 预估：1.5d
  - 可并行：否
  - 备注：正向转换是核心流程，出错影响大
  - 实施详情：
    - 扩展 TransitionContext 添加 AutoProceedThreshold(80)、ForceClarifyThreshold(40)、SkipClarityCheck
    - 新建 transition_decision.go 定义 TransitionDecision 枚举和 TransitionResult 结构体
    - 实现 EvaluateTransition 方法处理清晰度评分分级规则（自动转换/需确认/拒绝转换）
    - 实现 5 个阶段的评估方法（evaluateClarificationToDesign 等）
    - 添加 13 个测试场景覆盖各种边界情况

- [x] **T3.1.2** 实现阶段回退转换逻辑 ⚠️ **高风险**
  - 验收标准：
    - [x] 6 个回退规则 (R1-R6) 正确
    - [x] 回退条件验证完整
    - [x] 状态恢复正确
  - 依赖：T2.4.2
  - 风险：**高**
  - 预估：1.5d
  - 可并行：是（与 T3.1.1）
  - 备注：回退规则需要处理各种边界情况
  - 实施详情：
    - 新建 rollback_decision.go 定义 RollbackDecision、RollbackType、RollbackResult
    - 扩展 StageTransitionService 接口添加 EvaluateRollback、GetRollbackRule、ValidateRollbackConditions 方法
    - 实现 R1-R6 回退规则评估逻辑，包含回退效果标志（WillDeprecateBranch、WillClosePR 等）
    - 添加回退相关领域事件（ExecutionRolledBackToDesign、DesignRolledBackToClarification 等）
    - 编写 rollback_test.go 覆盖所有回退规则场景

- [x] **T3.1.3** 实现暂停/恢复/终止逻辑
  - 验收标准：
    - [x] 所有暂停原因处理正确
    - [x] 恢复逻辑正确
    - [x] 终止清理正确
  - 依赖：T2.4.2
  - 风险：中
  - 预估：1d
  - 可并行：否
  - 实施详情：
    - 新建 pause_reason.go 定义 14 种暂停原因枚举和 PauseContext、PauseRecord 结构体
    - 实现恢复分类（Auto/SemiAuto/Manual）和恢复操作列表
    - 增强 WorkSession 聚合根添加 PauseContext/PauseHistory 字段
    - 实现 PauseWithContext、ResumeWithAction、CanAutoResume 等方法
    - 新建 session_control.go 实现 SessionControlService
    - 添加暂停恢复领域事件（WorkSessionPausedWithContext、WorkSessionResumedWithAction 等）
    - 编写 pause_reason_test.go 和 session_control_test.go 测试

---

## 3.2 Task 状态机

- [x] **T3.2.1** 实现 Task 状态转换
  - 验收标准：
    - [x] Pending→InProgress→Completed/Failed/Skipped 转换正确
    - [x] 非法转换拦截
    - [x] 状态变更事件发布
  - 依赖：T2.2.4
  - 风险：中
  - 预估：1d
  - 可并行：否
  - 实施详情：
    - 创建 TaskTransitionService 处理状态转换和事件发布
    - 实现 StartTask/CompleteTask/FailTask/SkipTask/RetryTask 方法
    - 添加 EventDispatcher 接口定义（同步/异步分发）
    - 实现 TaskTransitionResult 结构体包含转换结果和事件
    - 编写 task_transition_test.go 覆盖所有转换场景和事件发布验证

- [x] **T3.2.2** 实现 Task 重试逻辑
  - 验收标准：
    - [x] 重试次数限制正确（默认 3 次）
    - [x] 退避策略正确
    - [x] 最终失败处理正确
  - 依赖：T3.2.1
  - 风险：中
  - 预估：1d
  - 可并行：否
  - 实施详情：
    - 创建 retry_policy.go 定义 BackoffStrategy/BackoffConfig/RetryPolicy
    - 实现三种退避策略：Exponential（默认）、Linear、Constant
    - 定义 RetryContext 和 RetryRecord 跟踪重试历史和延迟计算
    - 实现 FinalFailureHandling 定义四种最终失败处理：PauseSession/SkipTask/Rollback/Terminate
    - 添加 HandleFinalFailure 方法处理重试耗尽后的决策
    - 编写 retry_policy_test.go 覆盖退避计算和重试策略验证

---

## 3.3 状态持久化

- [x] **T3.3.1** 实现数据库状态持久化
  - 验收标准：
    - [x] 状态变更正确写入数据库
    - [x] 事务保证一致性
    - [x] 支持乐观锁
  - 依赖：T2.5.1
  - 风险：中
  - 预估：1d
  - 可并行：否
  - 实施详情：
    - 修改 migrations/000001_init_schema.up.sql 添加 version、pause_context、pause_history 字段
    - 修改 models.go 添加 Version、PauseContext、PauseHistory 字段
    - 修改 converter.go 添加 PauseContext/PauseHistory JSON 序列化
    - 修改 work_session_repo.go 实现乐观锁更新（WHERE version = ? 检查 + version+1）
    - 添加 ErrVersionConflict 错误码，返回 409 Conflict

- [x] **T3.3.2** 实现文件缓存持久化
  - 验收标准：
    - [x] 正确读写执行上下文到文件
    - [x] 文件格式正确（JSON）
    - [x] 缓存失效处理（文件不存在返回 nil）
  - 依赖：T2.5.1
  - 风险：低
  - 预估：0.5d
  - 可并行：是（与 T3.3.1）
  - 实施详情：
    - 创建 internal/domain/repository/cache.go 定义 CacheRepository 接口和缓存数据结构
    - 创建 internal/infrastructure/cache/file_cache.go 实现 FileCacheRepository
    - 实现 Save/Load/Delete 方法，使用 JSON 格式存储到 {worktreePath}/.litchi/context.json
    - 创建 internal/infrastructure/cache/module.go 定义 Fx 模块
    - 编写 file_cache_test.go 覆盖所有场景（读写、不存在、删除、完整缓存结构）

- [x] **T3.3.3** 实现状态一致性检查和修复
  - 验收标准：
    - [x] 可检测不一致状态
    - [x] 可自动修复常见问题
    - [x] 检查报告生成
  - 依赖：T3.3.1, T3.3.2
  - 风险：**高**
  - 预估：1.5d
  - 可并行：否
  - 实施详情：
    - 创建 internal/domain/service/consistency_checker.go 定义 ConsistencyChecker 接口、IssueType、Severity、ConsistencyReport
    - 创建 internal/application/service/consistency_service.go 实现 Check、CheckAndRepair、Repair 方法
    - 实现 5 类检查规则：CacheMismatch、StatusMismatch、TaskProgress、PauseContextStale、DesignMissing
    - 实现自动修复逻辑：regenerate_cache、set_status_completed、clear_pause_context、clear_current_task_id
    - 编写 consistency_service_test.go 覆盖所有检查和修复场景（10 个测试用例）

---

## 阶段工时

**总计**: 8d

---

## 并行任务说明

- **T3.1.1, T3.1.2**: 正向转换和回退转换可并行
- **T3.3.1, T3.3.2**: 数据库持久化和文件缓存可并行
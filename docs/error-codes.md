# Litchi 错误码体系

## 错误码格式

错误码格式：`L{Severity}{Category}{Number}`

- **Severity**: 严重程度
  - `1` = Critical（系统级致命错误）
  - `2` = High（Agent 相关严重错误）
  - `3` = Medium（外部服务/环境错误）
  - `4` = Low（业务逻辑错误）

- **Category**: 错误类别
  - `SYS` = System（系统错误）
  - `AGE` = Agent（Agent 执行错误）
  - `GIT` = GitHub（GitHub API 错误）
  - `NET` = Network（网络错误）
  - `ENV` = Environment（环境错误）
  - `DOM` = Domain（领域错误）
  - `API` = API（API 错误）

- **Number**: 4 位序号

## 错误码列表

### Critical (Severity 1) - 系统级致命错误

| 错误码 | 消息 | 类别 | 说明 | 建议 |
|--------|------|------|------|------|
| L1SYS0001 | Database connection failed | SYS | 数据库连接失败 | 检查数据库配置、网络连接 |
| L1SYS0002 | Configuration load failed | SYS | 配置加载失败 | 检查配置文件格式、路径 |
| L1SYS0003 | Server startup failed | SYS | 服务器启动失败 | 检查端口占用、权限 |
| L1SYS0004 | Database migration failed | SYS | 数据库迁移失败 | 检查迁移脚本、数据库状态 |

### High (Severity 2) - Agent 相关错误

| 错误码 | 消息 | 类别 | 说明 | 建议 |
|--------|------|------|------|------|
| L2AGE0001 | Agent process crashed | AGE | Agent 进程崩溃 | 尝试恢复会话 |
| L2AGE0002 | Agent session context lost | AGE | Agent 会话上下文丢失 | 通知重新触发 |
| L2AGE0003 | Agent execution failed | AGE | Agent 执行失败 | 查看详细日志 |
| L2AGE0004 | Agent execution timeout | AGE | Agent 执行超时 | 检查任务复杂度 |
| L2AGE0005 | Agent permission denied | AGE | Agent 权限被拒绝 | 检查权限配置 |

### Medium (Severity 3) - 外部服务错误

| 错误码 | 消息 | 类别 | 说明 | 建议 |
|--------|------|------|------|------|
| L3GIT0001 | GitHub API rate limit exceeded | GIT | GitHub API 限流 | 等待重置或使用认证 |
| L3GIT0002 | GitHub authentication failed | GIT | GitHub 认证失败 | 检查 Token 配置 |
| L3GIT0003 | GitHub API error | GIT | GitHub API 错误 | 查看响应详情 |
| L3GIT0004 | Webhook signature verification failed | GIT | Webhook 签名验证失败 | 检查 Secret 配置 |
| L3NET0001 | Network timeout | NET | 网络超时 | 重试操作 |
| L3NET0002 | Network connection failed | NET | 网络连接失败 | 检查网络状态 |
| L3ENV0001 | Test environment unavailable | ENV | 测试环境不可用 | 等待或跳过测试 |
| L3ENV0002 | Git operation failed | ENV | Git 操作失败 | 查看错误详情 |

### Low (Severity 4) - 业务逻辑错误

| 错误码 | 消息 | 类别 | 说明 | 建议 |
|--------|------|------|------|------|
| L4TASK0001 | Task was skipped | DOM | 任务被跳过 | 查看跳过原因 |
| L4TASK0002 | Task already completed | DOM | 任务已完成 | 无需处理 |
| L4ENV0001 | No tests found | ENV | 未找到测试 | 可跳过测试 |
| L4DOM0001 | Work session not found | DOM | 工作会话不存在 | 检查 ID |
| L4DOM0002 | Issue not found | DOM | Issue 不存在 | 检查 Issue 编号 |
| L4DOM0003 | Invalid stage transition | DOM | 无效的阶段转换 | 查看当前状态 |
| L4API0001 | Permission denied | API | 权限被拒绝 | 检查用户权限 |
| L4API0002 | Validation failed | API | 验证失败 | 检查输入参数 |
| L4API0003 | Bad request | API | 错误的请求 | 检查请求格式 |

## API 错误码映射

| HTTP 状态码 | 消息 | 适用场景 |
|-------------|------|----------|
| 400 | Bad request | 参数验证失败 |
| 401 | Unauthorized | 认证失败 |
| 403 | Forbidden | 权限不足 |
| 404 | Not found | 资源不存在 |
| 409 | Conflict | 资源冲突 |
| 500 | Internal server error | 服务器内部错误 |
| 503 | Service unavailable | 服务不可用 |

## 使用示例

```go
// 创建新错误
err := errors.New(errors.ErrSessionNotFound).
    WithDetail("session ID: abc-123").
    WithContext("repository", "org/repo")

// 包装现有错误
err := errors.Wrap(errors.ErrDatabaseConnection, originalErr)

// 判断错误类型
if errors.Is(err, errors.ErrSessionNotFound) {
    // 处理会话不存在
}

// 获取错误码
code := errors.GetCode(err) // "L4DOM0001"

// 获取严重程度
severity := errors.GetSeverity(err) // 4
```
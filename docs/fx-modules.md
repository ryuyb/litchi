# Fx 模块化架构

## 模块设计原则

1. **每个模块独立封装**：通过 `fx.Module()` 定义，包含 Provider 和 Invoke
2. **依赖明确**：通过 fx.In 参数结构体明确依赖
3. **生命周期管理**：通过 fx.Lifecycle 管理资源启动和停止
4. **可测试性**：模块可独立测试，依赖可 mock

## 模块注册

每个模块通过 `init()` 函数自动注册到 `fxutil.Registry`：

```go
func init() {
    fxutil.RegisterModule(fxutil.ModuleInfo{
        Name:     "module_name",
        Provides: []string{"*Type"},
        Invokes:  []string{"HookFunction"},
        Depends:  []string{"*DependencyType"},
    })
}
```

## 当前模块列表

### logger 模块

| 属性 | 值 |
|------|-----|
| 名称 | `logger` |
| 提供 | `*zap.Logger`, `*zap.SugaredLogger` |
| 调用 | `RegisterLifecycle` |
| 依赖 | `*config.Config` |
| 文件 | `internal/pkg/logger/fx.go` |

功能：
- JSON/Console 格式日志
- 级别配置 (debug/info/warn/error)
- 输出路径配置
- Fx 生命周期同步

### server 模块

| 属性 | 值 |
|------|-----|
| 名称 | `server` |
| 提供 | `*fiber.App` |
| 调用 | `StartAppHook` |
| 依赖 | `*zap.Logger` |
| 文件 | `internal/application/server/module.go` |

功能：
- Fiber v3 HTTP 服务器
- 健康检查端点 (`/health`)
- 生命周期启动/停止

## 依赖图

```
main.go (CLI)
    │
    ├── config.NewConfigWithOptions() → *config.Config (loaded before Fx)
    │
    └── fx.New(
            fx.Supply(*config.Config),  // pre-loaded config
            logger.Module,
            infrastructure.Module,
            service.Module,
            server.Module,
            static.Module,
        ).Run()
```

## 模块组合

在 `cmd/litchi/server.go` 中按依赖顺序组合：

```go
fx.New(
    fx.Supply(loadedCfg),  // pre-loaded config
    logger.Module,
    infrastructure.Module,
    service.Module,
    server.Module,
    static.Module,
).Run()
```

## 未来模块

以下模块将在后续任务中实现：

| 模块 | 提供者 | 依赖 | 任务 |
|------|--------|------|------|
| `database` | `*gorm.DB` | config | T1.2.4 |
| `github` | `*github.Client` | config | T4.1.1 |
| `git` | `*GitOperator` | - | T4.2.1 |
| `agent` | `*AgentRunner` | config | T4.3.1 |
| `domain` | 领域服务 | database | T2.* |
| `application` | 应用服务 | domain, github, agent | T5.* |
| `api` | HTTP handlers | application | T6.* |
| `websocket` | WebSocket handler | application | T6.2.* |

## 最佳实践

1. **Provider vs Invoke**
   - Provider：创建并提供依赖
   - Invoke：执行初始化逻辑，不返回值

2. **生命周期钩子**
   ```go
   lifecycle.Append(fx.Hook{
       OnStart: func(ctx context.Context) error {
           // 启动资源
           return nil
       },
       OnStop: func(ctx context.Context) error {
           // 清理资源
           return nil
       },
   })
   ```

3. **依赖注入**
   ```go
   type Params struct {
       fx.In
       
       Logger *zap.Logger
       Config *config.Config
   }
   ```

4. **可选依赖**
   ```go
   type Params struct {
       fx.In
       
       Logger *zap.Logger `optional:"true"`
   }
   ```

5. **命名依赖**
   ```go
   fx.Provide(
       fx.Annotate(NewPrimaryDB, fx.ResultTags(`name:"primary"`)),
       fx.Annotate(NewSecondaryDB, fx.ResultTags(`name:"secondary"`)),
   )
   ```
# Go 审查规则

## 目录

1. [正确性](#正确性)
2. [安全性](#安全性)
3. [性能](#性能)
4. [架构合规](#架构合规)

---

## 正确性

### 错误处理

| 规则 | 严重程度 |
|------|----------|
| 不使用 `_` 忽略错误返回值 | 🔴 Critical |
| 使用 `errors.Is()` / `errors.As()` 比较 | 🟡 Medium |
| 错误中添加上下文信息 | 🟡 Medium |

```go
// ❌ 🔴 Critical
result, _ := db.First(&user)

// ✅ 正确
result := db.First(&user)
if result.Error != nil {
    return fmt.Errorf("failed to get user (id=%s): %w", id, result.Error)
}
```

### Nil 检查

| 规则 | 严重程度 |
|------|----------|
| 解引用前检查指针 | 🔴 Critical |
| 接口 nil 检查注意 type+value | 🟠 High |

```go
// ⚠️ 接口 nil 检查陷阱
var r io.Reader  // 接口包含 (type, value)
if r == nil {    // 只检查 value，type 可能非 nil
    // 可能不触发
}
```

### 并发安全

| 规则 | 严重程度 |
|------|----------|
| 共享数据用 mutex/channel 保护 | 🔴 Critical |
| 避免 goroutine 泄漏 | 🟠 High |
| `sync.Map` 用于读多写少 | 🟡 Medium |

```go
// ❌ 🔴 Critical - 数据竞争
var counter int
go func() { counter++ }()

// ✅ 正确
var counter int64
go func() { atomic.AddInt64(&counter, 1) }()
```

### Context 传递

| 规则 | 严重程度 |
|------|----------|
| 第一个参数是 `context.Context` | 🟡 Medium |
| 不在结构体中存储 context | 🟠 High |
| 用于超时/取消控制 | 🟡 Medium |

---

## 安全性

### 输入验证

| 规则 | 严重程度 |
|------|----------|
| 外部输入必须验证 | 🔴 Critical |
| SQL 参数化防止注入 | 🔴 Critical |
| `html/template` 防 XSS | 🔴 Critical |

```go
// ❌ 🔴 Critical - SQL 注入
db.Exec(fmt.Sprintf("SELECT * FROM users WHERE id = %s", input))

// ✅ 正确
db.Exec("SELECT * FROM users WHERE id = $1", input)
```

### 敏感数据

| 规则 | 严重程度 |
|------|----------|
| 不在日志打印密码/token | 🔴 Critical |
| 用 `crypto/rand` 生成随机 | 🟠 High |
| 敏感配置用环境变量 | 🟠 High |

### 路径安全

| 规则 | 严重程度 |
|------|----------|
| `filepath.Clean()` 清理路径 | 🟡 Medium |
| 检查路径遍历 `../` | 🟠 High |

---

## 性能

### 内存分配

| 规则 | 严重程度 |
|------|----------|
| 预分配 slice 容量 | 🟡 Medium |
| `sync.Pool` 复用对象 | 🟢 Low |
| 避免循环中频繁创建 | 🟡 Medium |

```go
// ❌ 频繁扩容
var items []Item
for _, v := range data {
    items = append(items, process(v))
}

// ✅ 预分配
items := make([]Item, 0, len(data))
```

### 数据库操作

| 规则 | 严重程度 |
|------|----------|
| 批量操作替代循环单条 | 🟠 High |
| GORM 避免 N+1，用 `Preload` | 🟠 High |

```go
// ❌ 🟠 High - N+1 查询
db.Find(&users)
for _, u := range users {
    db.Model(&u).Association("Orders").Find(&u.Orders)
}

// ✅ 正确
db.Preload("Orders").Find(&users)
```

### 字符串处理

| 规则 | 严重程度 |
|------|----------|
| 大量拼接用 `strings.Builder` | 🟡 Medium |
| 避免不必要的 `[]byte`/`string` 转换 | 🟢 Low |

---

## 架构合规

### DDD 分层

| 规则 | 严重程度 |
|------|----------|
| Domain 层无外部依赖 | 🔴 Critical |
| 依赖方向：Domain → Application → Infrastructure | 🟠 High |
| Repository 接口在 Domain 定义 | 🟡 Medium |

```go
// ❌ 🔴 Critical - Domain 直接依赖 Infrastructure
type WorkSession struct {
    db *gorm.DB  // 数据库连接
}

// ✅ 正确 - Domain 纯净
type WorkSession struct {
    ID     string
    Status Status
}

// Repository 接口在 Domain 定义
type WorkSessionRepository interface {
    Save(ctx context.Context, session *WorkSession) error
}
```

### 依赖注入

| 规则 | 严重程度 |
|------|----------|
| 使用 Uber Fx | 🟡 Medium |
| 依赖接口而非实现 | 🟡 Medium |
| 构造函数接收依赖 | 🟢 Low |

### 项目特定

| 技术 | 要求 |
|------|------|
| Web 框架 | Fiber v3 |
| ORM | GORM + PostgreSQL |
| 日志 | Zap (structured JSON) |
| 配置 | Viper |
| DI | Uber Fx |
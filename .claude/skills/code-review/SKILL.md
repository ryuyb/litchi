---
name: code-review
description: |
  Use ONLY when user explicitly requests: "/review", "review this code", "检查代码", "code review", "review PR/commit", or asks for feedback on shared code snippets. DO NOT auto-trigger when you see code changes or detect issues proactively. User must ask for review first.
---

# Code Review

执行代码审查，输出结构化报告。

## 触发方式

**仅响应用户显式请求**，不自动触发。

触发短语：
- `/review`、`review 这个 PR`、`检查代码`
- `code review`、`review this code`
- 用户粘贴代码并请求反馈

## 输入

| 类型 | 来源 | 示例 |
|------|------|------|
| 文件路径 | 用户指定 | `review src/handler.go` |
| PR 编号 | GitHub | `review PR #123` |
| Git diff | 当前变更 | `review staged changes` |
| 代码片段 | 直接粘贴 | 用户在消息中粘贴代码 |

**无输入时**: 使用 `git diff HEAD~1` 审查最近一次 commit。

## 工作流程

```
1. 收集代码 → 2. 识别语言 → 3. 加载规则 → 4. 分析问题 → 5. 输出报告
```

### Step 1: 收集审查范围

根据输入类型获取代码：
- **文件**: 直接读取
- **PR**: 用 `gh pr diff <number>` 获取
- **Git diff**: 用 `git diff` 或 `git diff --staged`
- **片段**: 直接分析用户提供的代码

### Step 2: 识别语言并加载规则

按文件扩展名选择规则文件：

| 扩展名 | 规则文件 |
|--------|----------|
| `.go` | `references/go-rules.md` |
| `.ts`, `.tsx` | `references/ts-rules.md` |
| 其他 | `references/common-rules.md` |

**多语言时**: 先读本文档的"核心约束"，再按语言读对应规则文件。

### Step 3: 执行审查

按以下顺序检查：
1. **正确性** → 错误处理、边界条件、类型安全
2. **安全性** → 注入、XSS、敏感数据
3. **性能** → N+1、内存、渲染
4. **架构** → DDD 分层、依赖方向

### Step 4: 输出报告

**必须**使用以下格式：

```markdown
## Code Review Report

### 概要
- 审查文件数：X
- 发现问题数：Y (🔴 Z, 🟠 A, 🟡 B, 🟢 C)

### 🔴 Critical Issues
#### `file:line` - 问题标题
**问题**: 描述
**影响**: 后果
**建议**: 修复方法 + 代码示例

### 🟠 High Issues
（同上结构）

### 🟡 Medium Issues
（同上结构）

### 🟢 Low Issues
（同上结构）

### 亮点
- 正面评价（可选）
```

### Step 5: 停止条件

审查完成当：
- 所有指定文件已分析
- 问题已按严重程度分类输出
- 报告已呈现给用户

**不执行修复** - 只输出报告，除非用户明确请求修改代码。

---

## 核心约束

这些规则 **始终适用**，无论语言：

### 严重程度定义

| 级别 | 触发条件 | 示例 |
|------|----------|------|
| 🔴 Critical | 安全漏洞、数据丢失、崩溃风险 | SQL 注入、未处理错误导致 nil pointer |
| 🟠 High | 逻辑错误、明显 bug、性能反模式 | N+1 查询、竞态条件 |
| 🟡 Medium | 可维护性问题、风格缺陷 | 缺少上下文的错误、命名不清 |
| 🟢 Low | 微小改进建议 | 可选的类型注解、注释优化 |

### 项目架构约束

本项目使用：
- **后端**: Go + Fiber v3 + GORM + Uber Fx
- **前端**: React + TanStack Start + TanStack Query + Zustand
- **架构**: DDD 分层（Domain → Application → Infrastructure → Presentation）

审查时检查：
- Domain 层不依赖 Infrastructure（无 `*gorm.DB` 等字段）
- 服务端数据获取使用 TanStack Query（不用手动 `fetch` + `useState`）
- 客户端状态使用 Zustand

---

## 规则文件导航

审查具体语言时，读取对应规则文件：

- **Go 代码**: → `references/go-rules.md`
- **TypeScript/React**: → `references/ts-rules.md`
- **通用规则**: → `references/common-rules.md`

规则文件包含：具体检查项、代码示例、修复建议模板。
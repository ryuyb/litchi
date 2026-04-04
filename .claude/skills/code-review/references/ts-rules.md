# TypeScript/React 审查规则

## 目录

1. [正确性](#正确性)
2. [安全性](#安全性)
3. [性能](#性能)
4. [架构合规](#架构合规)

---

## 正确性

### TypeScript 类型安全

| 规则 | 严重程度 |
|------|----------|
| 避免 `any`，用 `unknown` 或具体类型 | 🟠 High |
| 启用 `strict: true` | 🟡 Medium |
| 正确处理 Promise | 🟠 High |

```typescript
// ❌ 🟠 High
function process(data: any) { ... }

// ✅ 正确
function process(data: unknown) {
    if (typeof data === 'string') { ... }
}
```

### React Hooks 规则

| 规则 | 严重程度 |
|------|----------|
| Hooks 只在顶层调用 | 🔴 Critical |
| 依赖数组完整 | 🔴 Critical |
| 合理使用 `useCallback`/`useMemo` | 🟡 Medium |

```typescript
// ❌ 🔴 Critical - 缺少依赖
useEffect(() => {
    fetchData(userId);
}, []); // userId 变化时不重执行

// ✅ 正确
useEffect(() => {
    fetchData(userId);
}, [userId]);
```

### 状态管理

| 规则 | 严重程度 |
|------|----------|
| 服务端状态用 TanStack Query | 🟠 High |
| 客户端状态用 Zustand | 🟡 Medium |
| 避免 prop drilling | 🟢 Low |

---

## 安全性

### XSS 防护

| 规则 | 严重程度 |
|------|----------|
| 避免 `dangerouslySetInnerHTML` | 🔴 Critical |
| URL 参数用 `encodeURIComponent` | 🟠 High |
| 用户输入渲染前清理 | 🟠 High |

```typescript
// ❌ 🔴 Critical - XSS 风险
<div dangerouslySetInnerHTML={{ __html: userInput }} />

// ✅ 正确
<div>{userInput}</div>
```

### 敏感数据

| 规则 | 严重程度 |
|------|----------|
| 前端不存储敏感信息 | 🔴 Critical |
| API 密钥用环境变量 | 🟠 High |
| 生产构建不含 console.log | 🟡 Medium |

---

## 性能

### 渲染优化

| 规则 | 严重程度 |
|------|----------|
| `React.memo` 防不必要渲染 | 🟡 Medium |
| 大列表用虚拟滚动 | 🟠 High |
| 图片懒加载 | 🟢 Low |

```typescript
// ✅ React.memo
const UserCard = React.memo(({ user }: { user: User }) => {
    return <div>{user.name}</div>;
});
```

### 数据获取

| 规则 | 严重程度 |
|------|----------|
| TanStack Query 缓存/失效机制 | 🟠 High |
| 分页避免一次加载过多 | 🟡 Medium |
| Orval 生成的类型安全 hooks | 🟡 Medium |

```typescript
// ❌ 手动管理状态
const [user, setUser] = useState(null);
useEffect(() => { fetch(...).then(setUser) }, []);

// ✅ TanStack Query
const { data, isLoading, error } = useQuery({
    queryKey: ['user', userId],
    queryFn: () => fetchUser(userId),
});
```

### 代码分割

| 规则 | 严重程度 |
|------|----------|
| 路由级别分割 | 🟡 Medium |
| 大组件动态导入 | 🟡 Medium |
| SSR/SSG 优化 | 🟢 Low |

---

## 架构合规

### 组件结构

| 规则 | 严重程度 |
|------|----------|
| 展示/容器组件分离 | 🟡 Medium |
| 组件单一职责 | 🟡 Medium |
| 使用 shadcn/ui | 🟢 Low |

### 状态管理架构

| 类型 | 技术 |
|------|------|
| 服务端状态 | TanStack Query |
| 客户端状态 | Zustand |
| 表单状态 | TanStack Form |

```typescript
// ✅ Zustand store
const useUIStore = create<UIState>((set) => ({
    sidebarOpen: false,
    toggleSidebar: () => set((s) => ({ sidebarOpen: !s.sidebarOpen })),
}));
```

### 项目特定

| 技术 | 要求 |
|------|------|
| 框架 | TanStack Start (SSR/SSG) |
| 路由 | TanStack Router |
| 表格 | TanStack Table |
| 样式 | Tailwind CSS + shadcn/ui |
| API | Orval 生成 hooks |
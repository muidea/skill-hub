---
compatibility: Designed for Claude Code, Cursor, and OpenCode (or similar AI coding assistants)
description: Provides React + TypeScript development best practices. Use when building React applications with TypeScript, reviewing frontend code, or when the user asks about React patterns, TypeScript types, or modern frontend development.
metadata:
  author: skill-hub Team
  tags: react,typescript,frontend,web-development
  version: 1.0.0
name: react-typescript
---

# React + TypeScript最佳实践技能

## 项目结构

```
src/
├── components/         # 可复用组件
│   ├── common/        # 通用组件（Button, Input等）
│   ├── layout/        # 布局组件
│   └── features/      # 功能组件
├── hooks/             # 自定义Hooks
├── contexts/          # React Context
├── stores/            # 状态管理（Redux/Zustand）
├── services/          # API服务层
├── utils/             # 工具函数
├── types/             # TypeScript类型定义
├── constants/         # 常量定义
├── assets/            # 静态资源
│   ├── images/
│   ├── fonts/
│   └── styles/
├── pages/             # 页面组件
└── App.tsx            # 根组件
```

## TypeScript配置

### tsconfig.json推荐配置

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "lib": ["DOM", "DOM.Iterable", "ESNext"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["./src/*"],
      "@components/*": ["./src/components/*"],
      "@hooks/*": ["./src/hooks/*"]
    }
  },
  "include": ["src"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

## 组件开发

### 函数组件模式

```tsx
import React, { useState, useEffect, useCallback, memo } from "react";
import type { FC } from "react";

interface UserCardProps {
  user: User;
  onSelect?: (userId: string) => void;
  isActive?: boolean;
}

const UserCard: FC<UserCardProps> = memo(
  ({ user, onSelect, isActive = false }) => {
    const [isLoading, setIsLoading] = useState(false);

    const handleClick = useCallback(() => {
      if (onSelect) {
        onSelect(user.id);
      }
    }, [onSelect, user.id]);

    useEffect(() => {
      // 组件挂载/卸载逻辑
      return () => {
        // 清理函数
      };
    }, []);

    if (isLoading) {
      return <div>Loading...</div>;
    }

    return (
      <div
        className={`user-card ${isActive ? "active" : ""}`}
        onClick={handleClick}
        role="button"
        tabIndex={0}
        aria-label={`Select user ${user.name}`}
      >
        <img src={user.avatar} alt={user.name} />
        <div className="user-info">
          <h3>{user.name}</h3>
          <p>{user.email}</p>
        </div>
      </div>
    );
  },
);

UserCard.displayName = "UserCard";

export default UserCard;
```

### 自定义Hooks

```tsx
import { useState, useEffect, useCallback } from "react";
import type { DependencyList } from "react";

export function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState(value);

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => {
      clearTimeout(timer);
    };
  }, [value, delay]);

  return debouncedValue;
}

export function useFetch<T>(url: string, options?: RequestInit) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      const response = await fetch(url, options);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const result = await response.json();
      setData(result);
      setError(null);
    } catch (err) {
      setError(err as Error);
    } finally {
      setLoading(false);
    }
  }, [url, options]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refetch: fetchData };
}
```

## 状态管理

### Redux Toolkit模式

```tsx
// store/slices/userSlice.ts
import { createSlice, createAsyncThunk, PayloadAction } from "@reduxjs/toolkit";
import type { RootState } from "../store";

interface UserState {
  users: User[];
  loading: boolean;
  error: string | null;
}

const initialState: UserState = {
  users: [],
  loading: false,
  error: null,
};

export const fetchUsers = createAsyncThunk(
  "users/fetchUsers",
  async (_, { rejectWithValue }) => {
    try {
      const response = await fetch("/api/users");
      return await response.json();
    } catch (error) {
      return rejectWithValue("Failed to fetch users");
    }
  },
);

const userSlice = createSlice({
  name: "users",
  initialState,
  reducers: {
    addUser: (state, action: PayloadAction<User>) => {
      state.users.push(action.payload);
    },
    removeUser: (state, action: PayloadAction<string>) => {
      state.users = state.users.filter((user) => user.id !== action.payload);
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchUsers.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchUsers.fulfilled, (state, action) => {
        state.loading = false;
        state.users = action.payload;
      })
      .addCase(fetchUsers.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload as string;
      });
  },
});

export const { addUser, removeUser } = userSlice.actions;
export const selectUsers = (state: RootState) => state.users.users;
export default userSlice.reducer;
```

## 性能优化

### React.memo使用

```tsx
// 仅当props变化时重新渲染
const ExpensiveComponent = memo(
  ({ data }: { data: DataType }) => {
    // 复杂计算
    return <div>{/* 渲染逻辑 */}</div>;
  },
  (prevProps, nextProps) => {
    // 自定义比较函数
    return prevProps.data.id === nextProps.data.id;
  },
);
```

### useMemo和useCallback

```tsx
const expensiveCalculation = useMemo(() => {
  return computeExpensiveValue(deps);
}, [deps]);

const handleClick = useCallback(() => {
  // 事件处理逻辑
}, [dependencies]);
```

## 测试策略

### 单元测试（Jest + React Testing Library）

```tsx
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import Button from "./Button";

describe("Button Component", () => {
  it("renders button with correct text", () => {
    render(<Button>Click me</Button>);
    expect(screen.getByRole("button")).toHaveTextContent("Click me");
  });

  it("calls onClick handler when clicked", async () => {
    const handleClick = jest.fn();
    const user = userEvent.setup();

    render(<Button onClick={handleClick}>Click me</Button>);

    await user.click(screen.getByRole("button"));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it("disables button when disabled prop is true", () => {
    render(<Button disabled>Click me</Button>);
    expect(screen.getByRole("button")).toBeDisabled();
  });
});
```

### E2E测试（Cypress）

```tsx
describe("User Flow", () => {
  it("completes user registration", () => {
    cy.visit("/register");
    cy.get('[data-testid="email-input"]').type("test@example.com");
    cy.get('[data-testid="password-input"]').type("password123");
    cy.get('[data-testid="submit-button"]').click();
    cy.url().should("include", "/dashboard");
  });
});
```

## 代码质量

### ESLint配置

```json
{
  "extends": [
    "react-app",
    "react-app/jest",
    "plugin:@typescript-eslint/recommended",
    "plugin:react-hooks/recommended"
  ],
  "rules": {
    "@typescript-eslint/explicit-function-return-type": "off",
    "@typescript-eslint/no-explicit-any": "warn",
    "react-hooks/rules-of-hooks": "error",
    "react-hooks/exhaustive-deps": "warn"
  }
}
```

### Prettier配置

```json
{
  "semi": true,
  "trailingComma": "es5",
  "singleQuote": true,
  "printWidth": 80,
  "tabWidth": 2,
  "useTabs": false,
  "jsxSingleQuote": false,
  "arrowParens": "avoid"
}
```

## 部署优化

### 代码分割

```tsx
import { lazy, Suspense } from "react";

const Dashboard = lazy(() => import("./pages/Dashboard"));
const Settings = lazy(() => import("./pages/Settings"));

function App() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <Routes>
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/settings" element={<Settings />} />
      </Routes>
    </Suspense>
  );
}
```

### 环境变量管理

```env
# .env.development
REACT_APP_API_URL=http://localhost:3000/api
REACT_APP_DEBUG=true

# .env.production
REACT_APP_API_URL=https://api.example.com
REACT_APP_DEBUG=false
```

## 安全检查清单

- [ ] XSS防护：对用户输入进行转义
- [ ] CSRF防护：使用anti-CSRF token
- [ ] CORS配置：限制允许的源
- [ ] 敏感信息：不在客户端存储敏感数据
- [ ] 依赖安全：定期更新依赖包

## 性能检查清单

- [ ] 图片优化：使用WebP格式，懒加载
- [ ] 包大小：代码分割，tree shaking
- [ ] 渲染性能：避免不必要的重新渲染
- [ ] 网络请求：使用缓存，减少请求次数

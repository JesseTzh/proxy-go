# AGENTS.md

## 技术与运行约定

- 前端：React 19、TypeScript、Vite、Tailwind CSS 4、Radix UI/shadcn；入口为 `web/src/main.tsx`。
- 前端：所有页面元素均需要增加 data-testid 方便打包后定位
- 前端：所有组件优先使用 shadcn/ui
- Go 测试：始终使用工作区内缓存运行，例如 `GOCACHE=/Users/jessetzh/CodeSpace/proxy-go/.cache/go-build go test ./cmd/server ./internal/...`

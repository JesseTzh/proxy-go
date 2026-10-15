# 开发说明

## 环境与启动

项目由 Go 后端和 React/Vite 前端组成。建议使用 `go.mod` 声明的 Go 版本、Node.js 和 pnpm：

```sh
go version
node --version
pnpm --version
cd web
pnpm install
cd ..
./scripts/dev.sh
```

`scripts/dev.sh` 会创建隔离的 `.dev/proxy-go` 目录，启动后端 `127.0.0.1:30080` 和前端 `127.0.0.1:5173`。它使用 staging ACME 端点、关闭子进程启动，并将本地数据、证书和生成配置放在 `.dev/proxy-go`。

可通过以下环境变量覆盖本地启动参数：

| 变量 | 默认值 | 用途 |
|---|---|---|
| `PROXY_GO_INITIAL_PASSWORD` | `local-test-password` | 首次启动密码 |
| `PROXY_GO_ACME_EMAIL` | `dev@example.com` | ACME 注册邮箱 |
| `PROXY_GO_DEV_BACKEND_ADDR` | `127.0.0.1:30080` | Go 服务地址 |
| `PROXY_GO_DEV_INTERNAL_ADDR` | `127.0.0.1:30081` | 内部监听地址 |
| `PROXY_GO_DEV_FRONTEND_HOST` | `127.0.0.1` | Vite 监听主机 |
| `PROXY_GO_DEV_FRONTEND_PORT` | `5173` | Vite 监听端口 |

只调试前端时可以使用 Mock API：

```sh
cd web
VITE_MOCK_API=true pnpm dev
```

Mock 调试面板可以切换登录态、设置请求延迟、让下一次请求返回 503、查看请求记录和重置数据。Mock 数据只保存在浏览器本地。

## 结构

```text
cmd/server/main.go             Go 服务入口
internal/config/                YAML 和 PROXY_GO_* 配置加载
internal/httpapi/               REST 路由、认证中间件和处理器
internal/services/              域名、证书、反向代理、入站和运行时服务
internal/nginx/                 Nginx 配置生成、检查和进程管理
internal/singbox/               sing-box 配置生成、检查和进程管理
internal/runtimeconfig/         从数据库生成运行时快照
internal/models/                GORM 数据模型
configs/config.yml              Docker 默认配置
deploy/proxy-go.service         systemd 服务示例
scripts/                        开发、构建、部署和 CI 脚本
web/src/main.tsx                前端入口
web/src/pages/                  管理面板页面
web/src/components/             共享组件和 shadcn/ui 组件
web/src/lib/mockApi.ts          前端 Mock API
```

前端页面和交互必须为可定位的元素提供 `data-testid`。共享 UI 优先使用 `web/src/components/ui` 中的 shadcn/ui 组件。

## 配置与数据

默认配置见 [`configs/config.yml`](../configs/config.yml)。配置文件键会被 `PROXY_GO_` 环境变量覆盖，例如 `server.initial_addr` 对应 `PROXY_GO_SERVER_INITIAL_ADDR`。以下两个短变量是部署入口：

| 变量 | 用途 |
|---|---|
| `PROXY_GO_INITIAL_PASSWORD` | 首次启动时写入的管理密码；已有密码时不会覆盖 |
| `PROXY_GO_ACME_EMAIL` | ACME 证书注册和续期通知邮箱 |

默认运行目录如下：

| 路径 | 内容 |
|---|---|
| `/var/lib/proxy-go/proxy-go.db` | SQLite 数据库 |
| `/var/lib/proxy-go/certs` | ACME 证书和私钥 |
| `/var/lib/proxy-go/nginx` | 生成的 Nginx 配置 |
| `/var/lib/proxy-go/sing-box` | 生成的 sing-box 配置 |
| `/var/log/proxy-go` | 服务日志 |

不要提交这些目录或其中的密钥。

## API 概览

API 前缀为 `/api`，除认证和初始化接口外均需要登录：

| 方法 | 路径 | 说明 |
|---|---|---|
| `POST` | `/api/auth/login` | 管理员登录 |
| `POST` | `/api/auth/logout` | 退出登录 |
| `GET/POST` | `/api/domains` | 查询或创建域名 |
| `PUT/DELETE` | `/api/domains/:id` | 更新或删除域名 |
| `POST` | `/api/domains/:id/dns-check` | 执行 DNS 检查 |
| `POST` | `/api/domains/:id/certificate/issue` | 申请证书 |
| `POST` | `/api/domains/:id/certificate/renew` | 续期证书 |
| `GET/POST` | `/api/reverse-proxies` | 查询或创建反向代理 |
| `GET/POST` | `/api/inbounds` | 查询或创建 sing-box 入站 |
| `GET` | `/api/inbounds/:id/config` | 查看入站配置 |
| `GET` | `/api/inbounds/:id/share` | 获取分享链接和二维码数据 |
| `GET` | `/api/runtime/status` | 查看运行状态 |
| `GET` | `/api/runtime/nginx/config` | 查看生成的 Nginx 配置 |
| `GET` | `/api/audit-logs` | 查看审计日志 |
| `GET/PUT` | `/api/settings` | 查询或更新全局设置 |

路由的完整实现位于 [`internal/httpapi/router.go`](../internal/httpapi/router.go)。修改接口时应同步更新处理器、前端 API 类型和相关测试。

## 验证

Go 测试必须使用仓库缓存：

```sh
GOCACHE=/Users/jessetzh/CodeSpace/proxy-go/.cache/go-build go test ./cmd/server ./internal/...
GOCACHE=/Users/jessetzh/CodeSpace/proxy-go/.cache/go-build go vet ./cmd/server ./internal/...
```

前端检查和构建：

```sh
cd web
pnpm exec tsc --noEmit
pnpm build
pnpm test:login
pnpm test:dashboard
pnpm test:inbounds
```

构建完整产物：

```sh
./scripts/build.sh
```

该脚本构建前端、生成 `dist/proxy-go` 和 SHA-256 校验文件。

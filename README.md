# proxy-go

## 基本介绍

proxy-go 是一个面向单机部署的代理网关管理程序，提供 Web 管理面板，并统一编排证书、反向代理和 Xray 代理入口配置。

1. 自动申请 SSL 证书：基于域名配置发起 ACME HTTP-01 验证，完成证书签发、落盘和续期管理，供后续 HTTPS 入口和反向代理配置使用。
2. 端口反向代理：通过内置 Nginx 生成和加载反向代理配置，将普通 HTTPS 流量按域名转发到指定后端服务。
3. 使用 Xray 创建指定协议代理：通过 Xray-core 生成代理入站配置，REALITY 入口直接承接公网 HTTPS 端口，并由程序统一管理配置生成、校验和运行状态。

## 网络流量架构

当前运行模型中，公网 `443` 由 Xray REALITY 直接监听。合法 REALITY 客户端由 Xray 处理；非 REALITY、鉴权失败或普通 HTTPS 流量会通过 REALITY `dest` 回落到内置 Nginx，再由 Nginx 按域名处理管理面板和反向代理。

```text
Internet
  |
  | HTTP :80
  v
Nginx public HTTP
  |-- /.well-known/acme-challenge/ -> proxy-go internal API
  `-- other paths                   -> HTTPS redirect

Internet
  |
  | HTTPS :443
  v
Xray REALITY public inbound
  |-- valid REALITY client
  |     `-> Xray proxy outbound
  |
  `-- non-REALITY / invalid REALITY / normal HTTPS
        `-> REALITY dest: 127.0.0.1:30443
              |
              v
            Nginx internal HTTPS
              |-- management domain -> proxy-go Web/API
              `-- reverse proxy     -> configured upstream service
```

关键端口默认值：

- `80`：Nginx 公网 HTTP，处理 ACME HTTP-01 和 HTTP 到 HTTPS 跳转。
- `443`：Xray REALITY 公网入口。
- `127.0.0.1:30443`：内置 Nginx HTTPS 承接点，作为 REALITY `dest` 和反向代理入口。
- `127.0.0.1:30081`：proxy-go 内部 Web/API 服务。

VLESS Reality Vision 使用标准 REALITY 语义：

- 客户端连接地址：入口绑定域名或服务器地址，例如 `proxy.example.com:443`。
- REALITY `serverName` / 分享链接 `sni`：握手服务器，例如 `www.cloudflare.com`。
- Xray 服务端 `serverNames`：握手服务器。
- Xray 服务端 `dest`：内置 Nginx HTTPS 地址 `127.0.0.1:30443`。

由于 REALITY 入口直接占用公网 `443`，同一时间只能启用一个 `VLESS Reality Vision` 入站。

## 项目结构

```text
cmd/server/          Go 服务入口
internal/            后端核心代码
internal/httpapi/    REST API 路由、中间件和处理器
internal/services/   业务服务层
internal/nginx/      Nginx 配置生成与进程管理
internal/xray/       Xray 配置生成与进程管理
configs/             默认配置文件
runtime-assets/      运行时资产
scripts/             开发与构建脚本
web/                 React 前端项目
web/src/main.tsx     前端入口
docker-compose.yml   Docker Compose 启动配置
Dockerfile.amd64     AMD64 Docker 镜像构建文件
```

## 技术栈

- 后端：Go、Gin、GORM、SQLite、Koanf
- 前端：React 19、TypeScript、Vite、Tailwind CSS 4、Radix UI/shadcn
- 运行时：Nginx、Xray-core
- 部署：Docker、Docker Compose

## 启动命令

### Docker Compose

```bash
cp .env.example .env
docker compose up -d
```

访问管理面板：

```text
http://127.0.0.1:30080
```

### 本地开发

```bash
cd web
pnpm install
pnpm build
cd ..
./scripts/dev.sh
```

# proxy-go

## 基本介绍

proxy-go 是一个面向单机部署的代理网关管理程序，提供 Web 管理面板，并统一编排证书、反向代理和 Xray 代理入口配置。

1. 自动申请 SSL 证书：基于域名配置发起 ACME HTTP-01 验证，完成证书签发、落盘和续期管理，供后续 HTTPS 入口和反向代理配置使用。
2. 端口反向代理：通过内置 Nginx 生成和加载反向代理配置，将普通 HTTPS 流量按域名转发到指定后端服务。
3. 使用 Xray 创建指定协议代理：通过 Xray-core 生成代理入站配置，公网 HTTPS 由 Nginx stream 按 SNI 分流到 Xray 或内部 HTTPS，并由程序统一管理配置生成、校验和运行状态。

## 网络流量架构

当前运行模型中，公网 `443` 由 Nginx stream 监听。Nginx 使用 `ssl_preread` 读取 TLS ClientHello 的 SNI：匹配 REALITY 握手服务器的流量转发到本机 Xray 入站，其他普通 HTTPS 流量转发到内置 Nginx HTTPS，再由 Nginx 按域名处理管理面板和反向代理。

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
Nginx stream ssl_preread
  |-- SNI = REALITY handshake server, e.g. apple.com
  |     `-> 127.0.0.1:31001 Xray REALITY inbound
  |           `-> valid REALITY client -> Xray proxy outbound
  |
  `-- default / normal HTTPS
        `-> 127.0.0.1:30443 Nginx internal HTTPS
              |-- management domain -> proxy-go Web/API
              `-- reverse proxy     -> configured upstream service
```

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

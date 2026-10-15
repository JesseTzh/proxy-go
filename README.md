# proxy-go

proxy-go 是一个面向单机部署的代理网关管理程序。它提供 Web 管理面板，统一管理域名、ACME 证书、Nginx 反向代理和 sing-box 入站。

## 主要功能

- 管理域名并执行 DNS 检查。
- 通过 ACME HTTP-01 自动申请、续期和删除证书。
- 为域名配置 HTTPS 反向代理和上游服务。
- 管理 sing-box VLESS Reality Vision、AnyTLS 和 Hysteria2 入站，生成分享链接或二维码。
- 查看 Nginx、sing-box 运行状态、生成配置和审计日志。
- 使用 Mock API 调试前端，并在 CI 中生成桌面截图与项目文档站。

## 快速开始

生产环境使用 Docker Compose：

```sh
cp .env.example .env
# 编辑 .env，至少设置 PROXY_GO_INITIAL_PASSWORD
docker compose up -d
```

管理面板默认地址为 <http://127.0.0.1:30080>。生产环境应将 80/443 暴露到公网，并为数据目录配置持久化存储。

本地开发：

```sh
cd web
pnpm install
cd ..
./scripts/dev.sh
```

前端地址为 <http://127.0.0.1:5173>，后端地址为 <http://127.0.0.1:30080>。本地开发默认使用 Let's Encrypt staging ACME 端点，不会申请生产证书。

## 文档

- [开发说明](docs/DEVELOPMENT.md)：架构、目录、配置、API、本地开发和验证命令。
- [发布说明](docs/RELEASE.md)：Docker 部署、CI、容器冒烟检查和项目文档站发布。
- [前端设计说明](web/DESIGN.md)：管理面板的视觉和交互约定。

## 技术栈

- 后端：Go、Gin、GORM、SQLite、Koanf
- 前端：React 19、TypeScript、Vite、Tailwind CSS 4、Radix UI/shadcn
- 运行时：Nginx、sing-box
- 部署：Docker、Docker Compose

## 网络流量架构

公网 `443` 由 Nginx stream 监听，并通过 `ssl_preread` 按 TLS SNI 分流：代理 SNI 转发到 sing-box 入站，普通 HTTPS 转发到内部 Nginx；内部 Nginx 再按域名处理管理面板和反向代理。公网 `80` 用于 ACME challenge 和 HTTPS 跳转。

```text
Internet :80  -> Nginx public HTTP -> ACME challenge / HTTPS redirect
Internet :443 -> Nginx stream ssl_preread
                  |-- proxy SNI    -> sing-box 127.0.0.1:31001/31002/31003
                  `-- normal HTTPS  -> Nginx 127.0.0.1:30443
                                      |-- management domain -> proxy-go
                                      `-- reverse proxy      -> upstream
```

## 许可证和安全

请勿提交 `.env`、数据库、证书、私钥或运行日志。生产环境必须使用 HTTPS，并备份 Docker Compose 挂载的数据目录和日志目录。

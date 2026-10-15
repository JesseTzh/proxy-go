# 发布说明

## Docker 部署

准备持久化目录和环境变量：

```sh
mkdir -p data logs
cp .env.example .env
# 编辑 .env，设置强密码和 ACME 邮箱
docker compose pull
docker compose up -d
```

`docker-compose.yml` 暴露管理初始端口 `30080`、公网 HTTP `80` 和 HTTPS `443`，并将 `./data` 挂载到 `/var/lib/proxy-go`、`./logs` 挂载到 `/var/log/proxy-go`。首次登录后应设置管理域名并关闭初始端口，再通过 HTTPS 访问管理面板。

生产环境至少需要：

- DNS 将管理域名和代理域名解析到服务器。
- 服务器允许 TCP 80、TCP/UDP 443，以及首次初始化所需的 30080。
- ACME HTTP-01 能够访问公网 80 端口。
- `data` 和 `logs` 使用持久化磁盘，并定期备份数据库、证书和生成配置。

构建本地 AMD64 镜像：

```sh
docker build -f Dockerfile.amd64 -t proxy-go:local .
docker run --rm -p 30080:30080 -p 80:80 -p 443:443/tcp -p 443:443/udp \
  -e PROXY_GO_INITIAL_PASSWORD='change-me-now' \
  -e PROXY_GO_ACME_EMAIL='admin@example.com' \
  -v "$PWD/data:/var/lib/proxy-go" -v "$PWD/logs:/var/log/proxy-go" \
  proxy-go:local
```

## CI 工作流

仓库仅保留 `.github/workflows/dockerhub.yml`，统一负责检查、项目文档、容器验证、镜像发布和部署。

执行顺序：

1. `validate`：Go 格式、vet、测试，以及前端类型检查和生产构建。
2. 检查通过后，`container` 与 `documentation` 并行：前者构建 AMD64 镜像并验证服务及新旧数据库启动；后者构建 Mock 前端、验证登录及各菜单操作、生成桌面截图，并构建与验证文档站。
3. 两项验证均通过后，默认分支发布 Docker Hub 镜像和 GitHub Pages 文档站；`v*` 标签只发布镜像。
4. 默认分支镜像发布成功后，执行服务器远程部署。Pages 发布独立于 Docker Hub 凭据和服务器部署。

Pull Request 只执行检查并上传文档 Artifact，不发布镜像或部署。手动运行也执行全部检查，选择 `main` 时会发布镜像；Pages 和服务器部署仅由推送 `main` 触发。镜像发布和服务器部署需要仓库配置 Docker Hub 凭据及 SSH 部署凭据。

## 容器本地验证

需要 Docker、Go、Node.js、pnpm、curl 和 jq：

```sh
GOCACHE=/Users/jessetzh/CodeSpace/proxy-go/.cache/go-build go test ./cmd/server ./internal/...
docker build -f Dockerfile.amd64 -t proxy-go:ci-local .
TEST_IMAGE=proxy-go:ci-local ./scripts/ci-smoke-services.sh
TEST_IMAGE=proxy-go:ci-local ./scripts/ci-smoke-startup.sh
```

启动检查会创建临时容器和数据卷，验证健康状态、管理 API、全新数据库初始化、重启后数据库仍可用。失败时脚本会输出容器日志和检查结果。

## GitHub Pages 项目文档

文档任务构建 `VITE_MOCK_API=true` 的前端，仅在桌面视口验证页面和关键操作，并上传 `proxy-go-documentation` Artifact。推送 `main` 后将 `mock-page-artifacts/site` 发布到 GitHub Pages。

首页直接读取同一次提交中的 `README.md`，通过 Markdown 解析器生成正文，支持标题目录、代码块、列表和表格。README 中的开发、发布和设计说明链接指向站内解析后的文档；其他仓库相对链接指向对应提交的源文件。使用指南按“登录 → Dashboard → 域名管理 → 反向代理 → 代理管理 → 系统设置 → 审计日志”排列，每章提供功能说明和桌面截图。Mock 调试工具单独放在开发导航中。

首次启用时，在仓库 **Settings → Pages → Build and deployment** 将 Source 设为 **GitHub Actions**。

本地生成并验证文档站：

```sh
cd web
pnpm install --frozen-lockfile
VITE_MOCK_API=true pnpm build
cd ..
npm ci --prefix scripts/ci/web
npx --prefix scripts/ci/web playwright install chromium
bash scripts/ci/verify-mock-pages.sh
```

生成的静态站点位于 `mock-page-artifacts/site`。更新 README、页面说明或文档样式后，可使用已有截图重新生成：

```sh
node scripts/ci/web/build-mock-gallery.mjs
```

文档站只包含仓库文档、演示截图和功能说明，不应上传包含真实凭据、生产域名或生产日志的文件。

## 版本与回滚

使用版本标签触发正式镜像发布：

```sh
git tag v0.1.0
git push origin v0.1.0
```

回滚时将 Compose 镜像固定到已验证的标签，保留 `data` 和 `logs` 挂载目录，再执行 `docker compose up -d`。不要通过删除数据卷回滚数据库；数据库迁移和回滚策略应在发布前确认。

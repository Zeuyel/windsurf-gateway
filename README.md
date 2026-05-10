# Windsurf Gateway

Windsurf Gateway 是一个参照 `augment-open-gateway` 思路实现的远程账号管理与请求分发网关。目标是把 Windsurf 客户端的标准 API 请求转发到远程 gateway，由 gateway 统一完成用户鉴权、账号池选择、额度管理和请求代理。

## 核心架构

- **Windsurf 客户端 Patch**：通过修改 Windsurf 配置项 `codeium.apiServerUrl`，把默认的 `https://server.codeium.com` 改为你的 gateway 地址。
- **Gateway 接口层**：Windsurf 原生 API path 会直接进入 gateway；`/proxy/*path` 也保留为显式代理入口。
- **账号池管理**：管理员在后台录入多个 Windsurf/Codeium token 和租户地址。
- **负载分发**：客户端不需要再登录真实 Windsurf 账号；gateway 从后台账号池选择真实 token 代发请求。
- **用户系统**：用于 gateway 管理后台、额度控制和可选客户端标识；Windsurf patch 后不依赖原 Windsurf 注册登录。
- **隐私隔离**：默认开启 privacy mode，gateway 不向上游透传客户端 IP / Cookie / Origin / Referer / 原始 session header，而是按 backend 账号生成稳定的上游 `User-Agent` 与 `X-Request-Session-Id`。

## Windsurf Patch 关键发现

Windsurf 插件中存在默认 API 地址：

```text
DEFAULT_API_SERVER_URL="https://server.codeium.com"
```

同时 Windsurf 支持配置覆盖：

```json
{
  "codeium.apiServerUrl": "https://your-gateway.example.com"
}
```

因此相比 Augment 的解包注入，Windsurf 的优先方案是配置 patch；必要时也可以直接 patch：

```text
/opt/windsurf/resources/app/extensions/windsurf/dist/extension.js
```

## 快速开始

```bash
cp .env.example .env
# 修改 .env 中 MySQL、Redis、JWT_SECRET、管理员密码

docker compose up -d
```

访问：

- 前端：`http://localhost:8080`
- 管理后台：`http://localhost:8080/admin`
- 默认管理员：`admin / admin123`

## 本地开发

```bash
go mod tidy
go build -buildvcs=false -o ./windsurf-gateway ./cmd/

cd web
npm install
npm run dev
```

直接 `go run ./cmd` 时，程序会优先尝试读取当前目录下的 `.env`。

## Patch Windsurf

```bash
cd patch-tool
npm install
node patch.js --gateway=https://your-gateway.example.com
```

也可以使用环境变量：

```bash
WINDSURF_GATEWAY_URL=https://your-gateway.example.com node patch.js
```

恢复：

```bash
node patch.js --restore
```

Patch 工具会尝试处理三类位置：

- `~/.config/Windsurf/User/settings.json`
- `/opt/windsurf/resources/app/extensions/windsurf/dist/extension.js`
- `~/.config/Windsurf/User/globalStorage/state.vscdb`

同时会在 `~/.config/Windsurf/User/globalStorage/windsurf-gateway-patch.json` 保存一个本地 placeholder key。这个 key 只用于 gateway 内部把同一台 patched Windsurf 稳定绑定到某个 backend 账号，不会转发给上游服务；真正发往上游的仍然是 backend token 池中的真实 token。

## Patcher Binary

现在仓库里还提供了一个独立的 Go patcher，可直接作为 GitHub Release 二进制发布，不依赖 Node。

本地启动 UI：

```bash
go run ./cmd/windsurf-patcher
```

它会启动一个本地网页，用来填写：

- `Gateway Endpoint`
- 可选 `Gateway User Token`
- patch 模式
- 可选自定义 Windsurf 配置目录 / 安装目录

执行 patch / restore 前，先完全退出 Windsurf，避免 `state.vscdb` 或 `extension.js` 被客户端占用。

两种路由模式：

- `Anonymous Sticky`
  - 不给 gateway 用户 token
  - patcher 会生成每台客户端独有的 placeholder key
  - gateway 用它做匿名 sticky 分配
- `Gateway User Token`
  - 填 `ws-...` token
  - Windsurf 请求会先带这个 token 到 gateway
  - gateway 再按对应用户做分发、额度和速率控制

命令行无界面模式：

```bash
go run ./cmd/windsurf-patcher -apply --gateway=https://your-gateway.example.com
```

使用 gateway 用户 token：

```bash
go run ./cmd/windsurf-patcher -apply \
  --gateway=https://your-gateway.example.com \
  --auth-token=ws-xxxxxxxx
```

检测当前 patch 状态：

```bash
go run ./cmd/windsurf-patcher -detect
```

恢复最近一次备份：

```bash
go run ./cmd/windsurf-patcher -restore
```

## 网关请求入口

Windsurf patch 后，客户端的 `codeium.apiServerUrl` 指向 gateway 根地址。Windsurf 原生请求 path 会由 gateway 的 `NoRoute` 兜底接住并转发；同时保留显式入口：

```text
/proxy/*path
```

如果请求带 `Authorization: Bearer ws-xxx`，gateway 会按对应 gateway 用户做额度和频率控制；如果没有用户 token，gateway 会根据 patch 写入的本地 placeholder key 对客户端做匿名 sticky，再从系统 Windsurf token 池选择并固定到可用真实账号代发请求。也就是说，patch 后客户端不需要再走 Windsurf 的真实注册/登录流程，账号信息来自 gateway 后台维护的 token 池。

如果你是从旧版本 patch 升级，请重新执行一次：

```bash
cd patch-tool
node patch.js --gateway=https://your-gateway.example.com --mode=all
```

Node 版 patch 工具现在也支持可选 `--auth-token=ws-...`：

```bash
node patch.js \
  --gateway=https://your-gateway.example.com \
  --auth-token=ws-xxxxxxxx \
  --mode=all
```

这样会把旧的共享 placeholder 升级成“每个 patched 客户端独有”的 placeholder，避免 gateway 只能退回到 `Authorization + IP` 做匿名分配。

## GitHub Actions

仓库现在包含两条工作流：

- `.github/workflows/docker-image.yml`
  - 在 `master` 和 `v*` tag 上构建并推送 Docker 镜像
- `.github/workflows/patcher-release.yml`
  - 在 `patcher-v*` tag 上构建 `windows/linux/macos` patcher 二进制并发布 GitHub Release
- `.github/workflows/ci.yml`
  - 在 `master` / PR 上跑 `go test`、后端构建、patcher 构建、web build

推送前需要先在 GitHub 仓库里配置这些变量和 secrets。

Repository variables:

- `DOCKER_REGISTRY`
  - 例如 `ghcr.io` 或 `docker.io`
- `DOCKER_IMAGE_NAME`
  - 会和 registry 拼成 `${DOCKER_REGISTRY}/${DOCKER_IMAGE_NAME}`
  - 例如：
    - `your-org/windsurf-gateway`
    - `yourname/windsurf-gateway`
- `DOCKER_PLATFORMS`
  - 可选
  - 默认建议：`linux/amd64,linux/arm64`

Repository secrets:

- `DOCKER_USERNAME`
- `DOCKER_PASSWORD`

如果你要发 Docker 镜像到 GHCR，一般可以这样配：

- `DOCKER_REGISTRY=ghcr.io`
- `DOCKER_IMAGE_NAME=<github-owner>/windsurf-gateway`
- `DOCKER_USERNAME=<github-username>`
- `DOCKER_PASSWORD=<github-personal-access-token>`

`patcher-release.yml` 不依赖这些 Docker secrets；它只需要你打一个 `patcher-v*` tag。

## 管理接口

- `POST /api/auth/login` 管理员登录
- `GET /api/tokens` token 列表
- `POST /api/tokens` 添加 token
- `POST /api/tokens/batch-import` 批量导入 token
- `GET /api/users` 用户列表
- `GET /api/stats/overview` 统计概览
- `GET /api/request-records` 请求记录

## 已验证

- `go build -buildvcs=false -o ./windsurf-gateway ./cmd/` 通过
- `npm run build` 通过

# Windsurf Gateway

Windsurf Gateway 是一个参照 `augment-open-gateway` 思路实现的远程账号管理与请求分发网关。目标是把 Windsurf 客户端的标准 API 请求转发到远程 gateway，由 gateway 统一完成用户鉴权、账号池选择、额度管理和请求代理。

## 核心架构

- **Windsurf 客户端 Patch**：通过修改 Windsurf 配置项 `codeium.apiServerUrl`，把默认的 `https://server.codeium.com` 改为你的 gateway 地址。
- **Gateway 接口层**：Windsurf 原生 API path 会直接进入 gateway；`/proxy/*path` 也保留为显式代理入口。
- **账号池管理**：管理员在后台录入多个 Windsurf/Codeium token 和租户地址。
- **负载分发**：客户端不需要再登录真实 Windsurf 账号；gateway 从后台账号池选择真实 token 代发请求。
- **用户系统**：用于 gateway 管理后台、额度控制和可选客户端标识；Windsurf patch 后不依赖原 Windsurf 注册登录。

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

## 网关请求入口

Windsurf patch 后，客户端的 `codeium.apiServerUrl` 指向 gateway 根地址。Windsurf 原生请求 path 会由 gateway 的 `NoRoute` 兜底接住并转发；同时保留显式入口：

```text
/proxy/*path
```

如果请求带 `Authorization: Bearer ws-xxx`，gateway 会按对应 gateway 用户做额度和频率控制；如果没有用户 token，gateway 会直接从系统 Windsurf token 池选择可用真实账号代发请求。也就是说，patch 后客户端不需要再走 Windsurf 的真实注册/登录流程，账号信息来自 gateway 后台维护的 token 池。

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

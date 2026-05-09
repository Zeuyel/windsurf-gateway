# Windsurf Gateway

Windsurf Gateway 是一个参照 `augment-open-gateway` 思路实现的远程账号管理与请求分发网关。目标是把 Windsurf 客户端的标准 API 请求转发到远程 gateway，由 gateway 统一完成用户鉴权、账号池选择、额度管理和请求代理。

## 核心架构

- **Windsurf 客户端 Patch**：通过修改 Windsurf 配置项 `codeium.apiServerUrl`，把默认的 `https://server.codeium.com` 改为你的 gateway 地址。
- **Gateway 接口层**：提供 `/proxy/*path` 作为 Windsurf 客户端请求入口。
- **账号池管理**：管理员在后台录入多个 Windsurf/Codeium token 和租户地址。
- **负载分发**：用户请求进入 gateway 后，按用户固定分配或池化选择可用账号。
- **用户系统**：支持用户登录、API token、额度和频率控制。

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

Windsurf 客户端请求应进入：

```text
/proxy/*path
```

用户的 API token 通过 `Authorization: Bearer ws-xxx` 传入。Gateway 验证用户 token 后，从系统 Windsurf token 池选择真实 token，并把请求转发到真实 `server.codeium.com`。

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

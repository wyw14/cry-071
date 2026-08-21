# 公共空间反馈处理追踪模块

这是一个离线可运行的公共空间问题提交与处理系统。提交者可获得受理编号和查询令牌，处理人员在队列中受理、补充、分派、回复、合并和关闭反馈；所有对外状态变化都会追加到不可修改的时间线。系统同时提供受控附件、敏感字段脱敏、审计、解决公告和区域处理质量分析。

## 模块职责

- `cmd/server`：配置加载、依赖装配、HTTP 服务与优雅停机。
- `internal/domain`：反馈聚合、六态状态机、时间线、附件、合并组、公告、审计和报表值对象。
- `internal/application`：提交、查询、状态流转、分派、回复、补充、满意度、批量操作、合并、公告、附件与趋势用例。
- `internal/repository/memory`：支持事务回滚、乐观并发和深拷贝隔离的离线仓储。
- `internal/repository/postgres`：基于 pgx 的 PostgreSQL 仓储和 context 事务绑定。
- `internal/transport/http`：Gin `/api/v1` 路由、请求校验、分页/排序和稳定错误响应。
- `internal/middleware`：request_id、角色身份、CORS、安全响应头、超时、日志和 panic 恢复。
- `internal/platform`：本地附件、通知、时钟和 ID 适配器。
- `web`：Vue 3、TypeScript、Vite、Pinia 中文工作台。
- `migrations`：可重复执行的数据库结构与演示元数据。
- `api/openapi`：OpenAPI 3.0 接口契约。

## 状态规则

状态按以下闭环流转：

```text
待受理 -> 待补充 -> 待受理/处理中
待受理 -> 处理中 -> 待确认 -> 已关闭
处理中 -> 待补充/已驳回
已关闭 -> 处理中（必须填写明确重开原因）
已驳回 -> 待受理（必须填写重新受理原因）
```

进入处理中前必须完成分派。待补充、驳回、重开和重新受理都要求具体原因。提交、公开回复、分派、状态变化、补充材料、合并、满意度和公告关联会产生时间线事件；内部备注仅对工作人员可见。PostgreSQL 通过 trigger 禁止更新或删除时间线记录。

## 本地启动

需要 Go 1.24+。默认使用内存仓储，不依赖外部服务：

```bash
cp .env.example .env
go run ./cmd/server
```

健康检查：

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

前端开发环境：

```bash
cd web
npm ci
npm run dev
```

浏览器打开 `http://localhost:5173`。Vite 会把 `/api` 与健康检查代理到本机 8080 端口。

## PostgreSQL 与迁移

创建数据库后设置 `STORE_MODE=postgres` 和 `DATABASE_URL`，再运行：

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 \
  -f migrations/001_init.sql \
  -f migrations/002_seed.sql
go run ./cmd/server
```

两份迁移均可重复执行。种子数据只使用 `ON CONFLICT DO NOTHING`，不会覆盖已修改的区域、设施类别和反馈主题。

## Docker

完整 PostgreSQL、迁移、API 与前端环境：

```bash
docker compose up --build
```

- API：`http://localhost:8080`
- 前端：`http://localhost:8088`

镜像均使用官方多架构基础镜像，可在 amd64 或 arm64 主机上构建。Compose 使用本地卷保存 PostgreSQL 数据和受控附件，不调用第三方接口、云存储或在线消息服务。

## 配置

| 环境变量 | 默认值 | 用途 |
|---|---|---|
| `APP_ENV` | `development` | 日志和 Gin 运行模式 |
| `HTTP_ADDRESS` | `:8080` | HTTP 监听地址 |
| `STORE_MODE` | `memory` | `memory` 或 `postgres` |
| `DATABASE_URL` | 空 | PostgreSQL 连接串 |
| `ATTACHMENT_ROOT` | `./var/attachments` | 本地附件根目录 |
| `REQUEST_TIMEOUT` | `5s` | 单个请求 context 超时 |
| `SHUTDOWN_TIMEOUT` | `10s` | 优雅停机等待时间 |
| `ALLOWED_ORIGINS` | `http://localhost:5173` | 逗号分隔 CORS 白名单 |
| `QUERY_TOKEN_PEPPER` | 仅开发值 | 查询令牌 HMAC pepper，生产必须替换 |
| `SEED_DEMO_DATA` | `true` | 内存适配器是否写入演示元数据 |

不得把 `.env`、真实连接串、查询令牌或附件数据提交到仓库。

## 接口示例

提交反馈：

```bash
curl -X POST http://localhost:8080/api/v1/feedback \
  -H 'Content-Type: application/json' \
  -d '{"area_id":"area-central-park","facility_category_id":"lighting","subject_code":"safety","priority":"high","title":"公园东门路灯持续闪烁","description":"夜间路灯持续闪烁并影响台阶辨识","location":"中心公园东门","submitter_name":"市民"}'
```

响应包含受理编号和只展示一次的查询令牌。之后通过：

```bash
curl http://localhost:8080/api/v1/feedback/query/<query-token>
```

工作人员接口使用本地身份头进行离线演示：

```bash
curl 'http://localhost:8080/api/v1/admin/queue?page=1&page_size=20&sort=due_at&order=asc' \
  -H 'X-Actor-ID: manager-local' \
  -H 'X-Actor-Role: manager'
```

错误响应固定包含 `code`、`message`、`field_errors` 和 `request_id`。完整 method、path 和 schema 见 `api/openapi/openapi.yaml`。

## 事务、并发与隐私

- 提交反馈会在同一事务中保存反馈、查询令牌、首条时间线和审计记录。
- 状态、分派、合并和公告发布通过版本号执行乐观并发控制。
- PostgreSQL 使用 serializable transaction；内存适配器使用隔离快照，失败操作不会暴露部分写入。
- 查询令牌只存 HMAC 摘要；公开视图隐藏内部备注，并脱敏手机、邮箱和身份证号。
- 附件限制为 JPEG、PNG、PDF 和 10 MB，文件名与存储 key 均执行路径逃逸检查。
- 附件读取要求管理权限、审计权限或与反馈匹配的查询令牌。
- 通知、文件与定时超期扫描均为可测试的本地适配器。

## 测试与构建

```bash
go build ./...
go test ./...
go test -race ./...
go vet ./...
cd web && npm ci && npm test && npm run build
```

行为测试覆盖状态机、重开原因、不可变时间线、事务回滚、乐观并发、查询令牌、脱敏、内部备注隔离、合并公告、附件权限、HTTP 提交查询、排序白名单和稳定错误结构。仓储集成路径由 PostgreSQL migration 和 pgx adapter 提供；需要真实数据库时可通过 Compose 执行。

## 实际验证结果

项目交付前在 Windows amd64 环境执行上述 Go build/test/race/vet、前端 test/build 和 Docker 当前平台构建。具体结果以当前提交对应的 CI 或交付校准记录为准，不在仓库中保存运行期日志和缓存。

# AI Gateway (AIGateway)

本项目是基于 Gin 的 API 网关，当前重点是 AI 代理链路：认证、模型路由/映射、限流、配额、缓存、负载均衡、可观测、Prompt 装饰、IP 限制和 CORS。

## 1. 快速启动

### 1.1 环境要求

- Go 1.24+
- MySQL 8.x（默认 `127.0.0.1:3306`）
- Redis 6.x/7.x（默认 `127.0.0.1:6379`）

### 1.2 配置文件

开发环境默认目录：`./conf/dev/`

- `conf/dev/base.toml`：管理端（dashboard）服务配置
- `conf/dev/proxy.toml`：代理端（server）服务配置，默认 `:8080`
- `conf/dev/mysql_map.toml`：MySQL 连接
- `conf/dev/redis_map.toml`：Redis 连接
- `conf/dev/ai.toml`：AI 网关能力总配置（开关、链路参数等）

仓库默认提交的是脱敏模板文件（`*.example.toml`），本地运行前请复制为真实配置：

```powershell
Copy-Item .\conf\dev\base.example.toml .\conf\dev\base.toml
Copy-Item .\conf\dev\proxy.example.toml .\conf\dev\proxy.toml
Copy-Item .\conf\dev\mysql_map.example.toml .\conf\dev\mysql_map.toml
Copy-Item .\conf\dev\redis_map.example.toml .\conf\dev\redis_map.toml
Copy-Item .\conf\dev\redis.example.toml .\conf\dev\redis.toml
Copy-Item .\conf\dev\ai.example.toml .\conf\dev\ai.toml
```

### 1.3 数据库初始化

先创建数据库（默认名 `gateway`），再导入表结构：

```powershell
mysql -uroot -p gateway < .\sql\ai_tables.sql
```

如果你的环境已经有表，可跳过导入。

### 1.4 启动命令

在项目根目录执行：

1. 启动 AI 代理服务（主链路）

```powershell
go run main.go -endpoint server -config ./conf/dev/
```

2. 启动管理后台服务（配置管理/Swagger）

```powershell
go run main.go -endpoint dashboard -config ./conf/dev/
```

### 1.5 健康检查

- 代理服务：`GET http://127.0.0.1:8080/ping`
- 管理服务：`GET http://127.0.0.1:8880/ping`
- Swagger：`http://127.0.0.1:8880/swagger/index.html`

## 2. AI 链路顺序（server 模式）

注册位置：`http_proxy_router/router.go`

当前是“阶段化 + 动态计划”执行，不再是固定硬编码全开链路。

### 2.1 执行阶段

1. `HTTPAccessModeMiddleware`
2. `PipelinePlanMiddleware`
3. `PipelineExecutorMiddleware`（按 Plan 顺序执行插件链）

说明：
- 主链路已由 Executor 接管，不再在 `router.go` 静态逐条注册 AI/代理中间件。
- 当前已迁移为 native plugin：`core.flow_count/core.flow_limit/core.white_list/core.black_list/ai.auth/ai.ip_restriction/ai.token_ratelimit/ai.quota/ai.model_router/ai.prompt_decorator/ai.cache/ai.load_balancer/ai.observability/proxy.header_transfer/proxy.strip_uri/proxy.url_rewrite/proxy.reverse_proxy`。
- 当前仅 `ai.cors` 仍通过 adapter 运行（Preflight 阶段），其余主链路插件已迁移为 native plugin。

### 2.2 动态计划能力

- 按 `service_id + config_version` 构建并缓存 Plan。
- 支持插件依赖校验（strict / non-strict）。
- 支持全局与服务级插件优先级覆写。
- Debug 模式可返回 `X-Pipeline-*` 头查看计划。

### 2.3 热更新行为

服务配置变更后（新增/更新/删除），会自动：
- `ServiceManager.Reload()` 重新加载服务路由缓存。
- `http_proxy_pipeline.InvalidateService(serviceID)` 失效该服务 Plan 缓存。

## 3. 功能与文件对照

### 3.1 启动与初始化

- `main.go`
  - 程序入口，按 `-endpoint` 选择 `server` 或 `dashboard`
- `ai_gateway/bootstrap.go`
  - AI 配置加载、Redis 客户端初始化、Consumer 预热、Metrics 启动
- `ai_gateway/init.go`
  - 全局组件初始化（quota/prompt/ip restriction 等）
- `ai_gateway/config/config.go`
  - AI 配置结构体定义
- `ai_gateway/config/manager.go`
  - AI 配置加载管理（读取 `conf/dev/ai.toml`）

### 3.2 认证与身份

- `ai_gateway/auth/key_auth.go`
  - Key 认证核心逻辑
- `ai_gateway/auth/jwt_auth.go`
  - JWT 验证逻辑
- `ai_gateway/jwt/validator.go`
  - JWT 校验器（算法/签名校验）
- `http_proxy_middleware/ai_auth_middleware.go`
  - 统一鉴权中间件（JWT 优先，Key 回退）
- `http_proxy_middleware/ai_key_auth_middleware.go`
  - Key 认证实现（保留，当前主链路由统一鉴权接管）
- `http_proxy_middleware/ai_jwt_auth_middleware.go`
  - JWT 认证实现（保留，当前主链路由统一鉴权接管）

### 3.3 Token 解析

- `ai_gateway/token/parser.go`
  - 请求体 token 估算与解析
- `ai_gateway/token/stream_parser.go`
  - 流式响应 token 解析
- `http_proxy_middleware/ai_middleware_helpers.go`
  - 请求体读写、上下文字段辅助

### 3.4 模型路由与映射

- `ai_gateway/model/router.go`
  - 模型路由规则匹配
- `ai_gateway/model/mapper.go`
  - 模型名映射
- `ai_gateway/model/middleware.go`
  - 模型处理中间件能力
- `http_proxy_middleware/ai_model_router_middleware.go`
  - 代理链路中的模型路由/映射入口

### 3.5 限流与配额

- `ai_gateway/ratelimit/token_limiter.go`
  - Token 限流核心逻辑
- `ai_gateway/ratelimit/redis_script.go`
  - Redis Lua 脚本与窗口计数
- `http_proxy_middleware/ai_token_ratelimit_middleware.go`
  - 限流中间件
- `ai_gateway/quota/manager.go`
  - 配额管理（Redis + DB）
- `ai_gateway/quota/middleware.go`
  - 配额中间件
- `http_proxy_middleware/ai_quota_middleware.go`
  - 代理链路配额入口

### 3.6 缓存与负载均衡

- `ai_gateway/cache/string_cache.go`
  - 字符串缓存实现
- `ai_gateway/cache/cache_writer.go`
  - 响应写入缓存逻辑
- `http_proxy_middleware/ai_cache_middleware.go`
  - 缓存中间件
- `ai_gateway/loadbalancer/global_least_request.go`
  - 最少请求负载均衡器
- `http_proxy_middleware/ai_loadbalancer_middleware.go`
  - 负载均衡中间件

### 3.7 可观测与日志

- `ai_gateway/observability/metrics.go`
  - Prometheus 指标定义与输出
- `ai_gateway/observability/logger.go`
  - 结构化日志
- `ai_gateway/observability/middleware.go`
  - 请求耗时/状态码/模型/token 指标采集
- `http_proxy_middleware/ai_observability_middleware.go`
  - 可观测中间件接入

### 3.8 Prompt 装饰、IP 限制、CORS

- `ai_gateway/prompt/decorator.go`
  - Prompt 规则装饰与改写
- `http_proxy_middleware/ai_prompt_middleware.go`
  - Prompt 中间件
- `ai_gateway/security/ip_restriction.go`
  - IP/CIDR 黑白名单判断
- `http_proxy_middleware/ai_ip_restriction_middleware.go`
  - IP 限制中间件
- `http_proxy_middleware/ai_cors_middleware.go`
  - CORS 预检与响应头处理

### 3.9 管理接口（Admin）

路由注册：`router/route.go`

- Consumer 管理：`controller/ai_consumer.go`
  - `GET /admin/ai/consumer/list`
  - `POST /admin/ai/consumer/add`
  - `PUT /admin/ai/consumer/update/:id`
  - `DELETE /admin/ai/consumer/delete/:id`
  - `POST /admin/ai/consumer/reload`
- Quota 管理：`controller/ai_quota.go`
  - `GET /admin/ai/quota?consumer_name=...`
  - `POST /admin/ai/quota/refresh`
  - `POST /admin/ai/quota/delta`
- AI Service 配置：`controller/ai_service_config.go`
  - `GET /admin/ai/service-config/list`
  - `GET /admin/ai/service-config/detail?service_id=...`
  - `POST /admin/ai/service-config/upsert`
  - `DELETE /admin/ai/service-config/delete/:service_id`
  - `POST /admin/ai/service-config/reload`
- Pipeline 管理：`controller/pipeline.go`
  - `GET /admin/pipeline/plugins`
  - `GET /admin/pipeline/plan?service_id=...`（或 `service_name=...`）
  - `GET /admin/pipeline/cache`
  - `POST /admin/pipeline/invalidate`（body/query 可带 `service_id`，不带则全量失效）

说明：`/admin/*` 默认有 session 鉴权中间件，调用前需先登录后台。

### 3.10 数据模型与表结构

- DAO：
  - `dao/ai_consumer.go` -> `ai_consumer`
  - `dao/ai_quota.go` -> `ai_quota`
  - `dao/ai_service_config.go` -> `ai_service_config`
- SQL 初始化：`sql/ai_tables.sql`

## 4. 常用测试命令

```powershell
go test ./ai_gateway/...
go test ./http_proxy_middleware
go test ./http_proxy_router
go test ./controller ./router ./dao
```

## 5. 常见问题

### 5.1 启动报错 `open ai: The system cannot find the file specified`

原因：AI 配置加载路径错误。

现版本已修复为从 `lib.GetConfPath("ai")` 读取（即 `./conf/dev/ai.toml`）。如果仍报错，请确认：

- 启动参数为 `-config ./conf/dev/`
- `conf/dev/ai.toml` 文件存在
- 当前工作目录是项目根目录




# AIGateway

一个面向 AI 场景的 Go 网关项目，提供“认证、策略、改写、缓存、观测、转发”一体化能力。  
当前主链路已完成插件化重构：由 `PipelinePlan + Executor` 动态编排执行，支持按服务生效与热更新。

---

## Why AIGateway

相比传统固定中间件链路，AIGateway 的目标是：

- 按服务动态编排：同一网关内，不同服务可跑不同插件链
- 插件原生化：能力以 Plugin 组织，新增能力不改路由主链
- 可观测可回滚：提供计划查看、缓存查看、失效接口
- 面向 AI 代理：内置模型路由、Prompt 装饰、Token 限流、Quota、缓存、AI 负载均衡等

---

## Core Features

- 动态执行链（Phase + Plan + Executor）
- 统一认证（JWT / API Key）
- AI 策略能力
- `ai.model_router`
- `ai.prompt_decorator`
- `ai.token_ratelimit`
- `ai.quota`
- `ai.cache`
- `ai.load_balancer`
- `ai.observability`
- 代理能力
- `proxy.header_transfer`
- `proxy.strip_uri`
- `proxy.url_rewrite`
- `proxy.reverse_proxy`
- 运行时管理能力
- `/admin/pipeline/plugins`
- `/admin/pipeline/plan`
- `/admin/pipeline/cache`
- `/admin/pipeline/invalidate`

---

## Architecture

### Request Flow

```mermaid
flowchart LR
    A[Client Request] --> B[HTTPAccessModeMiddleware]
    B --> C[PipelinePlanMiddleware]
    C --> D[PipelineExecutorMiddleware]
    D --> E[Phase: preflight]
    E --> F[Phase: edge_guard]
    F --> G[Phase: authn/policy]
    G --> H[Phase: transform]
    H --> I[Phase: traffic/observe]
    I --> J[Phase: proxy]
    J --> K[Upstream Service]
```

### Runtime Layers

- `http_proxy_router`：server 入口路由（只保留主链入口）
- `http_proxy_pipeline`：Plan 编译、缓存、执行器调度
- `http_proxy_plugin`：插件契约、注册表、原生插件实现
- `ai_gateway/*`：AI 域能力模块
- `dao` / `controller`：配置与管理接口

---

## Repository Layout

```text
.
├── main.go
├── http_proxy_router/      # 代理入口
├── http_proxy_pipeline/    # 计划编排与执行器
├── http_proxy_plugin/      # 插件运行时与插件实现
├── http_proxy_middleware/  # 历史中间件（兼容/参考）
├── ai_gateway/             # AI 子模块
├── controller/             # 管理端控制器
├── dao/                    # 数据访问与运行时模型
├── router/                 # dashboard 路由
├── conf/dev/               # 本地配置模板（*.example.toml）
└── sql/                    # 表结构脚本
```

---

## Quick Start

### 1. Prerequisites

- Go 1.24+
- MySQL 8+
- Redis 6+

### 2. 配置

仓库默认提交的是脱敏模板配置，请先复制为本地配置：

```powershell
Copy-Item .\conf\dev\base.example.toml .\conf\dev\base.toml
Copy-Item .\conf\dev\proxy.example.toml .\conf\dev\proxy.toml
Copy-Item .\conf\dev\mysql_map.example.toml .\conf\dev\mysql_map.toml
Copy-Item .\conf\dev\redis_map.example.toml .\conf\dev\redis_map.toml
Copy-Item .\conf\dev\redis.example.toml .\conf\dev\redis.toml
Copy-Item .\conf\dev\ai.example.toml .\conf\dev\ai.toml
```

再按你的环境修改：MySQL/Redis 地址、账号、密码、端口等。

### 3. 初始化数据库

```powershell
mysql -uroot -p gateway < .\sql\ai_tables.sql
```

### 4. 启动服务

启动代理服务（server）：

```powershell
go run main.go -endpoint server -config ./conf/dev/
```

启动管理服务（dashboard）：

```powershell
go run main.go -endpoint dashboard -config ./conf/dev/
```

### 5. 健康检查

- Proxy: `GET http://127.0.0.1:8080/ping`
- Dashboard: `GET http://127.0.0.1:8880/ping`
- Swagger: `http://127.0.0.1:8880/swagger/index.html`

### 6. 项目启动教学（手把手）

如果你是第一次跑这个项目，建议按下面顺序执行：

1. 安装依赖并确认版本

```powershell
go version
mysql --version
redis-server --version
```

2. 启动 MySQL 和 Redis（或使用你本地已运行实例）

```powershell
# 示例：本地直接启动 Redis（如已在服务中运行可跳过）
redis-server
```

3. 复制配置模板并填写真实连接信息

```powershell
Copy-Item .\conf\dev\base.example.toml .\conf\dev\base.toml
Copy-Item .\conf\dev\proxy.example.toml .\conf\dev\proxy.toml
Copy-Item .\conf\dev\mysql_map.example.toml .\conf\dev\mysql_map.toml
Copy-Item .\conf\dev\redis_map.example.toml .\conf\dev\redis_map.toml
Copy-Item .\conf\dev\redis.example.toml .\conf\dev\redis.toml
Copy-Item .\conf\dev\ai.example.toml .\conf\dev\ai.toml
```

4. 初始化数据库

```powershell
mysql -uroot -p -e "CREATE DATABASE IF NOT EXISTS gateway DEFAULT CHARSET utf8mb4;"
mysql -uroot -p gateway < .\sql\ai_tables.sql
```

5. 启动代理服务（窗口 A）

```powershell
go run main.go -endpoint server -config ./conf/dev/
```

6. 启动管理服务（窗口 B）

```powershell
go run main.go -endpoint dashboard -config ./conf/dev/
```

7. 验证服务是否启动成功

```powershell
curl http://127.0.0.1:8080/ping
curl http://127.0.0.1:8880/ping
```

如果两个接口都返回 `pong`（或 200），说明项目已可用。

### 7. 常见启动问题

- 报错 `open ai: The system cannot find the file specified`
- 检查是否使用了 `-config ./conf/dev/`，并确认 `conf/dev/ai.toml` 存在。
- 报错 MySQL 连接失败
- 检查 `conf/dev/mysql_map.toml` 的地址、账号、密码和数据库名是否正确。
- 报错 Redis 连接失败
- 检查 `conf/dev/redis_map.toml` 或 `conf/dev/redis.toml` 的地址和密码配置。
- 端口被占用（8080/8880）
- 修改 `conf/dev/proxy.toml` 或 `conf/dev/base.toml` 后重启服务。

---

## Pipeline Admin APIs

- `GET /admin/pipeline/plugins`
- `GET /admin/pipeline/plan?service_id=...` 或 `service_name=...`
- `GET /admin/pipeline/cache`
- `POST /admin/pipeline/invalidate`

说明：`/admin/*` 默认受后台会话鉴权保护。

---

## Performance Design

当前已落地的核心性能策略：

- Plan 缓存：按 `service_id + config_version` 缓存计划
- 并发去重：同 key 的并发 miss 只编译一次（singleflight 思路）
- 运行时配置缓存：`ai_service_config` 走内存缓存，避免请求期 DB 查询
- 失效机制：服务配置更新后按服务失效计划缓存并刷新运行时配置

---

## Testing

常用测试命令：

```powershell
go test ./http_proxy_pipeline ./http_proxy_plugin ./http_proxy_middleware
go test ./ai_gateway/... ./http_proxy_pipeline ./http_proxy_middleware ./http_proxy_router ./controller ./router ./dao
```

---

## Roadmap

- 完成 `ai.cors` 从 adapter 到 native plugin 迁移
- 增强插件参数化能力（按服务/按环境）
- 增强观测指标（plan cache hit/miss、build latency）
- 补齐更多 e2e 与流式场景集成测试

---

## Contributing

欢迎 Issue / PR。建议提交前：

- 保持变更最小化、可回溯
- 为新增插件补充单测（成功路径 + 失败路径）
- 跑通关键测试命令

---

## Security Notes

- 不要提交真实密钥、密码、Token、连接串
- 使用 `conf/dev/*.example.toml` 作为模板
- 生产环境请使用环境变量或密钥管理系统托管敏感信息

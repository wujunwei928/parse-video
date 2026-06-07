---
knowledge_version: 1
last_scanned_at: 2026-06-08
source_commit: ac2a71f
---

# 全局索引

## 按任务读取

| 任务 | 必读文档 | 可选文档 | 注意事项 |
|---|---|---|---|
| 新增平台解析器 | 01_product.md、02_project_map.md、04_code_rules.md | 03_core_flows.md | 先确认平台域名和解析方式 |
| 修改核心流程 | 03_core_flows.md、05_change_safety.md | 04_code_rules.md | 先看影响范围 |
| 新增/修改 API | 02_project_map.md、03_core_flows.md、04_code_rules.md | 06_build_run_deploy.md | 确认路由和响应格式 |
| 修复 Bug | 02_project_map.md、03_core_flows.md、06_build_run_deploy.md | 05_change_safety.md | 先复现问题 |
| 修改中间件 | 03_core_flows.md、05_change_safety.md | 04_code_rules.md | 高风险 |
| 修改配置 | 02_project_map.md、05_change_safety.md、06_build_run_deploy.md | — | 注意环境变量 |
| 部署上线 | 06_build_run_deploy.md | 05_change_safety.md | 不要猜命令 |

## 修改什么，先看哪里

| 想改的内容 | 先看 | 再看 |
|---|---|---|
| 新增平台 | 02_project_map.md | 04_code_rules.md |
| API 接口 | 02_project_map.md | 03_core_flows.md |
| 解析逻辑 | 03_core_flows.md | 05_change_safety.md |
| 中间件 | 03_core_flows.md | 05_change_safety.md |
| 部署 | 06_build_run_deploy.md | 05_change_safety.md |
| 响应格式 | 03_core_flows.md | 04_code_rules.md |

## 按模块读取

| 模块 | 相关文档 | 关键路径 |
|---|---|---|
| parser（解析器） | 02_project_map.md、03_core_flows.md、04_code_rules.md | `parser/vars.go`、`parser/parser.go`、`parser/*.go` |
| cmd（CLI + HTTP） | 02_project_map.md、03_core_flows.md、04_code_rules.md | `cmd/serve.go`、`cmd/handlers.go`、`cmd/middleware.go` |
| utils（工具函数） | 02_project_map.md | `utils/utils.go` |
| 中间件 | 03_core_flows.md、05_change_safety.md | `cmd/middleware.go` |
| 部署 | 06_build_run_deploy.md | `Dockerfile`、`.github/workflows/` |

## 核心代码索引

| 文件/函数 | 职责 | 相关流程 | 修改风险 |
|---|---|---|---|
| `parser/vars.go:videoSourceInfoMapping` | 27 个平台映射表 | 所有解析 | 高 |
| `parser/parser.go:ParseVideoShareUrl` | URL 域名匹配→平台路由 | 分享链接解析 | 高 |
| `parser/parser.go:ParseVideoId` | 平台+ID→解析路由 | ID 解析 | 高 |
| `parser/parser.go:BatchParseVideoId` | 批量并发解析 | 批量解析 | 中 |
| `utils/utils.go:RegexpMatchUrlFromString` | 正则提取 URL | 所有解析入口 | 高 |
| `cmd/handlers.go:v1ParseURLHandler` | v1 API 分享链接解析 | HTTP API | 中 |
| `cmd/handlers.go:v1ParseIDHandler` | v1 API ID 解析 | HTTP API | 中 |
| `cmd/middleware.go:basicAuthMiddleware` | Basic Auth 认证 | HTTP 安全 | 中 |
| `cmd/middleware.go:rateLimitMiddleware` | IP 速率限制 | HTTP 可用性 | 中 |
| `cmd/response.go:sendSuccess/sendError` | 统一响应格式 | HTTP API | 中 |
| `cmd/download.go:downloadMedia` | 媒体文件下载 | CLI 下载 | 低 |
| `cmd/serve.go:runServe` | HTTP 服务启动和路由注册 | 服务启动 | 中 |

## 配置索引

| 配置项 | 用途 | 使用位置 | 修改风险 |
|---|---|---|---|
| `PARSE_VIDEO_USERNAME` | Basic Auth 用户名 | `cmd/middleware.go:basicAuthMiddleware` | 中（影响 API 访问） |
| `PARSE_VIDEO_PASSWORD` | Basic Auth 密码 | `cmd/middleware.go:basicAuthMiddleware` | 中（影响 API 访问） |
| `RATE_LIMIT_RPM` | 每分钟每 IP 限流 | `cmd/middleware.go:newIPRateLimiter` | 低（重启生效） |
| `CORS_ORIGINS` | CORS 允许来源 | `cmd/middleware.go:corsMiddleware` | 低（重启生效） |

## 未确认事项总表

- 未确认是否有外部用户在进行大规模 API 调用。
- 未确认各平台解析器的实际可用性（依赖平台接口稳定性）。
- 未确认 `golang.org/x/net` 在哪些解析器中使用。
- 未确认各平台解析器内部 HTTP 请求的超时和重试策略。

---
knowledge_version: 1
last_scanned_at: 2026-06-08
source_commit: ac2a71f
---

# 项目地图

## 技术栈

| 类型 | 技术/框架 | 版本 | 证据来源 |
|---|---|---|---|
| 编程语言 | Go | 1.24.0 | `go.mod:go` |
| Web 框架 | Gin | v1.11.0 | `go.mod:require` |
| HTTP 客户端 | Resty | v2.16.5 | `go.mod:require` |
| JSON 解析 | gjson | v1.18.0 | `go.mod:require` |
| HTML 解析 | goquery | v1.10.3 | `go.mod:require` |
| CLI 框架 | Cobra | v1.9.1 | `go.mod:require` |
| 速率限制 | golang.org/x/time | v0.6.0 | `go.mod:require` |
| HTML 模板 | Go embed + html/template | 标准库 | `main.go://go:embed` |
| 容器 | Docker (scratch) | — | `Dockerfile` |
| CI | GitHub Actions | — | `.github/workflows/` |

## 目录结构

| 路径 | 作用 | 是否核心 | 证据来源 |
|---|---|---|---|
| `main.go` | 程序入口，嵌入模板与静态资源（`//go:embed templates/* all:static`） | 核心 | `main.go` |
| `cmd/` | Cobra CLI + Gin HTTP 服务 | 核心 | `cmd/*.go` |
| `parser/` | 平台解析器 + 路由映射 | 核心 | `parser/*.go` |
| `utils/` | URL 提取工具函数 | 核心 | `utils/utils.go` |
| `templates/` | Web UI HTML 模板（骨架，引用外部 CSS/JS） | 辅助 | `templates/index.html` |
| `static/` | Web UI 静态资源（7 CSS 主题 + theme/parse/download JS + favicon），经 embed 提供于 `/static` | 辅助 | `static/css/*`、`static/js/*` |
| `api/` | OpenAPI 规范文件 | 辅助 | `api/openapi.yaml` |
| `resources/` | 静态资源（海报等） | 辅助 | `resources/` |
| `.github/workflows/` | CI/CD 配置 | 辅助 | `.github/workflows/*.yml` |
| `docs/` | 文档目录 | 辅助 | `docs/agents/`、`docs/superpowers/` |

## 启动入口

| 类型 | 入口位置 | 启动方式 | 证据来源 |
|---|---|---|---|
| HTTP 服务 | `cmd/serve.go:runServe` | `go run main.go` 或 `go run main.go serve --port 8080` | `cmd/root.go:18-20`（默认子命令 serve） |
| CLI 解析 | `cmd/parse.go:parseCmd` | `go run main.go parse "链接"` | `cmd/parse.go` |
| CLI ID 解析 | `cmd/id.go:idCmd` | `go run main.go id --source douyin "ID"` | `cmd/id.go` |
| 版本信息 | `cmd/version.go:versionCmd` | `go run main.go version` | `cmd/version.go` |

## 核心模块

| 模块 | 路径 | 职责 | 被谁调用 | 修改风险 | 证据来源 |
|---|---|---|---|---|---|
| 解析路由 | `parser/parser.go` | URL 域名匹配 → 平台分发 | `cmd/handlers.go`、`cmd/parse.go` | 高 | `parser/parser.go` |
| 平台映射表 | `parser/vars.go` | 定义 27 个平台常量、接口、数据结构 | 所有解析器 | 高 | `parser/vars.go` |
| URL 提取 | `utils/utils.go` | 正则提取字符串中的 URL | `parser/parser.go` | 中 | `utils/utils.go:RegexpMatchUrlFromString` |
| HTTP 中间件 | `cmd/middleware.go` | Recovery/CORS/日志/限流/BasicAuth | `cmd/serve.go` | 中 | `cmd/serve.go:43-47` |
| API Handler | `cmd/handlers.go` | v1 + legacy API 路由处理 | `cmd/serve.go` | 中 | `cmd/handlers.go` |
| 统一响应 | `cmd/response.go` | API 响应格式（success/error） | `cmd/handlers.go`、`cmd/middleware.go` | 中 | `cmd/response.go` |
| 媒体下载 | `cmd/download.go` | 视频/图集/封面/音乐文件下载 | `cmd/parse.go`、`cmd/id.go` | 低 | `cmd/download.go` |
| 输出格式化 | `cmd/output.go` | text/JSON 输出 | `cmd/parse.go`、`cmd/id.go` | 低 | `cmd/output.go` |

## 外部依赖

| 依赖 | 用途 | 配置位置 | 调用位置 | 证据来源 |
|---|---|---|---|---|
| gin | HTTP 路由和中间件 | `go.mod` | `cmd/serve.go`、`cmd/handlers.go`、`cmd/middleware.go` | `go.mod:require` |
| resty | 请求平台接口 | `go.mod` | 所有 `parser/*.go` 解析器、`cmd/download.go` | `go.mod:require` |
| gjson | 从 JSON 响应中提取字段 | `go.mod` | 多个解析器（douyin、kuaishou 等） | `go.mod:require` |
| goquery | 解析 HTML 页面 | `go.mod` | 部分解析器（如 `parser/redbook.go`） | `go.mod:require` |
| cobra | CLI 子命令管理 | `go.mod` | `cmd/root.go`、`cmd/serve.go`、`cmd/parse.go`、`cmd/id.go` | `go.mod:require` |
| golang.org/x/time | IP 速率限制令牌桶 | `go.mod` | `cmd/middleware.go:newIPRateLimiter` | `go.mod:require` |

## 数据存储

当前项目未发现数据库相关代码。解析服务为无状态设计，不持久化数据。
已检查路径：`go.mod`（无 ORM/驱动依赖）、`parser/`、`cmd/`、`utils/`。

## 未确认事项

- 未确认 `golang.org/x/net` 在哪些解析器中使用（可能是 goquery 的间接依赖）。

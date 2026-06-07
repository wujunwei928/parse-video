---
knowledge_version: 1
last_scanned_at: 2026-06-08
source_commit: ac2a71f
confidence: high
---

# 产品上下文

## 项目一句话定位

Go 语言实现的短视频去水印解析工具，支持 20+ 中国社交平台，提供 CLI、HTTP API 和 Go 库三种使用方式。

证据来源：`README.md`、`go.mod:module`

## 目标用户

| 用户类型 | 痛点 | 使用场景 | 证据来源 |
|---|---|---|---|
| 开发者 | 需要批量解析短视频链接 | 集成到自己的应用中，调用 HTTP API | `README.md:安装` |
| 终端用户 | 需要去水印下载视频 | 使用 CLI 命令或 Web UI 下载 | `cmd/parse.go`、`templates/index.html` |
| Docker 用户 | 需要自建解析服务 | 部署 Docker 容器对外提供 API | `Dockerfile`、`README.md:Docker` |

## 核心功能

| 功能 | 用户价值 | 当前实现状态 | 入口/路径 | 优先级 | 证据来源 |
|---|---|---|---|---|---|
| 分享链接解析 | 一键去水印获取视频 | 已实现 | `parser.ParseVideoShareUrl` | P0 | `parser/parser.go:ParseVideoShareUrl` |
| 视频 ID 解析 | 通过平台+ID 直接解析 | 已实现（部分平台不支持） | `parser.ParseVideoId` | P0 | `parser/parser.go:ParseVideoId` |
| HTTP API | 提供网络服务接口 | 已实现 | `/api/v1/parse`、`/api/v1/parse/:source/:video_id` | P0 | `cmd/handlers.go:v1ParseURLHandler` |
| CLI 工具 | 本地命令行解析 | 已实现 | `go run main.go parse/id` | P1 | `cmd/parse.go`、`cmd/id.go` |
| Web UI | 可视化操作界面 | 已实现 | `GET /` | P1 | `cmd/serve.go:49-61` |
| 媒体下载 | 解析后直接下载 | 已实现 | `--download` 标志 | P2 | `cmd/download.go:downloadMedia` |
| 批量解析 | 多链接并发处理 | 已实现 | `parse` 命令多参数或 `--file` | P2 | `cmd/parse.go:runBatchParse` |
| Basic Auth | API 访问控制 | 已实现（可选） | 环境变量配置 | P2 | `cmd/middleware.go:basicAuthMiddleware` |
| 速率限制 | 防滥用 | 已实现（默认 60 RPM/IP） | 环境变量配置 | P2 | `cmd/middleware.go:rateLimitMiddleware` |

## 商业模式

当前项目为开源工具（无收费功能）。未发现付费、订阅、积分等商业化实现。
证据来源：全代码库搜索未发现 pay/subscription/billing 等关键词。

## MVP 边界

**做什么**：
- 解析中国主流社交平台的短视频分享链接，提取去水印视频地址
- 提供多平台图集解析（抖音、快手、小红书、皮皮虾、微博）
- 提供小红书 LivePhoto 解析
- HTTP API + CLI + Go 库三种使用形态

**不做什么**：
- 不做视频下载托管（仅返回直链）
- 不做用户系统（无数据库、无状态）
- 不做视频编辑/转码
- 不做国际化（仅中文平台，代码注释中文）

推断依据：代码库无数据库、无用户系统、无文件存储服务。

## 产品原则

- 无状态设计：解析服务不持久化任何数据，每次请求独立处理。
- 移动端优先：默认 User-Agent 为 iPhone Safari，适配移动端分享链接。
- 向后兼容：保留旧版 API 路由（`/video/share/url/parse`），不破坏已有集成。

## 未确认事项

- 未确认是否有外部用户在使用 HTTP API 进行大规模调用。
- 未确认各平台解析器的实际可用性（依赖平台接口稳定性）。

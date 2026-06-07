---
knowledge_version: 1
last_scanned_at: 2026-06-08
source_commit: ac2a71f
---

# 编码规则

## 通用原则

- 不做无需求的大重构。
- 不引入不必要的新依赖。
- 优先复用现有组件、工具函数。
- 新增代码必须放在已有目录体系内。
- 禁止硬编码密钥、平台接口地址。

## 目录放置规则

| 代码类型 | 应放位置 | 参考文件 | 禁止放置 |
|---|---|---|---|
| 平台解析器 | `parser/<平台名>.go` | `parser/douyin.go` | `cmd/`、根目录 |
| CLI 子命令 | `cmd/<命令名>.go` | `cmd/parse.go` | `parser/`、根目录 |
| 中间件 | `cmd/middleware.go` | `cmd/middleware.go` | 独立 `middleware/` 目录 |
| API Handler | `cmd/handlers.go` | `cmd/handlers.go` | 独立 `handlers/` 目录 |
| 工具函数 | `utils/utils.go` | `utils/utils.go` | `parser/`、`cmd/` |
| Web UI 模板 | `templates/*.html` | `templates/index.html` | `cmd/`、根目录 |
| API 规范 | `api/openapi.yaml` | `api/openapi.yaml` | `cmd/` |

## 命名规则

- **平台常量**：`Source` + 驼峰平台名，如 `SourceDouYin`、`SourceKuaiShou`（`parser/vars.go`）
- **解析器结构体**：小写驼峰平台名，如 `douYin{}`、`kuaiShou{}`（`parser/vars.go`）
- **文件命名**：与平台 source 一致，如 `douyin.go`、`kuaishou.go`
- **CLI 标志**：短横线分隔，如 `--output-dir`、`--source`
- **导出函数**：大写开头驼峰，如 `ParseVideoShareUrl`
- **内部函数**：小写开头驼峰，如 `parseShareUrl`

## 错误处理规则

- 解析器返回 `error` 接口（无类型化错误码）。
- HTTP API 层统一通过 `classifyParseError` 将所有解析错误归类为 422 PARSE_FAILED。
- 使用 `fmt.Errorf` + `%w` 包装错误，保留原始错误链。
- panic 由 Recovery 中间件统一捕获，返回 500。

证据来源：`cmd/response.go:classifyParseError`、`cmd/middleware.go:recoveryMiddleware`

## 日志规则

- 使用标准库 `log`（`log.New(os.Stderr, ...)`）。
- 请求日志格式：`方法 路径 状态码 耗时`。
- 无结构化日志、无链路 ID。
- 下载进度输出到 `os.Stderr`。

证据来源：`cmd/middleware.go:requestLogMiddlewareWithWriter`

## 配置读取规则

- 所有配置通过环境变量读取，使用 `os.Getenv`。
- 禁止硬编码端口、密码、限流阈值。
- 默认值在代码中提供（`getEnvDefault`、`getEnvInt`）。

证据来源：`cmd/serve.go:getEnvDefault`、`cmd/serve.go:getEnvInt`

## 前端规则

当前项目前端仅有 `templates/index.html` 一个 Web UI 页面，通过 Go `html/template` 渲染。不使用前端框架。

## 后端规则

- **Handler**：直接在 `cmd/handlers.go` 中定义，使用 `gin.Context`。
- **Service**：无独立 Service 层，Handler 直接调用 `parser` 包函数。
- **解析器**：每个平台实现 `videoShareUrlParser` 和/或 `videoIdParser` 接口。
- **响应格式**：统一使用 `sendSuccess`/`sendError`（`cmd/response.go`）。

## 数据库规则

当前项目未发现数据库相关代码。解析服务为无状态设计。

## AI 调用规则

当前项目未发现 AI 调用相关代码。

## 测试规则

- 测试框架：Go 标准 `testing` 包。
- 测试文件放在源码同目录（`*_test.go`）。
- 运行命令：`go test ./...` 或 `go test -v ./...`。
- Pre-commit hook 包含 `go-unit-tests`。
- 测试命名风格：`TestIntegrationV1ParseURLSuccess` 等描述性名称。

证据来源：`parser/douyin_test.go`、`cmd/handlers_test.go`、`.pre-commit-config.yaml`

## 事实来源

- `cmd/handlers.go`：API 路由和响应格式
- `cmd/middleware.go`：中间件链和错误处理
- `cmd/response.go`：统一响应格式
- `parser/vars.go`：接口定义和命名规范
- `parser/parser.go`：解析路由逻辑
- `utils/utils.go`：URL 提取工具
- `.pre-commit-config.yaml`：代码质量检查

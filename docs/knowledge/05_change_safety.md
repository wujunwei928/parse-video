---
knowledge_version: 1
last_scanned_at: 2026-06-08
source_commit: ac2a71f
---

# 变更安全规则

## 高风险区域

| 区域 | 路径 | 风险 | 修改前必须检查 | 风险等级 | 证据来源 |
|---|---|---|---|---|---|
| 解析路由 | `parser/parser.go:ParseVideoShareUrl` | 域名匹配逻辑影响所有平台 | 域名字符串匹配是否遗漏 | 高 | `parser/parser.go:26-36` |
| 平台映射表 | `parser/vars.go:videoSourceInfoMapping` | 新增/修改平台影响全局 | 常量名、域名列表、接口实现 | 高 | `parser/vars.go:87-234` |
| URL 提取正则 | `utils/utils.go:RegexpMatchUrlFromString` | 正则变化影响所有解析入口 | 测试各种 URL 格式 | 高 | `utils/utils.go:9` |
| API 路由 | `cmd/handlers.go` + `cmd/serve.go` | 路由变更影响所有 API 消费者 | 旧路由兼容性 | 中 | `cmd/serve.go:64-74` |
| 中间件栈 | `cmd/middleware.go` | 顺序错误导致安全/性能问题 | 中间件执行顺序 | 中 | `cmd/serve.go:43-47` |
| Basic Auth | `cmd/middleware.go:basicAuthMiddleware` | 认证绕过风险 | 豁免路径列表 | 中 | `cmd/middleware.go:197-217` |
| 速率限制 | `cmd/middleware.go:rateLimitMiddleware` | 限流过严/过松 | RPM 默认值、清理周期 | 中 | `cmd/middleware.go:158-176` |
| 统一响应格式 | `cmd/response.go` | 错误码变化破坏客户端 | 错误码常量名 | 中 | `cmd/response.go:10-17` |

## 禁止事项

- 禁止一次性大范围重构。
- 禁止删除未知用途代码。
- 禁止未确认调用方就改公共函数签名。
- 禁止把密钥写入代码。
- 禁止为了"看起来更优雅"重写稳定模块。
- 禁止修改解析器时不测试实际平台链接。

## Web UI 静态资源注意事项

- **静态资源必须走 `/static/` 前缀**：`rateLimitMiddleware` 已对 `/static/` 前缀豁免限流（`exemptPrefixes` 参数）。`newIPRateLimiter` 写死 `burst=1`，若静态资源不豁免，页面首次加载并发请求多个 CSS/JS 会从第二个起全部 429，导致样式/脚本加载失败、`handleDownloadClick is not defined` 等连锁错误。新增静态资源路径务必保持 `/static/` 前缀；限流本意是保护解析 API，静态资源是纯文件服务无计算成本。
- **开发态警惕 `go run` 的 embed 缓存**：修改 `static/` 或 `templates/` 内容后，`go run` 可能复用旧 embed（实测会出现新静态资源 404）。开发验证改用 `go build -o /tmp/pv . && /tmp/pv serve` 或 `go run -a`。生产 Docker 构建每次全新编译，不受影响。

## 修改前检查清单

- 这个函数被谁调用？（可用 `grep` 或 `LSP: findReferences`）
- 是否影响 API 响应格式？
- 是否影响中间件行为？
- 是否影响平台解析结果？
- 是否需要新增或修改测试？
- 是否需要更新 `api/openapi.yaml`？
- 是否需要更新知识库？

## 变更影响地图

| 修改内容 | 可能影响 | 必须测试 | 必须同步更新的文档 |
|---|---|---|---|
| `parser/vars.go` 常量或映射 | 所有平台解析 | 受影响平台的解析 | `02_project_map.md`、`99_global_index.md` |
| `parser/parser.go` 路由逻辑 | 所有解析入口 | 分享链接解析 + ID 解析 | `03_core_flows.md`、`05_change_safety.md` |
| `utils/utils.go` 正则 | URL 提取 | 各种 URL 格式 | `03_core_flows.md` |
| `cmd/middleware.go` | 所有 HTTP 请求 | 限流、认证、CORS | `03_core_flows.md`、`05_change_safety.md` |
| `cmd/handlers.go` 路由 | API 消费者 | 接口响应格式 | `02_project_map.md`、`03_core_flows.md`、`99_global_index.md` |
| `cmd/response.go` 错误码 | 客户端错误处理 | 错误响应格式 | `03_core_flows.md` |
| 新增平台解析器 | 仅新平台 | 该平台解析 | `02_project_map.md`、`99_global_index.md` |
| `Dockerfile` | 部署 | 构建和运行 | `06_build_run_deploy.md` |
| 环境变量 | 配置 | 对应功能 | `02_project_map.md`、`06_build_run_deploy.md` |

## 知识库更新规则

| 代码变更 | 必须更新文档 |
|---|---|
| 新增/删除 API | `02_project_map.md`、`03_core_flows.md`、`99_global_index.md` |
| 修改中间件 | `03_core_flows.md`、`05_change_safety.md` |
| 新增/修改平台解析器 | `02_project_map.md`、`99_global_index.md` |
| 修改环境变量 | `02_project_map.md`、`06_build_run_deploy.md`、`05_change_safety.md` |
| 修改部署方式 | `06_build_run_deploy.md`、`05_change_safety.md` |
| 新增核心模块 | `02_project_map.md`、`99_global_index.md` |

## 不需要更新知识库的情况

- 纯 UI 文案微调。
- 无业务含义的样式调整。
- 局部 bugfix 且不改变流程。
- 测试用例补充但不改变规则。
- 注释修正。

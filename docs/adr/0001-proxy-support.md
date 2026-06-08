# 统一代理支持

所有解析器的 HTTP 请求通过 `parser/client.go` 中的 `newClient()` 工厂函数创建 resty client，从包级变量 `proxyURL`（由 `InitProxy()` 设置）读取代理地址并注入。仅支持 HTTP/HTTPS 代理协议，不支持 SOCKS。代理不可用时直接失败，不静默降级为直连。

**为什么用 client 工厂而不是全局环境变量透传：** 项目有 33 个独立的 `resty.New()` 调用点散落在 25 个解析器中，另有 bilibili 使用 `net/http`。Go 的 `HTTP_PROXY` 环境变量会影响同进程内所有 HTTP 流量（包括 Gin 框架自身的请求），副作用不可控。工厂函数提供了可控的注入点，未来加超时、重试等逻辑也有统一入口。

**为什么 `InitProxy` 在 `cmd/` 层调用而非 parser 包内直接读环境变量：** parser 包作为库应保持纯净，不直接依赖 `os.Getenv`。代理配置由调用方（cobra `PersistentPreRunE`）读取环境变量并注入，便于测试（直接传参）和复用（不同调用方可以用不同配置来源）。

**为什么用 `atomic.Value` 而非 `sync.RWMutex`：** `proxyURL` 是一次写入、持续读取的场景。`atomic.Value` 零锁、零等待，比 `RWMutex` 更轻量，同时保证并发安全（`go test -race` 验证通过）。

**为什么不做代理池轮换：** 单代理覆盖大部分场景。需要轮换时，前置一个代理池服务（如 Squid、tinyproxy）比在代码里实现更可靠，也避免引入状态管理。

**为什么不支持 SOCKS：** 视频解析场景中 SOCKS 代理使用率极低，加 `golang.org/x/net/proxy` 依赖增加维护负担。后续有需求可在 `newClient()` 中扩展。

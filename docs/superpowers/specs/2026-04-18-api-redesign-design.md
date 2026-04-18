# API 层重设计方案

## 背景

当前 parse-video 已完成 CLI 子命令化重构（cobra），具备 Go SDK + HTTP API + CLI 三种接口形态。但 HTTP API 层存在以下不足：

- 非 RESTful 风格，路由命名不规范
- 响应始终 HTTP 200，靠 `code` 字段区分成功/失败
- 缺少 CORS、速率限制、健康检查等基础设施
- 无 API 文档
- 无版本化机制

本次重设计的目标是将 API 层提升到现代 Web 服务标准，参考 cobalt 的 API 设计理念。

## 设计原则

- **parser 包零改动**：API 层只调用现有 `parser.ParseVideoShareUrlByRegexp()` 和 `parser.ParseVideoId()`
- **向后兼容**：旧路由保留独立的响应适配器（HTTP 200 + `{code, msg, data}`），不重用 v1 处理函数
- **语义化**：HTTP 状态码、错误码、响应格式统一规范（仅 v1 API）
- **渐进式**：可分步实施，每步都有可验证的产出

## API 路由设计

### 新路由

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/parse?url=<share_url>` | 分享链接解析 |
| GET | `/api/v1/parse/<source>/<video_id>` | 视频 ID 解析 |
| GET | `/api/v1/health` | 健康检查 |
| GET | `/api/v1/platforms` | 支持平台列表 |

### 旧路由（保留兼容）

| 旧路由 | 行为 |
|--------|------|
| `/video/share/url/parse?url=` | 独立适配器，HTTP 200 + `{code, msg, data}` 格式不变 |
| `/video/id/parse?source=&video_id=` | 独立适配器，HTTP 200 + `{code, msg, data}` 格式不变 |
| `GET /` | 保留现有 Web UI |

**兼容性保证**：旧路由保持原有行为——始终 HTTP 200，响应体 `{code: 200, msg, data}` 或 `{code: 201, msg}`。Web UI 硬依赖 `jsonObj.code == 200` 判断逻辑，不可改变。旧路由和新路由共享底层解析调用（`parser` 包），但各自拥有独立的 HTTP 适配器。

### v1 端点详细语义

**`/api/v1/parse?url=<share_url>`**：
- `url` 参数缺失 → 400 `MISSING_PARAMETER`
- `url` 中无法提取有效链接（正则不匹配）→ 400 `UNSUPPORTED_URL`
- `url` 中提取到链接但无法识别对应平台（域名不在映射表中）→ 400 `UNSUPPORTED_URL`
- 解析成功 → 200 + 解析数据
- 平台接口异常 → 422 `PARSE_FAILED`

**`/api/v1/parse/<source>/<video_id>`**：
- `source` 不在合法列表 → 400 `UNSUPPORTED_SOURCE`
- `source` 不支持 ID 解析（如 kuaishou、redbook）→ 400 `ID_PARSE_NOT_SUPPORTED`
- 路径参数缺失（Gin 路由不匹配）→ 404（由 Gin 默认处理，非 v1 错误格式）
- 解析成功 → 200 + 解析数据
- 平台接口异常 → 422 `PARSE_FAILED`
- `source` 或 `video_id` 含特殊字符 → URL 已由 Gin 路由解码，无需额外处理

**`/api/v1/health`**：
- 始终返回 200（服务存活即健康）

**`/api/v1/platforms`**：
- 始终返回 200 + 平台列表（按 `source` 字母序排列）

## 响应格式（v1 API）

### 成功响应

`data` 字段结构对应 `parser.VideoParseInfo`（`parser/vars.go`），所有字段始终存在：

| 字段 | 类型 | 说明 |
|------|------|------|
| `author.uid` | string | 作者 ID，可能为空字符串 |
| `author.name` | string | 作者名称，可能为空字符串 |
| `author.avatar` | string | 作者头像 URL，可能为空字符串 |
| `title` | string | 视频标题 |
| `video_url` | string | 视频播放地址，可能为空字符串（图集内容） |
| `music_url` | string | 音乐地址，可能为空字符串 |
| `cover_url` | string | 封面地址，可能为空字符串 |
| `images` | array | 图集列表，`null` 表示非图集内容；非空时每项含 `url`（图片地址）和 `live_photo_url`（LivePhoto 视频地址，可能为空字符串） |

```json
{
  "status": "success",
  "data": {
    "author": {"uid": "123", "name": "张三", "avatar": "..."},
    "title": "视频标题",
    "video_url": "https://...",
    "music_url": "https://...",
    "cover_url": "https://...",
    "images": null
  }
}
```

### 错误响应

```json
{
  "status": "error",
  "error": {
    "code": "UNSUPPORTED_URL",
    "message": "该链接无法识别对应平台"
  }
}
```

### 健康检查响应（特殊格式）

健康检查使用独立的简化格式，不遵循 `{status, data}` 信封：

```json
{
  "status": "ok",
  "version": "1.0.0",
  "platforms": 23
}
```

### 旧路由响应（不改变）

旧路由始终 HTTP 200，使用原有格式：

```json
// 成功
{"code": 200, "msg": "解析成功", "data": {...}}

// 失败
{"code": 201, "msg": "错误信息"}
```

### HTTP 状态码映射（仅 v1）

| 场景 | HTTP 状态码 | error.code |
|------|------------|-----------|
| 解析成功 | 200 | - |
| URL 参数缺失 | 400 | `MISSING_PARAMETER` |
| 链接无法提取或无法识别平台 | 400 | `UNSUPPORTED_URL` |
| source 不在合法列表 | 400 | `UNSUPPORTED_SOURCE` |
| source 不支持 ID 解析 | 400 | `ID_PARSE_NOT_SUPPORTED` |
| 解析失败（平台接口异常） | 422 | `PARSE_FAILED` |
| 认证失败 | 401 | `UNAUTHORIZED` |
| 速率超限 | 429 | `RATE_LIMITED` |
| 服务器内部错误（panic 等） | 500 | `INTERNAL_ERROR` |

### 错误码常量定义

```go
const (
    ErrMissingParameter    = "MISSING_PARAMETER"
    ErrUnsupportedURL      = "UNSUPPORTED_URL"
    ErrUnsupportedSource   = "UNSUPPORTED_SOURCE"
    ErrIDParseNotSupported = "ID_PARSE_NOT_SUPPORTED"
    ErrParseFailed         = "PARSE_FAILED"
    ErrUnauthorized        = "UNAUTHORIZED"
    ErrRateLimited         = "RATE_LIMITED"
    ErrInternal            = "INTERNAL_ERROR"
)
```

### 错误分类策略

`parser` 包返回 `error` 接口，无类型化错误。API 层采用**两层分类**：

1. **预验证层**（调用 `parser` 之前）：
   - 参数非空检查 → `MISSING_PARAMETER`
   - source 合法性检查 → `UNSUPPORTED_SOURCE`
   - ID 解析支持检查 → `ID_PARSE_NOT_SUPPORTED`
   - URL 提取预验证（调用 `utils.RegexpMatchUrlFromString` 提取 URL，再遍历 `parser.VideoSourceInfoMapping` 匹配域名）→ `UNSUPPORTED_URL`
2. **统一兜底层**：`parser` 返回的 `error` 一律归类为 `PARSE_FAILED`（422），不尝试解析 `error.Error()` 内容
3. **panic 兜底**：`parser` panic 由 recovery 中间件捕获，归类为 `INTERNAL_ERROR`（500）

## 平台列表端点

```json
GET /api/v1/platforms
{
  "status": "success",
  "data": [
    {"source": "acfun", "name": "AcFun", "url_parse": true, "id_parse": true},
    {"source": "bilibili", "name": "哔哩哔哩", "url_parse": true, "id_parse": false},
    {"source": "douyin", "name": "抖音", "url_parse": true, "id_parse": true}
  ]
}
```

**平台元数据**：`name` 字段在 `cmd/handlers.go` 中硬编码维护为有序映射（`source → display name`），按 `source` 字母序排列。`url_parse` 和 `id_parse` 通过检查 `parser.VideoSourceInfoMapping[source]` 中对应 parser 是否为 `nil` 来判断。

## 中间件

### 中间件栈（执行顺序）

```
请求 → Recovery → CORS → 请求日志 → 速率限制 → Basic Auth（可选）→ 路由处理 → 响应
```

### Recovery 中间件

- 捕获 handler 中的 panic，返回 500 `INTERNAL_ERROR`
- 防止单个请求异常导致整个服务崩溃

### CORS 中间件

- 默认 `Access-Control-Allow-Origin: *`，`Access-Control-Allow-Methods: GET, OPTIONS`，`Access-Control-Allow-Headers: Content-Type, Authorization`
- `OPTIONS` 预检请求直接返回 204，不经过后续中间件
- 当启用 Basic Auth 时，CORS 不设置 `Access-Control-Allow-Credentials: true`（自部署场景下用户通过同源访问，不需要跨域携带凭证）
- 可通过 `CORS_ORIGINS` 环境变量配置白名单（逗号分隔，如 `https://a.com,https://b.com`）

### 速率限制中间件

- 基于 IP 地址：直接读取 `r.RemoteAddr`（不含端口部分），**不读取 `X-Forwarded-For` 等可伪造的 header**。部署在反向代理后时，由代理层（Nginx 等）负责限速或将真实 IP 写入 `RemoteAddr`（如 Nginx `set_real_ip_from` + `real_ip_header`）
- 实现方式：内存限速（`golang.org/x/time/rate`），每个 IP 一个 `rate.Limiter` 实例
- **过期清理**：使用 `sync.Map` + 后台 goroutine 每 10 分钟清理超过 30 分钟无活动的 limiter 条目，防止内存无限增长
- 默认 60 次/分钟（`rate.Every(time.Second)`，burst=1），可通过 `RATE_LIMIT_RPM` 环境变量配置
- 超限时返回 429 + `Retry-After` header（值 = `60 / rateLimitRPM`，四舍五入到整秒，最少 1 秒）
- **豁免规则**：`/api/v1/health` 不受限速影响

### Basic Auth 中间件

- 保留现有逻辑：`PARSE_VIDEO_USERNAME` 和 `PARSE_VIDEO_PASSWORD` 环境变量启用
- **认证覆盖范围**（启用时）：
  - **需要认证**：`/api/v1/parse`、`/api/v1/parse/<source>/<id>`、旧路由 `/video/share/url/parse`、`/video/id/parse`
  - **无需认证**：`/api/v1/health`（方便负载均衡器探活）、`/api/v1/platforms`（公开信息）、`GET /`（Web UI 页面本身）
- **Web UI 兼容**：Web UI 通过浏览器 XHR 调用旧路由。当 Basic Auth 启用时，浏览器会弹出认证对话框（HTTP Basic Auth 标准行为），用户输入凭证后浏览器自动附加 `Authorization` header，无需修改 Web UI 代码。这与现有行为一致。

### 请求日志中间件

- 结构化日志：方法、路径、状态码、耗时
- 使用标准库 `log` 输出到 stderr

## 环境变量

| 变量 | 用途 | 默认值 |
|------|------|--------|
| `PARSE_VIDEO_USERNAME` | Basic Auth 用户名 | 空（不启用） |
| `PARSE_VIDEO_PASSWORD` | Basic Auth 密码 | 空（不启用） |
| `RATE_LIMIT_RPM` | 速率限制（次/分钟） | 60 |
| `CORS_ORIGINS` | CORS 允许来源 | `*` |

## 文件结构

### 新增文件

```
cmd/
├── middleware.go        # Recovery、CORS、速率限制、请求日志中间件
├── handlers.go          # API v1 处理函数 + 平台元数据映射 + 旧路由适配器
└── response.go          # v1 统一响应格式、错误码常量、错误分类

api/
└── openapi.yaml         # OpenAPI 3.0 文档
```

### 修改文件

```
cmd/serve.go             # 重写：注册新路由 + 中间件 + 旧路由注册
```

### 不变文件

```
cmd/root.go              # 根命令不变
cmd/parse.go             # CLI parse 子命令不变
cmd/id.go                # CLI id 子命令不变
cmd/output.go            # CLI 输出格式化不变
cmd/version.go           # 版本子命令不变
parser/                  # 解析器包不变
templates/               # Web UI 不变
```

## 实施步骤

1. **`response.go`**：定义 `apiResponse`、`apiError` 类型、错误码常量、错误分类函数
2. **`middleware.go`**：实现 Recovery、CORS、速率限制（含过期清理）、请求日志中间件
3. **`handlers.go`**：实现 v1 处理函数（解析、健康检查、平台列表）+ 旧路由适配器 + 平台元数据映射
4. **重写 `serve.go`**：注册中间件栈 + v1 路由 + 旧路由 + Web UI
5. **`api/openapi.yaml`**：编写 OpenAPI 文档
6. **测试**：API 集成测试覆盖各状态码场景

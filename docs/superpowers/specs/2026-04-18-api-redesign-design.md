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
- **向后兼容**：旧路由保留（标记 deprecated），通过转发到新路由处理逻辑
- **语义化**：HTTP 状态码、错误码、响应格式统一规范
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
| `/video/share/url/parse?url=` | 转发到 `/api/v1/parse?url=` 处理逻辑 |
| `/video/id/parse?source=&video_id=` | 转发到 `/api/v1/parse/<source>/<video_id>` 处理逻辑 |
| `GET /` | 保留现有 Web UI |

## 响应格式

### 成功响应

```json
{
  "status": "success",
  "data": {
    "author": {"uid": "123", "name": "张三", "avatar": "..."},
    "title": "视频标题",
    "video_url": "https://...",
    "music_url": "https://...",
    "cover_url": "https://...",
    "images": [...]
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

### 旧路由兼容响应

旧路由保持原有 `{code, msg, data}` 格式，不改变。

### HTTP 状态码映射

| 场景 | HTTP 状态码 | error.code |
|------|------------|-----------|
| 解析成功 | 200 | - |
| URL 参数缺失 | 400 | `MISSING_PARAMETER` |
| 平台不支持 | 400 | `UNSUPPORTED_URL` |
| 解析失败（平台接口异常） | 422 | `PARSE_FAILED` |
| 认证失败 | 401 | `UNAUTHORIZED` |
| 速率超限 | 429 | `RATE_LIMITED` |
| 服务器内部错误 | 500 | `INTERNAL_ERROR` |

### 错误码常量定义

```go
const (
    ErrMissingParameter = "MISSING_PARAMETER"
    ErrUnsupportedURL   = "UNSUPPORTED_URL"
    ErrParseFailed      = "PARSE_FAILED"
    ErrUnauthorized     = "UNAUTHORIZED"
    ErrRateLimited      = "RATE_LIMITED"
    ErrInternal         = "INTERNAL_ERROR"
)
```

## 健康检查端点

```json
GET /api/v1/health
{
  "status": "ok",
  "version": "1.0.0",
  "platforms": 23
}
```

## 平台列表端点

```json
GET /api/v1/platforms
{
  "status": "success",
  "data": [
    {"source": "douyin", "name": "抖音", "url_parse": true, "id_parse": true},
    {"source": "kuaishou", "name": "快手", "url_parse": true, "id_parse": false}
  ]
}
```

## 中间件

### 中间件栈

```
请求 → CORS → 速率限制 → Basic Auth（可选）→ 路由处理 → 响应
```

### CORS 中间件

- 默认允许所有来源（`*`）
- 可通过 `CORS_ORIGINS` 环境变量配置白名单

### 速率限制中间件

- 基于 IP 地址
- 默认 60 次/分钟
- 可通过 `RATE_LIMIT_RPM` 环境变量配置
- 超限时返回 429 + `Retry-After` header
- 实现方式：内存限速（`golang.org/x/time/rate`），无需外部依赖

### Basic Auth 中间件

- 保留现有逻辑
- `PARSE_VIDEO_USERNAME` 和 `PARSE_VIDEO_PASSWORD` 环境变量启用

### 请求日志中间件

- 结构化日志：方法、路径、状态码、耗时

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
├── middleware.go        # CORS、速率限制、请求日志中间件
├── handlers.go          # API v1 处理函数（解析、健康检查、平台列表）
└── response.go          # 统一响应格式、错误码常量

api/
└── openapi.yaml         # OpenAPI 3.0 文档
```

### 修改文件

```
cmd/serve.go             # 重写：新路由注册 + 旧路由兼容转发
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

1. **`response.go`**：定义 `apiResponse`、`apiError` 类型和错误码常量
2. **`middleware.go`**：实现 CORS、速率限制、请求日志中间件
3. **`handlers.go`**：实现 v1 API 处理函数
4. **重写 `serve.go`**：注册新路由 + 中间件 + 旧路由兼容
5. **`api/openapi.yaml`**：编写 OpenAPI 文档
6. **测试**：API 集成测试覆盖各状态码场景

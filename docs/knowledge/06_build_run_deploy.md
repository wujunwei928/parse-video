---
knowledge_version: 1
last_scanned_at: 2026-06-08
source_commit: ac2a71f
---

# 构建运行部署

## 本地开发环境

| 依赖 | 版本/要求 | 证据来源 |
|---|---|---|
| Go | 1.24.0 | `go.mod:go` |
| Docker（可选） | 任意版本 | `Dockerfile` |
| pre-commit（可选） | v5.0.0 hooks | `.pre-commit-config.yaml` |
| Air（热重载，可选） | 最新版 | `.air.toml` |

## 环境变量

| 变量 | 用途 | 是否敏感 | 默认值/示例 | 使用位置 | 证据来源 |
|---|---|---|---|---|---|
| `PARSE_VIDEO_USERNAME` | Basic Auth 用户名 | 是 | 无（未设置则不开启认证） | `cmd/serve.go:34` | `cmd/middleware.go:basicAuthMiddleware` |
| `PARSE_VIDEO_PASSWORD` | Basic Auth 密码 | 是 | 无（未设置则不开启认证） | `cmd/serve.go:35` | `cmd/middleware.go:basicAuthMiddleware` |
| `RATE_LIMIT_RPM` | 每 IP 每分钟请求上限 | 否 | `60` | `cmd/serve.go:32` | `cmd/middleware.go:newIPRateLimiter` |
| `CORS_ORIGINS` | 允许的跨域来源 | 否 | `*`（允许所有） | `cmd/serve.go:33` | `cmd/middleware.go:corsMiddleware` |

## 安装依赖

```bash
go mod download
```

## 本地启动

```bash
# 默认 8080 端口（cobra 默认子命令为 serve）
go run main.go

# 自定义端口
go run main.go --port 9090

# 热重载（需安装 air）
air
```

## 测试命令

```bash
# 运行所有测试
go test ./...

# 详细输出
go test -v ./...

# 运行特定包测试
go test ./parser/...
go test ./cmd/...

# Pre-commit 检查
pre-commit run --all-files
```

## 构建命令

```bash
# 构建二进制
go build -o parse-video .

# 构建（精简体积）
go build -ldflags="-s -w" -o parse-video .
```

## 部署方式

**Docker 容器部署**（主要方式）：

```bash
# 构建镜像
docker build -t parse-video .

# 运行（默认 8080）
docker run -d -p 8080:8080 parse-video

# 自定义端口
docker run -d -p 9090:9090 parse-video -port 9090

# 开启 Basic Auth
docker run -d -p 8080:8080 \
  -e PARSE_VIDEO_USERNAME=user \
  -e PARSE_VIDEO_PASSWORD=pass \
  parse-video
```

**CI/CD**（GitHub Actions）：
- `.github/workflows/go.yml`：push/PR 到 main 时自动 build + test
- `.github/workflows/docker.yml`：push 到 main 时自动构建多架构 Docker 镜像并推送到 Docker Hub

**Docker 镜像特性**：
- 多阶段构建：`golang:alpine` 编译 → `scratch` 运行
- 支持多架构：`linux/amd64`、`linux/arm/v7`、`linux/arm64/v8`
- 时区设置为 `Asia/Shanghai`

证据来源：`Dockerfile`、`.github/workflows/docker.yml`

## 数据库初始化/迁移

当前项目未使用数据库，无需初始化。

## 常见问题

| 问题 | 可能原因 | 排查方式 | 相关文件 |
|---|---|---|---|
| 端口占用 | 其他进程占用 8080 | `lsof -i :8080` 或换端口 | `cmd/serve.go:82` |
| 解析失败 | 平台接口变更 | 检查对应平台解析器的 HTTP 请求 | `parser/<平台>.go` |
| 401 认证失败 | 环境变量未设置或不匹配 | 确认 `PARSE_VIDEO_USERNAME` 和 `PARSE_VIDEO_PASSWORD` | `cmd/middleware.go:basicAuthMiddleware` |
| 429 限流 | 请求频率超限 | 调整 `RATE_LIMIT_RPM` 或降低请求频率 | `cmd/middleware.go:rateLimitMiddleware` |
| Docker 构建慢 | Go 依赖下载 | 确认 `GOPROXY` 配置 | `Dockerfile:ENV GOPROXY` |

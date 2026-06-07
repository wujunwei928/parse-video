# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based video parsing tool that removes watermarks from videos across 20+ Chinese social media platforms. The project provides a CLI tool, a web API, and a Go library for parsing video share links and extracting clean video URLs.

## Development Commands

### Building and Running
```bash
# Run the web server locally (default port 8080, cobra default subcommand: serve)
go run main.go

# Run with custom port
go run main.go serve --port 9090

# Run with basic auth (requires both environment variables)
export PARSE_VIDEO_USERNAME=your_username
export PARSE_VIDEO_PASSWORD=your_password
go run main.go serve

# CLI: parse a share link
go run main.go parse "分享链接"

# CLI: parse by video ID
go run main.go id --source douyin "视频ID"

# Build the binary
go build -o parse-video .
```

### Testing
```bash
# Run all tests
go test ./...

# Run specific test file
go test ./parser/douyin_test.go

# Run tests with verbose output
go test -v ./...
```

### Docker
```bash
# Build Docker image
docker build -t parse-video .

# Run container (default port 8080)
docker run -d -p 8080:8080 parse-video

# Run with custom port
docker run -d -p 9090:9090 parse-video -port 9090

# Run with basic auth
docker run -d -p 8080:8080 -e PARSE_VIDEO_USERNAME=user -e PARSE_VIDEO_PASSWORD=pass parse-video
```

## Architecture

### Core Components

1. **Parser System** (`parser/`):
   - `parser.go`: Main entry point with URL routing logic
   - `vars.go`: Defines platform constants, interfaces, and data structures
   - Platform-specific parsers (e.g., `douyin.go`, `kuaishou.go`)

2. **CLI & Web Server** (`cmd/`):
   - `root.go`: Cobra root command (default subcommand: serve)
   - `serve.go`: Gin-based HTTP server with middleware stack
   - `parse.go`: CLI subcommand for parsing share links (single/batch)
   - `id.go`: CLI subcommand for parsing by video ID + platform
   - `download.go`: Media file download logic
   - `output.go`: Output formatting (text/JSON)
   - `handlers.go`: HTTP route handlers (v1 API + legacy compat)
   - `response.go`: Unified API response helpers
   - `middleware.go`: Recovery, CORS, rate limiting, basic auth, logging

3. **Entry Point** (`main.go`):
   - Embeds HTML templates via `//go:embed`
   - Initializes Cobra CLI and delegates to `cmd` package

4. **Utilities** (`utils/`):
   - `utils.go`: URL extraction utilities using regex

### Key Design Patterns

- **Strategy Pattern**: Each platform has its own parser implementing `videoShareUrlParser` and `videoIdParser` interfaces
- **Factory Pattern**: `VideoSourceInfoMapping` maps platform identifiers to their respective parsers
- **Interface Segregation**: Separate interfaces for share URL parsing and video ID parsing
- **Cobra CLI**: `cmd/` package uses spf13/cobra for subcommands (serve, parse, id, version)

### Data Flow

1. **Share URL Parsing**:
   - Extract URL from input string using regex
   - Match URL domain to platform in `VideoSourceInfoMapping`
   - Call platform-specific `parseShareUrl()` method

2. **Video ID Parsing**:
   - Direct lookup by platform source and video ID
   - Call platform-specific `parseVideoID()` method

3. **Batch Processing** (CLI `parse` subcommand):
   - Concurrent parsing using goroutines and semaphore channel (default concurrency: 8)
   - Supports file input (`--file`) and stdin (`-f -`)

4. **HTTP API**:
   - v1 API: `GET /api/v1/parse`, `GET /api/v1/parse/:source/:video_id`
   - Legacy compat: `GET /video/share/url/parse`, `GET /video/id/parse`
   - Middleware stack: Recovery → CORS → Logging → Rate Limiting → Basic Auth

## Platform Support

The project supports 20+ video platforms and 4 image album platforms. Each platform is defined in `vars.go` with:
- Unique source identifier (e.g., `SourceDouYin`)
- Associated domains for URL matching
- Parser implementation

Key platforms include:
- Video: 抖音, 快手, 小红书, 微博, 西瓜视频, etc.
- Image Albums: 抖音, 快手, 小红书, 皮皮虾
- LivePhoto: 小红书

## Configuration

### Environment Variables
- `PARSE_VIDEO_USERNAME`: Basic auth username (optional)
- `PARSE_VIDEO_PASSWORD`: Basic auth password (optional)
- `RATE_LIMIT_RPM`: Rate limit per IP per minute (default: 60)
- `CORS_ORIGINS`: Allowed CORS origins, comma-separated (default: `*`)

### Dependencies
- `github.com/gin-gonic/gin`: Web framework
- `github.com/go-resty/resty/v2`: HTTP client
- `github.com/tidwall/gjson`: JSON parsing
- `github.com/PuerkitoBio/goquery`: HTML parsing
- `github.com/spf13/cobra`: CLI framework
- `golang.org/x/time`: Rate limiting

## Code Style and Conventions

- Follow standard Go formatting
- Use interfaces for platform-specific parsers
- Error handling with descriptive messages
- Concurrent processing with proper synchronization
- Mobile user agents for platform compatibility

## Testing

- Unit tests in `*_test.go` files
- Pre-commit hooks configured for running tests
- Test cases cover platform-specific parsing logic, CLI commands, and HTTP handlers
- Focus on ID extraction and URL validation

## Adding New Platforms

1. Add source constant in `vars.go`
2. Create platform parser file implementing interfaces
3. Add mapping in `videoSourceInfoMapping`
4. Write unit tests for new parser
5. Update README.md with platform support

## Agent skills

### Issue tracker

Issues are tracked in GitHub Issues using the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Uses the default five-label triage vocabulary (needs-triage, needs-info, ready-for-agent, ready-for-human, wontfix). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context layout: one `CONTEXT.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.

---

## AI 知识库使用规则（强制）

> **IMPORTANT**: 本项目有 AI 编程知识库。任何代码修改前，必须先读取知识库中的相关文档。这不是建议，是硬性规则。违反此规则可能导致误改核心逻辑、破坏现有功能。

### 知识库位置
- 项目知识库：`docs/knowledge/`
- **入口文件（必读）**：`docs/knowledge/00_ai_entry.md`
- **全局索引（必读）**：`docs/knowledge/99_global_index.md`

### 强制工作流

**每次接到任务时，执行以下步骤：**

1. **先读** `docs/knowledge/99_global_index.md`，根据任务类型确定需要读哪些文档
2. **再读**对应文档，理解相关流程和约束
3. **然后**才开始编写或修改代码
4. **最后**判断是否需要更新知识库

跳过步骤 1-2 直接修改代码是**被禁止的**。

### 按任务类型的必读文档

| 任务类型 | 必须先读 | 原因 |
|---|---|---|
| 新增功能 | 全局索引 → 编码规则 | 确认目录放置和代码风格 |
| 修改业务逻辑 | 全局索引 → 核心流程 → 变更安全 | 确认影响范围 |
| 修改高风险区域 | 全局索引 → 变更安全 | 解析路由/中间件/认证 |
| 新增接口 | 全局索引 → 核心流程 → 编码规则 | 确认路由和响应格式 |
| 修 Bug | 全局索引 → 核心流程 | 理解上下文再修复 |

### 高风险修改约束

修改以下区域前**必须阅读变更安全文档**，否则禁止修改：
- 解析路由和平台映射表
- HTTP 中间件栈
- Basic Auth 认证逻辑
- URL 提取正则
- 部署配置/CI/CD

禁止事项：
- 禁止一次性大范围重构稳定代码
- 禁止删除未知用途代码
- 禁止未确认调用方就改公共函数签名
- 禁止把密钥写入代码

### 变更后知识库维护

代码发生以下变更后，必须同步更新对应的知识库文档：
- 新增/删除 API → 项目地图 + 核心流程 + 全局索引
- 修改中间件 → 核心流程 + 变更安全
- 新增/修改平台解析器 → 项目地图 + 全局索引
- 新增环境变量 → 项目地图 + 构建部署 + 变更安全
- 修改部署方式 → 构建部署 + 变更安全

### 不需要更新知识库的情况
- 纯 UI 文案微调
- 无业务含义的样式调整
- 局部 bugfix 且不改变流程
- 测试用例补充但不改变规则
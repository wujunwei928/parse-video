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
# Repository Guidelines

## Project Structure & Module Organization
`main.go` is the entrypoint and embeds `templates/` for the web UI. `cmd/` contains the Cobra CLI and Gin HTTP server (`serve`, `parse`, `id`, middleware, handlers, output helpers). `parser/` holds platform-specific parsing logic plus shared routing in `parser.go` and source metadata in `vars.go`. Keep small shared helpers in `utils/`. API definitions live in `api/openapi.yaml`, static assets in `resources/`, and contributor planning notes in `docs/superpowers/`.

## Build, Test, and Development Commands
Use Go 1.24 as declared in `go.mod`.

- `go run main.go` starts the default `serve` command on port `8080`.
- `go run main.go parse "https://v.douyin.com/xxxxx"` parses a share URL from the CLI.
- `go run main.go id --source douyin "123456"` parses by platform and video ID.
- `go build -o parse-video .` builds the local binary.
- `go test ./...` runs the full test suite across `cmd/` and `parser/`.
- `pre-commit run --all-files` runs the configured checks, including `go-unit-tests` and basic YAML/JSON/TOML validation.
- `docker build -t parse-video .` builds the container image from `Dockerfile`.

## Coding Style & Naming Conventions
Follow standard Go formatting: tabs, `gofmt`, short functions where practical, and package-level organization by responsibility. Use `CamelCase` for exported names, `camelCase` for internal helpers, and keep platform parser files named after the source, such as `parser/douyin.go` or `parser/weibo.go`. Prefer descriptive error messages and keep new CLI/API options aligned with the existing Cobra and Gin patterns in `cmd/`.

## Testing Guidelines
Place tests next to the code they cover in `*_test.go` files. Match existing names like `TestIntegrationV1ParseURLSuccess` and keep table-driven cases where a parser handles multiple inputs. Run `go test ./...` before opening a PR; add or update tests whenever you change parser behavior, HTTP handlers, middleware, or CLI output.

## Commit & Pull Request Guidelines
Recent history uses conventional prefixes such as `feat(api):`, `fix:`, `docs:`, `refactor:`, and `test(api):`. Keep commits focused and scoped to one change. PRs should explain the user-visible impact, list affected platforms/endpoints/commands, mention test coverage, and update `README.md` or `api/openapi.yaml` when behavior changes. Include screenshots only when changing `templates/index.html` or other UI output.

## Configuration & Security
Use environment variables instead of hardcoded secrets: `PARSE_VIDEO_USERNAME`, `PARSE_VIDEO_PASSWORD`, `RATE_LIMIT_RPM`, and `CORS_ORIGINS`. When adding new parsers, avoid logging credentials or raw private tokens and keep outbound requests consistent with the existing platform client behavior.

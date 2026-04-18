# CLI 子命令化重构实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 parse-video 从单一 HTTP 服务器入口重构为基于 cobra 的 CLI 工具，支持 parse/id/serve/version 四个子命令。

**Architecture:** 引入 `cmd/` 包管理所有子命令，`main.go` 保留 embed 模板资源并通过 `cmd.SetTemplates(fs.FS)` 注入。parser 包零改动，CLI 层直接调用现有导出 API。所有子命令统一使用 `RunE` 返回 error，不在内部 `os.Exit`（serve 的 goroutine 通过 channel 返回错误），由 `cmd.Execute()` 统一处理退出码。`batchResult` 以 `Failed bool` 为唯一状态源，各 formatter 自行映射展示文本。

**Tech Stack:** Go 1.24, cobra + pflag, text/tabwriter, encoding/json

**Worktree:** `/code/parse-video/.worktrees/cli-refactor`（分支 `feature/cli-refactor`）

---

## 文件结构总览

| 文件 | 操作 | 职责 |
|------|------|------|
| `main.go` | 重写 | embed 模板 + `-port` 兼容层 + 调用 cmd.Execute() |
| `cmd/root.go` | 新建 | 根命令、版本变量、无参数默认 serve |
| `cmd/serve.go` | 新建 | HTTP 服务器（迁移自原 main.go） |
| `cmd/parse.go` | 新建 | 分享链接解析子命令 |
| `cmd/id.go` | 新建 | 视频 ID 解析子命令 |
| `cmd/output.go` | 新建 | 输出格式化公共逻辑（text/json/table） |
| `cmd/version.go` | 新建 | 版本子命令 |
| `cmd/cmd_test.go` | 新建 | CLI 集成测试 |
| `cmd/output_test.go` | 新建 | 输出格式化单元测试 |

---

### Task 1: 安装 cobra 依赖

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: 安装 cobra（固定版本）**

```bash
cd /code/parse-video/.worktrees/cli-refactor
go get github.com/spf13/cobra@v1.9.1
```

- [ ] **Step 2: 验证依赖安装成功**

```bash
go mod tidy
grep "spf13/cobra" go.mod
```

Expected: 输出包含 `github.com/spf13/cobra v1.9.1`

- [ ] **Step 3: 提交**

```bash
git add go.mod go.sum
git commit -m "chore: add cobra v1.9.1 dependency for CLI framework

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 2: 创建 cmd/root.go + cmd/serve.go 骨架

> **注意**：root.go 的 `RunE` 引用 `runServe`，因此 serve.go 必须在同一个 Task 中一起创建，否则编译不过。

**Files:**
- Create: `cmd/root.go`
- Create: `cmd/serve.go`

- [ ] **Step 1: 创建 cmd 目录**

```bash
mkdir -p /code/parse-video/.worktrees/cli-refactor/cmd
```

- [ ] **Step 2: 创建 cmd/root.go**

```go
package cmd

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/cobra"
)

// Version 版本号，通过 ldflags 注入
var Version = "dev"

// templateFS 模板文件系统，由 main.go 通过 SetTemplates 注入
var templateFS fs.FS

var rootCmd = &cobra.Command{
	Use:   "parse-video",
	Short: "视频解析工具，支持 20+ 平台去水印解析",
	// 无子命令时默认执行 serve
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServe(cmd, args)
	},
}

// SetTemplates 注入模板资源（由 main.go 调用）
func SetTemplates(f fs.FS) {
	templateFS = f
}

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("port", "p", "8080", "服务监听端口")
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate(fmt.Sprintf("parse-video %s\n", Version))
}
```

- [ ] **Step 3: 创建 cmd/serve.go（骨架，包含 runServe 函数）**

```go
package cmd

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"

	"github.com/wujunwei928/parse-video/parser"
)

type httpResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动 HTTP 解析服务",
	RunE:  runServe,
}

// runServe 由 serveCmd.RunE 和 rootCmd.RunE（无参数默认）共同调用
func runServe(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetString("port")
	addr := ":" + port

	r := gin.Default()

	// Basic Auth 中间件
	if os.Getenv("PARSE_VIDEO_USERNAME") != "" && os.Getenv("PARSE_VIDEO_PASSWORD") != "" {
		r.Use(gin.BasicAuth(gin.Accounts{
			os.Getenv("PARSE_VIDEO_USERNAME"): os.Getenv("PARSE_VIDEO_PASSWORD"),
		}))
	}

	// 加载模板
	if templateFS != nil {
		tmpl, err := template.ParseFS(templateFS, "*.html")
		if err != nil {
			return fmt.Errorf("模板加载失败: %w", err)
		}
		r.SetHTMLTemplate(tmpl)
		r.GET("/", func(c *gin.Context) {
			c.HTML(200, "index.html", gin.H{
				"title": "github.com/wujunwei928/parse-video Demo",
			})
		})
	}

	// 解析接口
	r.GET("/video/share/url/parse", func(c *gin.Context) {
		paramUrl := c.Query("url")
		parseRes, err := parser.ParseVideoShareUrlByRegexp(paramUrl)
		jsonRes := httpResponse{
			Code: 200,
			Msg:  "解析成功",
			Data: parseRes,
		}
		if err != nil {
			jsonRes = httpResponse{
				Code: 201,
				Msg:  err.Error(),
			}
		}
		c.JSON(http.StatusOK, jsonRes)
	})

	r.GET("/video/id/parse", func(c *gin.Context) {
		videoId := c.Query("video_id")
		source := c.Query("source")

		parseRes, err := parser.ParseVideoId(source, videoId)
		jsonRes := httpResponse{
			Code: 200,
			Msg:  "解析成功",
			Data: parseRes,
		}
		if err != nil {
			jsonRes = httpResponse{
				Code: 201,
				Msg:  err.Error(),
			}
		}
		c.JSON(200, jsonRes)
	})

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	log.Printf("服务启动，监听端口 %s", addr)

	// 通过 channel 接收 goroutine 中的启动错误，避免 os.Exit
	serveErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serveErr <- fmt.Errorf("端口 %s 已被占用: %w", addr, err)
			return
		}
		serveErr <- nil
	}()

	// 等待中断信号或启动错误
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	select {
	case err := <-serveErr:
		return err
	case <-quit:
	}
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("服务器关闭超时: %w", err)
	}
	log.Println("Server exiting")
	return nil
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
```

- [ ] **Step 4: 验证编译**

```bash
cd /code/parse-video/.worktrees/cli-refactor
go build ./cmd/...
```

Expected: 无错误输出

- [ ] **Step 5: 提交**

```bash
git add cmd/root.go cmd/serve.go
git commit -m "feat: add cobra root command and serve subcommand

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 3: 创建 cmd/version.go（版本子命令）

**Files:**
- Create: `cmd/version.go`

- [ ] **Step 1: 创建 cmd/version.go**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("parse-video %s\n", Version)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
```

- [ ] **Step 2: 验证编译**

```bash
cd /code/parse-video/.worktrees/cli-refactor
go vet ./cmd/...
```

Expected: 无错误输出

- [ ] **Step 3: 提交**

```bash
git add cmd/version.go
git commit -m "feat: add version subcommand

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 4: 创建 cmd/output.go（输出格式化）

**Files:**
- Create: `cmd/output.go`
- Create: `cmd/output_test.go`

- [ ] **Step 1: 编写输出格式化测试**

```go
package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/wujunwei928/parse-video/parser"
)

func TestFormatText(t *testing.T) {
	info := &parser.VideoParseInfo{}
	info.Author.Uid = "123"
	info.Author.Name = "张三"
	info.Title = "测试视频"
	info.VideoUrl = "https://example.com/video.mp4"
	info.CoverUrl = "https://example.com/cover.jpg"
	info.MusicUrl = "https://example.com/music.mp3"

	out := &bytes.Buffer{}
	formatText(out, info)

	result := out.String()
	if !strings.Contains(result, "标题: 测试视频") {
		t.Errorf("text 输出应包含标题行，实际: %s", result)
	}
	if !strings.Contains(result, "张三") {
		t.Errorf("text 输出应包含作者名，实际: %s", result)
	}
	if !strings.Contains(result, "https://example.com/video.mp4") {
		t.Errorf("text 输出应包含视频地址，实际: %s", result)
	}
}

func TestFormatTextWithImages(t *testing.T) {
	info := &parser.VideoParseInfo{}
	info.Author.Uid = "123"
	info.Author.Name = "张三"
	info.Title = "图集测试"
	info.Images = []parser.ImgInfo{
		{Url: "https://example.com/img1.jpg", LivePhotoUrl: "https://example.com/live.mp4"},
		{Url: "https://example.com/img2.jpg"},
	}

	out := &bytes.Buffer{}
	formatText(out, info)

	result := out.String()
	if !strings.Contains(result, "图片列表") {
		t.Errorf("含图集时应输出图片列表，实际: %s", result)
	}
	if !strings.Contains(result, "https://example.com/img1.jpg") {
		t.Errorf("应包含图片地址，实际: %s", result)
	}
	if !strings.Contains(result, "LivePhoto: https://example.com/live.mp4") {
		t.Errorf("应包含 LivePhoto 地址，实际: %s", result)
	}
}

func TestFormatJSON(t *testing.T) {
	info := &parser.VideoParseInfo{}
	info.Author.Uid = "123"
	info.Author.Name = "张三"
	info.Title = "JSON测试"
	info.VideoUrl = "https://example.com/video.mp4"

	out := &bytes.Buffer{}
	err := formatJSON(out, info)
	if err != nil {
		t.Fatalf("formatJSON 失败: %v", err)
	}

	// 用 json.Unmarshal 验证结构，而非字符串匹配
	var result parser.VideoParseInfo
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("输出不是有效 JSON: %v", err)
	}
	if result.Title != "JSON测试" {
		t.Errorf("title 应为 'JSON测试'，实际: %s", result.Title)
	}
	if result.VideoUrl != "https://example.com/video.mp4" {
		t.Errorf("video_url 不正确，实际: %s", result.VideoUrl)
	}
}

func TestFormatJSONBatch(t *testing.T) {
	info := &parser.VideoParseInfo{}
	info.Title = "成功视频"

	items := []batchResult{
		{Input: "https://v.douyin.com/xxx", Failed: false, Data: info},
		{Input: "https://bad.url", Failed: true, ErrMsg: "解析失败"},
	}

	out := &bytes.Buffer{}
	err := formatJSONBatch(out, items)
	if err != nil {
		t.Fatalf("formatJSONBatch 失败: %v", err)
	}

	// NDJSON：每行一个 JSON 对象
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("批量 json 应为 2 行 NDJSON，实际 %d 行", len(lines))
	}

	var first map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("第一行不是有效 JSON: %v", err)
	}
	if first["status"] != "success" {
		t.Errorf("第一行 status 应为 success，实际: %v", first["status"])
	}

	var second map[string]interface{}
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("第二行不是有效 JSON: %v", err)
	}
	if second["status"] != "error" {
		t.Errorf("第二行 status 应为 error，实际: %v", second["status"])
	}
}

func TestFormatTable(t *testing.T) {
	info := &parser.VideoParseInfo{}
	info.Author.Name = "张三"
	info.Title = "表格测试"
	info.VideoUrl = "https://example.com/video.mp4"

	items := []batchResult{
		{Input: "https://v.douyin.com/xxx", Failed: false, Data: info},
		{Input: "https://bad.url", Failed: true, ErrMsg: "解析失败"},
	}

	out := &bytes.Buffer{}
	formatTable(out, items)

	result := out.String()
	if !strings.Contains(result, "表格测试") {
		t.Errorf("table 输出应包含标题，实际: %s", result)
	}
	if !strings.Contains(result, "失败") {
		t.Errorf("table 输出应包含失败状态，实际: %s", result)
	}
}

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"text", true},
		{"json", true},
		{"table", true},
		{"xml", false},
		{"", false},
	}
	for _, tt := range tests {
		err := validateFormat(tt.input)
		if tt.valid && err != nil {
			t.Errorf("validateFormat(%q) 不应报错，实际: %v", tt.input, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("validateFormat(%q) 应报错", tt.input)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"你好世界测试", 4, "你..."},
	}
	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
		}
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
cd /code/parse-video/.worktrees/cli-refactor
go test ./cmd/ -run "TestFormat|TestValidate|TestTruncate" -v 2>&1 | head -20
```

Expected: 编译失败，类型和函数不存在

- [ ] **Step 3: 实现 cmd/output.go**

```go
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/wujunwei928/parse-video/parser"
)

// batchResult 批量解析的单条结果
// Failed 为唯一状态源，各 formatter 通过 toMarshal()/statusText() 自行映射
type batchResult struct {
	Input  string                 `json:"input"`
	Failed bool                   `json:"-"` // 内部使用，不序列化
	Data   *parser.VideoParseInfo `json:"data,omitempty"`
	ErrMsg string                 `json:"error,omitempty"`
}

// marshalBatchResult 用于 JSON 序列化的结构
type marshalBatchResult struct {
	Input  string                 `json:"input"`
	Status string                 `json:"status"`
	Data   *parser.VideoParseInfo `json:"data"`
	Error  string                 `json:"error,omitempty"`
}

// toMarshal 转换为 JSON 序列化结构
func (r batchResult) toMarshal() marshalBatchResult {
	status := "success"
	errMsg := ""
	if r.Failed {
		status = "error"
		errMsg = r.ErrMsg
	}
	return marshalBatchResult{
		Input:  r.Input,
		Status: status,
		Data:   r.Data,
		Error:  errMsg,
	}
}

// statusText 返回本地化状态文本（用于 text/table 输出）
func (r batchResult) statusText() string {
	if r.Failed {
		return "失败"
	}
	return "成功"
}

// validateFormat 校验输出格式是否合法
func validateFormat(format string) error {
	switch format {
	case "text", "json", "table":
		return nil
	default:
		return fmt.Errorf("不支持的输出格式: %s，可选值: json, table, text", format)
	}
}

// formatText 将单条解析结果格式化为 text 输出
func formatText(w io.Writer, info *parser.VideoParseInfo) {
	fmt.Fprintf(w, "标题: %s\n", info.Title)
	fmt.Fprintf(w, "作者: %s (UID: %s)\n", info.Author.Name, info.Author.Uid)
	if info.VideoUrl != "" {
		fmt.Fprintf(w, "视频地址: %s\n", info.VideoUrl)
	}
	if info.CoverUrl != "" {
		fmt.Fprintf(w, "封面地址: %s\n", info.CoverUrl)
	}
	if info.MusicUrl != "" {
		fmt.Fprintf(w, "音乐地址: %s\n", info.MusicUrl)
	}
	if len(info.Images) > 0 {
		fmt.Fprintf(w, "图片列表:\n")
		for i, img := range info.Images {
			if img.LivePhotoUrl != "" {
				fmt.Fprintf(w, "  [%d] %s (LivePhoto: %s)\n", i+1, img.Url, img.LivePhotoUrl)
			} else {
				fmt.Fprintf(w, "  [%d] %s\n", i+1, img.Url)
			}
		}
	} else {
		fmt.Fprintf(w, "图片数量: 0\n")
	}
}

// formatJSON 将单条解析结果格式化为 JSON 输出（无缩进，适合管道）
func formatJSON(w io.Writer, info *parser.VideoParseInfo) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(info)
}

// formatJSONBatch 将批量解析结果格式化为 NDJSON 输出
func formatJSONBatch(w io.Writer, items []batchResult) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for _, item := range items {
		if err := enc.Encode(item.toMarshal()); err != nil {
			return err
		}
	}
	return nil
}

// formatTable 将批量解析结果格式化为表格输出
func formatTable(w io.Writer, items []batchResult) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "输入\t状态\t标题\t作者\t视频地址")
	for _, item := range items {
		if item.Data != nil {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
				truncate(item.Input, 30),
				item.statusText(),
				truncate(item.Data.Title, 15),
				truncate(item.Data.Author.Name, 10),
				item.Data.VideoUrl,
			)
		} else {
			fmt.Fprintf(tw, "%s\t%s\t-\t-\t%s\n",
				truncate(item.Input, 30),
				item.statusText(),
				item.ErrMsg,
			)
		}
	}
	tw.Flush()
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	return string(r[:maxLen-3]) + "..."
}

// outputResult 根据格式输出单条结果
func outputResult(w io.Writer, format string, input string, info *parser.VideoParseInfo) error {
	switch format {
	case "json":
		return formatJSON(w, info)
	case "table":
		items := []batchResult{{Input: input, Failed: false, Data: info}}
		formatTable(w, items)
		return nil
	default:
		formatText(w, info)
		return nil
	}
}

// outputBatch 根据格式输出批量结果
func outputBatch(w io.Writer, format string, items []batchResult) error {
	switch format {
	case "json":
		return formatJSONBatch(w, items)
	case "table":
		formatTable(w, items)
		return nil
	default:
		for i, item := range items {
			if i > 0 {
				fmt.Fprintln(w)
			}
			if item.Data != nil {
				formatText(w, item.Data)
			} else {
				fmt.Fprintf(w, "[失败] %s\n错误: %s\n", item.Input, item.ErrMsg)
			}
		}
		return nil
	}
}
```

- [ ] **Step 4: 运行测试**

```bash
cd /code/parse-video/.worktrees/cli-refactor
go test ./cmd/ -run "TestFormat|TestValidate|TestTruncate" -v
```

Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add cmd/output.go cmd/output_test.go
git commit -m "feat: add output formatting for text/json/table

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 5: 创建 cmd/parse.go（parse 子命令）

**Files:**
- Create: `cmd/parse.go`

- [ ] **Step 1: 创建 cmd/parse.go**

```go
package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wujunwei928/parse-video/parser"
)

var parseCmd = &cobra.Command{
	Use:   "parse [url...]",
	Short: "解析视频分享链接",
	Long:  "解析视频分享链接，支持单条和多条。也可以直接传入包含链接的分享文案。",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		if err := validateFormat(format); err != nil {
			return err
		}

		filePath, _ := cmd.Flags().GetString("file")

		// 校验互斥：位置参数和 --file 不能同时提供
		if len(args) > 0 && filePath != "" {
			return fmt.Errorf("不能同时指定链接和文件输入")
		}

		// 收集所有输入
		var inputs []string
		if filePath != "" {
			var err error
			inputs, err = readInputsFromFile(filePath)
			if err != nil {
				return err
			}
		} else if len(args) > 0 {
			inputs = args
		} else {
			return fmt.Errorf("请提供要解析的链接或指定 --file")
		}

		// 空输入（如空 stdin）直接返回
		if len(inputs) == 0 {
			return nil
		}

		// 单条解析
		if len(inputs) == 1 {
			info, err := parser.ParseVideoShareUrlByRegexp(inputs[0])
			if err != nil {
				return fmt.Errorf("解析失败: %w", err)
			}
			return outputResult(os.Stdout, format, inputs[0], info)
		}

		// 批量解析
		return runBatchParse(inputs, format)
	},
}

func init() {
	rootCmd.AddCommand(parseCmd)
	parseCmd.Flags().StringP("file", "f", "", "从文件读取链接（每行一个，- 代表 stdin）")
	parseCmd.Flags().String("format", "text", "输出格式: json, table, text")
}

// readInputsFromFile 从文件或 stdin 读取输入
func readInputsFromFile(filePath string) ([]string, error) {
	var reader io.Reader

	if filePath == "-" {
		reader = os.Stdin
	} else {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("无法读取文件: %s: %w", filePath, err)
		}
		defer f.Close()
		reader = f
	}

	var inputs []string
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			inputs = append(inputs, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取输入失败: %w", err)
	}
	return inputs, nil
}

// runBatchParse 执行批量解析，返回 error（由上层决定退出码）
func runBatchParse(inputs []string, format string) error {
	items := make([]batchResult, 0, len(inputs))
	failCount := 0

	for _, input := range inputs {
		info, err := parser.ParseVideoShareUrlByRegexp(input)
		if err != nil {
			items = append(items, batchResult{
				Input:  input,
				Failed: true,
				ErrMsg: err.Error(),
			})
			failCount++
		} else {
			items = append(items, batchResult{
				Input:  input,
				Failed: false,
				Data:   info,
			})
		}
	}

	if err := outputBatch(os.Stdout, format, items); err != nil {
		return err
	}

	// 全部失败时返回错误（退出码 1）
	if len(inputs) > 0 && failCount == len(inputs) {
		return fmt.Errorf("所有 %d 条解析均失败", len(inputs))
	}
	return nil
}
```

- [ ] **Step 2: 验证编译**

```bash
cd /code/parse-video/.worktrees/cli-refactor
go vet ./cmd/...
```

Expected: 无错误

- [ ] **Step 3: 提交**

```bash
git add cmd/parse.go
git commit -m "feat: add parse subcommand for URL parsing

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 6: 创建 cmd/id.go（id 子命令）

**Files:**
- Create: `cmd/id.go`

- [ ] **Step 1: 创建 cmd/id.go**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wujunwei928/parse-video/parser"
)

// 仅支持 URL 解析（不支持 ID 解析）的平台
var urlOnlySources = map[string]bool{
	"kuaishou": true, "zuiyou": true, "xinpianchang": true,
	"redbook": true, "bilibili": true,
}

// validateSource 校验 source 是否合法，返回错误描述
func validateSource(source string) error {
	if source == "" {
		return fmt.Errorf("必须指定 --source 参数")
	}
	// 检查是否为仅支持 URL 解析的平台
	if urlOnlySources[source] {
		return fmt.Errorf("平台 %s 暂不支持视频 ID 解析，请使用 parse 命令通过分享链接解析", source)
	}
	// 遍历 parser.VideoSourceInfoMapping 检查是否有此平台且支持 ID 解析
	info, exists := parser.VideoSourceInfoMapping[source]
	if !exists {
		return fmt.Errorf("未知的平台: %s", source)
	}
	if info.VideoIdParser == nil {
		return fmt.Errorf("平台 %s 暂不支持视频 ID 解析，请使用 parse 命令通过分享链接解析", source)
	}
	return nil
}

var idCmd = &cobra.Command{
	Use:   "id <video_id>",
	Short: "根据视频 ID 解析",
	Long:  "根据视频 ID 和平台来源解析视频信息。需要通过 --source 指定平台。",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		if err := validateFormat(format); err != nil {
			return err
		}

		source, _ := cmd.Flags().GetString("source")
		if err := validateSource(source); err != nil {
			return err
		}

		videoID := args[0]
		info, err := parser.ParseVideoId(source, videoID)
		if err != nil {
			return fmt.Errorf("解析失败: %w", err)
		}
		return outputResult(os.Stdout, format, videoID, info)
	},
}

func init() {
	rootCmd.AddCommand(idCmd)
	idCmd.Flags().StringP("source", "s", "", "视频来源平台（必填）")
	idCmd.Flags().String("format", "text", "输出格式: json, table, text")
	_ = idCmd.MarkFlagRequired("source")
}
```

- [ ] **Step 2: 验证编译**

```bash
cd /code/parse-video/.worktrees/cli-refactor
go vet ./cmd/...
```

Expected: 无错误

- [ ] **Step 3: 提交**

```bash
git add cmd/id.go
git commit -m "feat: add id subcommand for video ID parsing

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 7: 重写 main.go（精简入口 + -port 兼容层）

**Files:**
- Modify: `main.go`

- [ ] **Step 1: 重写 main.go**

```go
package main

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/wujunwei928/parse-video/cmd"
)

//go:embed templates/*
var templateFS embed.FS

func main() {
	// pflag 不接受单横线长参数（如 -port），
	// 在进入 cobra 前将 -port 规范为 --port 以保持向后兼容
	normalizeArgs()

	sub, err := fs.Sub(templateFS, "templates")
	if err != nil {
		log.Fatal(err)
	}
	cmd.SetTemplates(sub)
	cmd.Execute()
}

// normalizeArgs 将 -flag 形式的已知参数转换为 --flag
func normalizeArgs() {
	for i, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") {
			name := strings.TrimPrefix(arg, "-")
			// 已知的长参数名（不含 shorthand）
			if name == "port" || name == "version" {
				os.Args[i+1] = "--" + name
			}
		}
	}
}
```

- [ ] **Step 2: 编译验证**

```bash
cd /code/parse-video/.worktrees/cli-refactor
go build -o /tmp/parse-video-test .
```

Expected: 编译成功，无错误

- [ ] **Step 3: 功能验证——无参数默认启动 serve**

```bash
timeout 3 /tmp/parse-video-test 2>&1 || true
```

Expected: 输出包含 "服务启动" 和 "监听端口 :8080"

- [ ] **Step 4: 功能验证——-port 向后兼容**

```bash
timeout 3 /tmp/parse-video-test -port 9091 2>&1 || true
```

Expected: 输出包含 "监听端口 :9091"（单横线 -port 仍然有效）

- [ ] **Step 5: 功能验证——version 子命令**

```bash
/tmp/parse-video-test version
```

Expected: 输出 "parse-video dev"

- [ ] **Step 6: 功能验证——--version flag**

```bash
/tmp/parse-video-test --version
```

Expected: 输出版本信息

- [ ] **Step 7: 功能验证——parse --help**

```bash
/tmp/parse-video-test parse --help
```

Expected: 显示 parse 子命令用法

- [ ] **Step 8: 功能验证——id --help**

```bash
/tmp/parse-video-test id --help
```

Expected: 显示 id 子命令用法

- [ ] **Step 9: 运行全部测试**

```bash
cd /code/parse-video/.worktrees/cli-refactor
go test ./... -v -timeout 60s
```

Expected: 全部 PASS

- [ ] **Step 10: 提交**

```bash
git add main.go
git commit -m "feat: rewrite main.go as thin entry point with cobra CLI

- Preserve -port backward compatibility via arg normalization
- Embed templates and inject into cmd package

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 8: 集成测试

**Files:**
- Create: `cmd/cmd_test.go`

- [ ] **Step 1: 编写 CLI 集成测试**

```go
package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRootCommandHasRunE(t *testing.T) {
	if rootCmd.RunE == nil {
		t.Error("rootCmd.RunE 不应为 nil（无参数时应默认执行 serve）")
	}
}

func TestParseCommandRequiresInput(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd := parseCmd
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)
	// 重置 flag 避免串测试
	cmd.Flags().Set("file", "")
	cmd.Flags().Set("format", "text")
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Error("无输入时应返回错误")
	}
	if !strings.Contains(err.Error(), "请提供") {
		t.Errorf("错误信息应提示提供链接，实际: %v", err)
	}
}

func TestParseCommandMutualExclusive(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd := parseCmd
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)
	cmd.Flags().Set("format", "text")
	cmd.SetArgs([]string{"https://v.douyin.com/xxx", "--file", "test.txt"})

	err := cmd.Execute()
	if err == nil {
		t.Error("同时指定链接和文件时应返回错误")
	}
	if !strings.Contains(err.Error(), "不能同时") {
		t.Errorf("错误信息应说明互斥，实际: %v", err)
	}
}

func TestIdCommandUnknownSource(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd := idCmd
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"123456", "--source", "unknown_platform"})

	err := cmd.Execute()
	if err == nil {
		t.Error("未知平台应返回错误")
	}
	if !strings.Contains(err.Error(), "未知") {
		t.Errorf("错误信息应提示未知平台，实际: %v", err)
	}
}

func TestIdCommandURLOnlySource(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd := idCmd
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)
	cmd.SetArgs([]string{"123456", "--source", "redbook"})

	err := cmd.Execute()
	if err == nil {
		t.Error("不支持 ID 解析的平台应返回错误")
	}
	if !strings.Contains(err.Error(), "暂不支持视频 ID 解析") {
		t.Errorf("错误信息应提示不支持 ID 解析，实际: %v", err)
	}
}

func TestValidateSource(t *testing.T) {
	tests := []struct {
		source string
		errMsg string // 空串表示不应报错
	}{
		{"douyin", ""},
		{"kuaishou", "暂不支持视频 ID 解析"},
		{"unknown", "未知"},
		{"", "必须指定"},
	}
	for _, tt := range tests {
		err := validateSource(tt.source)
		if tt.errMsg == "" && err != nil {
			t.Errorf("validateSource(%q) 不应报错，实际: %v", tt.source, err)
		}
		if tt.errMsg != "" {
			if err == nil {
				t.Errorf("validateSource(%q) 应报错", tt.source)
			} else if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("validateSource(%q) 错误应包含 %q，实际: %v", tt.source, tt.errMsg, err)
			}
		}
	}
}

func TestReadInputsFromFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "inputs-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := "https://v.douyin.com/xxx\n\nhttps://v.kuaishou.com/yyy\n"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	inputs, err := readInputsFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("readInputsFromFile 失败: %v", err)
	}
	if len(inputs) != 2 {
		t.Errorf("应读取 2 个链接（跳过空行），实际: %d", len(inputs))
	}
	if inputs[0] != "https://v.douyin.com/xxx" {
		t.Errorf("第一个链接不正确: %s", inputs[0])
	}
}

func TestReadInputsEmptyFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "empty-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	inputs, err := readInputsFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("读取空文件失败: %v", err)
	}
	if len(inputs) != 0 {
		t.Errorf("空文件应返回 0 个输入，实际: %d", len(inputs))
	}
}

func TestReadInputsFileNotFound(t *testing.T) {
	_, err := readInputsFromFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("文件不存在时应返回错误")
	}
}
```

- [ ] **Step 2: 运行全部测试**

```bash
cd /code/parse-video/.worktrees/cli-refactor
go test ./cmd/ -v -timeout 60s
```

Expected: 全部 PASS

- [ ] **Step 3: 运行 parser 包测试确认未受影响**

```bash
cd /code/parse-video/.worktrees/cli-refactor
go test ./parser/ -v -timeout 60s
```

Expected: 全部 PASS（parser 包未改动）

- [ ] **Step 4: 提交**

```bash
git add cmd/cmd_test.go
git commit -m "test: add CLI integration tests

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

### Task 9: 清理和验证

**Files:**
- 无新增/修改

- [ ] **Step 1: 运行完整测试套件**

```bash
cd /code/parse-video/.worktrees/cli-refactor
go test ./... -timeout 60s
```

Expected: 全部 PASS

- [ ] **Step 2: go vet 静态检查**

```bash
go vet ./...
```

Expected: 无警告

- [ ] **Step 3: 构建二进制并验证所有子命令**

```bash
go build -o /tmp/parse-video-final .

# 验证无参数默认 serve
timeout 2 /tmp/parse-video-final 2>&1 || true

# 验证 -port 向后兼容
timeout 2 /tmp/parse-video-final -port 9092 2>&1 || true

# 验证 serve 子命令
timeout 2 /tmp/parse-video-final serve -p 9093 2>&1 || true

# 验证 version
/tmp/parse-video-final version

# 验证 --version
/tmp/parse-video-final --version

# 验证 parse --help
/tmp/parse-video-final parse --help

# 验证 id --help
/tmp/parse-video-final id --help

# 验证无输入错误
/tmp/parse-video-final parse 2>&1; echo "exit: $?"

# 验证无效 source
/tmp/parse-video-final id 123 --source unknown 2>&1; echo "exit: $?"
```

Expected: 所有命令输出符合预期，退出码正确

---

## 自审清单

- [ ] **Spec 覆盖**：子命令结构（Task 2-7）、输出格式（Task 4）、向后兼容（Task 7 -port 规范化 + 无参数默认 serve）、错误处理（Task 5-6 RunE 返回 error）、版本管理（Task 2 + Task 3）
- [ ] **占位符扫描**：无 TBD/TODO，所有步骤包含完整代码
- [ ] **类型一致性**：`batchResult` 在 output.go 定义（Failed bool + toMarshal/statusText 方法），parse.go/id.go 使用一致；`outputResult` 签名包含 input 参数；`validateSource` 遍历 `parser.VideoSourceInfoMapping`
- [ ] **parser 包零改动**：计划中无修改 parser/ 文件的任务
- [ ] **无 os.Exit 在 RunE 中**：所有子命令通过 RunE 返回 error，由 cmd.Execute() 统一处理退出码；serve 的 goroutine 通过 channel 返回错误而非 os.Exit
- [ ] **依赖顺序正确**：Task 2 同时创建 root.go + serve.go，避免前向引用问题

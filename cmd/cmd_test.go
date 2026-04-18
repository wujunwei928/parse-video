package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/wujunwei928/parse-video/parser"
)

// executeSubcommand 通过 rootCmd 执行子命令，避免直接调用子命令导致 rootCmd.RunE 被触发
func executeSubcommand(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	// 每次执行前重置 port flag 避免串测试
	rootCmd.Flags().Set("port", "8080")
	err := rootCmd.Execute()
	return buf.String(), err
}

func TestRootCommandHasRunE(t *testing.T) {
	if rootCmd.RunE == nil {
		t.Error("rootCmd.RunE 不应为 nil（无参数时应默认执行 serve）")
	}
}

func TestParseCommandRequiresInput(t *testing.T) {
	_, err := executeSubcommand("parse")
	if err == nil {
		t.Error("无输入时应返回错误")
	}
	if !strings.Contains(err.Error(), "请提供") {
		t.Errorf("错误信息应提示提供链接，实际: %v", err)
	}
}

func TestParseCommandMutualExclusive(t *testing.T) {
	_, err := executeSubcommand("parse", "https://v.douyin.com/xxx", "--file", "test.txt")
	if err == nil {
		t.Error("同时指定链接和文件时应返回错误")
	}
	if !strings.Contains(err.Error(), "不能同时") {
		t.Errorf("错误信息应说明互斥，实际: %v", err)
	}
}

func TestIdCommandUnknownSource(t *testing.T) {
	_, err := executeSubcommand("id", "123456", "--source", "unknown_platform")
	if err == nil {
		t.Error("未知平台应返回错误")
	}
	if !strings.Contains(err.Error(), "未知") {
		t.Errorf("错误信息应提示未知平台，实际: %v", err)
	}
}

func TestIdCommandURLOnlySource(t *testing.T) {
	_, err := executeSubcommand("id", "123456", "--source", "redbook")
	if err == nil {
		t.Error("不支持 ID 解析的平台应返回错误")
	}
	if !strings.Contains(err.Error(), "暂不支持视频 ID 解析") {
		t.Errorf("错误信息应提示不支持 ID 解析，实际: %v", err)
	}
}

func TestIdCommandKuaishouNotSupported(t *testing.T) {
	_, err := executeSubcommand("id", "123456", "--source", "kuaishou")
	if err == nil {
		t.Error("快手不支持 ID 解析应返回错误")
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

	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("第一行不是有效 JSON: %v", err)
	}
	if first["status"] != "success" {
		t.Errorf("第一行 status 应为 success，实际: %v", first["status"])
	}

	var second map[string]any
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

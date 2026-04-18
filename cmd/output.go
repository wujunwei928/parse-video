package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/wujunwei928/parse-video/parser"
)

const (
	FormatText = "text"
	FormatJSON = "json"
)

type batchResult struct {
	Input  string                 `json:"input"`
	Failed bool                   `json:"-"`
	Data   *parser.VideoParseInfo `json:"data,omitempty"`
	ErrMsg string                 `json:"error,omitempty"`
}

type marshalBatchResult struct {
	Input  string                 `json:"input"`
	Status string                 `json:"status"`
	Data   *parser.VideoParseInfo `json:"data"`
	Error  string                 `json:"error,omitempty"`
}

func (r batchResult) toMarshal() marshalBatchResult {
	status := "success"
	errMsg := ""
	if r.Failed {
		status = "error"
		errMsg = r.ErrMsg
	}
	return marshalBatchResult{Input: r.Input, Status: status, Data: r.Data, Error: errMsg}
}

func validateFormat(format string) error {
	switch format {
	case FormatText, FormatJSON:
		return nil
	default:
		return fmt.Errorf("不支持的输出格式: %s，可选值: json, text", format)
	}
}

func formatTextOutput(w io.Writer, info *parser.VideoParseInfo) {
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

func formatJSONOutput(w io.Writer, info *parser.VideoParseInfo) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(info)
}

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

func outputResult(w io.Writer, format string, info *parser.VideoParseInfo) error {
	switch format {
	case FormatJSON:
		return formatJSONOutput(w, info)
	default:
		formatTextOutput(w, info)
		return nil
	}
}

func outputBatch(w io.Writer, format string, items []batchResult) error {
	switch format {
	case FormatJSON:
		return formatJSONBatch(w, items)
	default:
		for i, item := range items {
			if i > 0 {
				fmt.Fprintln(w)
			}
			if item.Data != nil {
				formatTextOutput(w, item.Data)
			} else {
				fmt.Fprintf(w, "[失败] %s\n错误: %s\n", item.Input, item.ErrMsg)
			}
		}
		return nil
	}
}

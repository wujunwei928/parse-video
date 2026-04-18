package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

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
		if len(args) > 0 && filePath != "" {
			return fmt.Errorf("不能同时指定链接和文件输入")
		}
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
		if len(inputs) == 0 {
			return nil
		}
		if len(inputs) == 1 {
			info, err := parser.ParseVideoShareUrlByRegexp(inputs[0])
			if err != nil {
				return fmt.Errorf("解析失败: %w", err)
			}
			if err := outputResult(os.Stdout, format, info); err != nil {
				return err
			}
			download, _ := cmd.Flags().GetBool("download")
			if download {
				outputDir, _ := cmd.Flags().GetString("output-dir")
				if err := downloadMedia(info, outputDir); err != nil {
					return err
				}
			}
			return nil
		}
		return runBatchParse(cmd, inputs, format)
	},
}

func init() {
	rootCmd.AddCommand(parseCmd)
	parseCmd.Flags().StringP("file", "f", "", "从文件读取链接（每行一个，- 代表 stdin）")
	parseCmd.Flags().String("format", FormatText, "输出格式: json, text")
	parseCmd.Flags().BoolP("download", "d", false, "下载解析到的媒体文件")
	parseCmd.Flags().StringP("output-dir", "o", ".", "下载文件保存目录")
}

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

func runBatchParse(cmd *cobra.Command, inputs []string, format string) error {
	items := make([]batchResult, len(inputs))
	var wg sync.WaitGroup
	for i, input := range inputs {
		wg.Add(1)
		go func(idx int, in string) {
			defer wg.Done()
			info, err := parser.ParseVideoShareUrlByRegexp(in)
			if err != nil {
				items[idx] = batchResult{Input: in, Failed: true, ErrMsg: err.Error()}
			} else {
				items[idx] = batchResult{Input: in, Failed: false, Data: info}
			}
		}(i, input)
	}
	wg.Wait()

	if err := outputBatch(os.Stdout, format, items); err != nil {
		return err
	}

	download, _ := cmd.Flags().GetBool("download")
	if download {
		outputDir, _ := cmd.Flags().GetString("output-dir")
		for _, item := range items {
			if !item.Failed && item.Data != nil {
				if err := downloadMedia(item.Data, outputDir); err != nil {
					fmt.Fprintf(os.Stderr, "下载失败 [%s]: %v\n", item.Input, err)
				}
			}
		}
	}

	failCount := 0
	for _, item := range items {
		if item.Failed {
			failCount++
		}
	}
	if failCount == len(inputs) {
		return fmt.Errorf("所有 %d 条解析均失败", len(inputs))
	}
	return nil
}

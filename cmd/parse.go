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
			return outputResult(os.Stdout, format, inputs[0], info)
		}
		return runBatchParse(inputs, format)
	},
}

func init() {
	rootCmd.AddCommand(parseCmd)
	parseCmd.Flags().StringP("file", "f", "", "从文件读取链接（每行一个，- 代表 stdin）")
	parseCmd.Flags().String("format", "text", "输出格式: json, table, text")
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

func runBatchParse(inputs []string, format string) error {
	items := make([]batchResult, 0, len(inputs))
	failCount := 0
	for _, input := range inputs {
		info, err := parser.ParseVideoShareUrlByRegexp(input)
		if err != nil {
			items = append(items, batchResult{Input: input, Failed: true, ErrMsg: err.Error()})
			failCount++
		} else {
			items = append(items, batchResult{Input: input, Failed: false, Data: info})
		}
	}
	if err := outputBatch(os.Stdout, format, items); err != nil {
		return err
	}
	if len(inputs) > 0 && failCount == len(inputs) {
		return fmt.Errorf("所有 %d 条解析均失败", len(inputs))
	}
	return nil
}

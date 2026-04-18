package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wujunwei928/parse-video/parser"
)

func validateSource(source string) error {
	if source == "" {
		return fmt.Errorf("必须指定 --source 参数")
	}
	info, exists := parser.VideoSourceInfoMapping[source]
	if !exists {
		var platforms []string
		for k := range parser.VideoSourceInfoMapping {
			platforms = append(platforms, k)
		}
		sort.Strings(platforms)
		return fmt.Errorf("未知的平台: %s\n可用平台: %s", source, strings.Join(platforms, ", "))
	}
	if info.VideoIdParser == nil {
		return fmt.Errorf("该平台暂不支持视频 ID 解析，请使用 parse 命令通过分享链接解析")
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

package cmd

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/cobra"
)

var Version = "dev"

var templateFS fs.FS

var rootCmd = &cobra.Command{
	Use:   "parse-video",
	Short: "视频解析工具，支持 20+ 平台去水印解析",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServe(cmd, args)
	},
}

func SetTemplates(f fs.FS) {
	templateFS = f
}

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

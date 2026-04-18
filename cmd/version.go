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

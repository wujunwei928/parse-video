package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func douyinCookieFlag(cmd *cobra.Command) (string, error) {
	cookie, err := cmd.Flags().GetString("cookie")
	if err != nil {
		return "", err
	}
	if cmd.Flags().Changed("cookie") && strings.TrimSpace(cookie) == "" {
		return "", fmt.Errorf("--cookie 不能为空")
	}
	return cookie, nil
}

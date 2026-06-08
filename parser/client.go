package parser

import (
	"fmt"
	"net/url"
	"sync/atomic"

	"github.com/go-resty/resty/v2"
)

var proxyURL atomic.Value // 存储 string

// InitProxy 初始化代理配置，校验 URL 合法性。
// 空字符串表示不使用代理。应在程序启动时调用。
func InitProxy(proxy string) error {
	if proxy == "" {
		return nil
	}
	if _, err := url.Parse(proxy); err != nil {
		return fmt.Errorf("PARSE_VIDEO_PROXY 格式无效: %w", err)
	}
	proxyURL.Store(proxy)
	return nil
}

// newClient 创建 resty.Client，自动注入代理配置。
func newClient() *resty.Client {
	client := resty.New()
	if proxy, ok := proxyURL.Load().(string); ok && proxy != "" {
		client.SetProxy(proxy)
	}
	return client
}

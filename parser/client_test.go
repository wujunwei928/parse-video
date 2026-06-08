package parser

import (
	"sync"
	"testing"
)

// resetProxy 清理代理状态，防止测试间互相污染
func resetProxy() {
	proxyURL.Store("")
}

func TestInitProxy_EmptyString(t *testing.T) {
	t.Cleanup(resetProxy)
	resetProxy()

	err := InitProxy("")
	if err != nil {
		t.Fatalf("InitProxy(\"\") 不应报错, got: %v", err)
	}

	client := newClient()
	if client.IsProxySet() {
		t.Fatal("未设置代理时, client 不应携带代理配置")
	}
}

func TestInitProxy_InvalidURL(t *testing.T) {
	t.Cleanup(resetProxy)
	resetProxy()

	err := InitProxy("://invalid")
	if err == nil {
		t.Fatal("无效代理地址应返回 error")
	}
}

func TestInitProxy_ValidURL(t *testing.T) {
	t.Cleanup(resetProxy)
	resetProxy()

	err := InitProxy("http://proxy.example.com:8080")
	if err != nil {
		t.Fatalf("合法代理地址不应报错, got: %v", err)
	}

	client := newClient()
	if !client.IsProxySet() {
		t.Fatal("设置代理后, client 应携带代理配置")
	}
}

func TestNewClient_Concurrent(t *testing.T) {
	t.Cleanup(resetProxy)
	resetProxy()

	_ = InitProxy("http://proxy.example.com:8080")

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := newClient()
			if !client.IsProxySet() {
				t.Errorf("goroutine %d: client 应携带代理配置", i)
			}
		}()
	}
	wg.Wait()
}

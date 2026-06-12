package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRateLimitExemptsStaticPrefix 验证 /static/ 前缀豁免限流。
// Web UI 底座重构后，页面首次加载会并发请求 7 个 CSS + 3 个 JS + favicon，
// burst=1 下若不豁免，第二个静态资源请求起即 429，导致样式/脚本加载失败。
// 静态资源是纯文件服务、无解析成本，不应受限流（限流本意是保护解析 API）。
func TestRateLimitExemptsStaticPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := newIPRateLimiterWithBurst(60, 1) // burst=1，最严格配置
	t.Cleanup(limiter.Stop)

	r := gin.New()
	r.Use(rateLimitMiddleware(limiter, "/api/v1/health", "/static/"))
	r.GET("/static/css/base.css", func(c *gin.Context) { c.String(200, "css") })
	r.GET("/api/v1/parse", func(c *gin.Context) { c.String(200, "ok") })

	// 连续 5 次 /static/ 请求：burst=1 下若不豁免会全部 429
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/static/css/base.css", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("第 %d 个 /static/ 请求应豁免限流返回 200，实际 %d", i+1, w.Code)
		}
	}

	// /api/v1/parse 仍受限流保护：burst=1，首个 200，紧接的第二个 429
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest(http.MethodGet, "/api/v1/parse", nil)
	r1.RemoteAddr = "1.2.3.4:1234"
	r.ServeHTTP(w1, r1)
	if w1.Code != 200 {
		t.Errorf("首个 /api 请求应 200，实际 %d", w1.Code)
	}
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest(http.MethodGet, "/api/v1/parse", nil)
	r2.RemoteAddr = "1.2.3.4:1234"
	r.ServeHTTP(w2, r2)
	if w2.Code != 429 {
		t.Errorf("burst=1 下第二个 /api 请求应 429（限流仍对 API 生效），实际 %d", w2.Code)
	}
}

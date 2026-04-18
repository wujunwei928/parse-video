package cmd

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRecoveryMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(recoveryMiddleware())
	r.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/panic", nil)
	r.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("panic 应返回 500，实际: %d", w.Code)
	}
	body := w.Body.String()
	if !containsJSONField(body, "status", "error") {
		t.Errorf("应返回 error 状态，实际: %s", body)
	}
	if !containsJSONField(body, "code", ErrInternal) {
		t.Errorf("应返回 INTERNAL_ERROR 错误码，实际: %s", body)
	}
}

func TestCORSMiddlewareDefaultOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(corsMiddleware("*"))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("CORS Origin 应为 *，实际: %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSMiddlewarePreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(corsMiddleware("*"))
	r.OPTIONS("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 204 {
		t.Errorf("预检应返回 204，实际: %d", w.Code)
	}
	methods := w.Header().Get("Access-Control-Allow-Methods")
	if methods != "GET, OPTIONS" {
		t.Errorf("Allow-Methods 应为 'GET, OPTIONS'，实际: %s", methods)
	}
}

func TestCORSMiddlewareWhitelist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(corsMiddleware("https://a.com,https://b.com"))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// 允许的来源
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://a.com")
	r.ServeHTTP(w, req)
	if w.Header().Get("Access-Control-Allow-Origin") != "https://a.com" {
		t.Errorf("白名单中的来源应被允许，实际: %s", w.Header().Get("Access-Control-Allow-Origin"))
	}

	// 不允许的来源
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("Origin", "https://evil.com")
	r.ServeHTTP(w2, req2)
	if w2.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("不在白名单的来源不应设置 CORS header，实际: %s", w2.Header().Get("Access-Control-Allow-Origin"))
	}
}

// containsJSONField 简单检查 JSON 响应是否包含指定字段值
func containsJSONField(body, field, value string) bool {
	return strings.Contains(body, `"`+field+`":"`+value+`"`)
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := newIPRateLimiter(2)
	r := gin.New()
	r.Use(rateLimitMiddleware(limiter, "/api/v1/health"))
	r.GET("/api/v1/parse", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/api/v1/parse", nil)
	req1.RemoteAddr = "1.2.3.4:1234"
	r.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Errorf("第一次请求应成功，实际: %d", w1.Code)
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/v1/parse", nil)
	req2.RemoteAddr = "1.2.3.4:1234"
	r.ServeHTTP(w2, req2)
	if w2.Code != 429 {
		t.Errorf("超限应返回 429，实际: %d", w2.Code)
	}
	if !containsJSONField(w2.Body.String(), "code", ErrRateLimited) {
		t.Errorf("应返回 RATE_LIMITED 错误码，实际: %s", w2.Body.String())
	}
	retryAfter := w2.Header().Get("Retry-After")
	if retryAfter != "30" {
		t.Errorf("Retry-After 应为 30（60/2），实际: %s", retryAfter)
	}
}

func TestRateLimitRetryAfterCalculation(t *testing.T) {
	tests := []struct {
		rpm      int
		expected int
	}{
		{60, 1},
		{30, 2},
		{120, 1},
		{2, 30},
		{29, 2},
		{45, 1},
		{80, 1},
	}
	for _, tt := range tests {
		limiter := newIPRateLimiter(tt.rpm)
		got := limiter.retryAfterSeconds()
		if got != tt.expected {
			t.Errorf("rpm=%d: retryAfterSeconds 应为 %d，实际: %d", tt.rpm, tt.expected, got)
		}
	}
}

func TestRateLimitExemptHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := newIPRateLimiter(1)
	r := gin.New()
	r.Use(rateLimitMiddleware(limiter, "/api/v1/health"))
	r.GET("/api/v1/health", func(c *gin.Context) {
		c.String(200, "ok")
	})
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("health 端点第 %d 次请求应成功（不限速），实际: %d", i+1, w.Code)
		}
	}
}

func TestRateLimitDifferentIPs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := newIPRateLimiter(1)
	r := gin.New()
	r.Use(rateLimitMiddleware(limiter, "/api/v1/health"))
	r.GET("/api/v1/parse", func(c *gin.Context) {
		c.String(200, "ok")
	})
	for _, ip := range []string{"1.1.1.1:1111", "2.2.2.2:2222"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/parse", nil)
		req.RemoteAddr = ip
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("不同 IP 应独立限速，%s 应成功，实际: %d", ip, w.Code)
		}
	}
}

func TestRateLimitRemoteAddrOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := newIPRateLimiter(1)
	r := gin.New()
	r.Use(rateLimitMiddleware(limiter, "/api/v1/health"))
	r.GET("/api/v1/parse", func(c *gin.Context) {
		c.String(200, "ok")
	})
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/api/v1/parse", nil)
	req1.RemoteAddr = "1.2.3.4:1234"
	req1.Header.Set("X-Forwarded-For", "9.9.9.9")
	r.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Errorf("第一次请求应成功，实际: %d", w1.Code)
	}

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/v1/parse", nil)
	req2.RemoteAddr = "1.2.3.4:1234"
	req2.Header.Set("X-Forwarded-For", "8.8.8.8")
	r.ServeHTTP(w2, req2)
	if w2.Code != 429 {
		t.Errorf("相同 RemoteAddr 应被限速（忽略 X-Forwarded-For），实际: %d", w2.Code)
	}
}

func TestRateLimitCleanupOnce(t *testing.T) {
	limiter := newIPRateLimiter(60)
	limiter.getLimiter("1.1.1.1")
	limiter.getLimiter("2.2.2.2")
	if v, ok := limiter.visitors.Load("1.1.1.1"); ok {
		entry := v.(*visitorEntry)
		entry.lastSeen = time.Now().Add(-31 * time.Minute)
	}
	limiter.cleanupOnce(time.Now().Add(-30 * time.Minute))
	if _, ok := limiter.visitors.Load("1.1.1.1"); ok {
		t.Error("过期条目应被清理")
	}
	if _, ok := limiter.visitors.Load("2.2.2.2"); !ok {
		t.Error("近期条目不应被清理")
	}
}

func TestRequestLogMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	r := gin.New()
	r.Use(requestLogMiddlewareWithWriter(&buf))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("请求应成功，实际: %d", w.Code)
	}
	logOutput := buf.String()
	if !strings.Contains(logOutput, "GET") {
		t.Errorf("日志应包含请求方法 GET，实际: %s", logOutput)
	}
	if !strings.Contains(logOutput, "/test") {
		t.Errorf("日志应包含请求路径 /test，实际: %s", logOutput)
	}
	if !strings.Contains(logOutput, "200") {
		t.Errorf("日志应包含状态码 200，实际: %s", logOutput)
	}
}

func TestBasicAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(basicAuthMiddleware("testuser", "testpass",
		map[string]bool{"/api/v1/health": true, "/api/v1/platforms": true, "/": true}))
	r.GET("/api/v1/parse", func(c *gin.Context) {
		c.String(200, "ok")
	})
	r.GET("/api/v1/health", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// 无凭证
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/api/v1/parse", nil)
	r.ServeHTTP(w1, req1)
	if w1.Code != 401 {
		t.Errorf("无凭证应返回 401，实际: %d", w1.Code)
	}
	if !containsJSONField(w1.Body.String(), "code", ErrUnauthorized) {
		t.Errorf("401 应返回 UNAUTHORIZED 错误码，实际: %s", w1.Body.String())
	}
	if w1.Header().Get("WWW-Authenticate") == "" {
		t.Error("401 应设置 WWW-Authenticate header")
	}

	// 豁免路由
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/v1/health", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Errorf("health 端点应无需认证，实际: %d", w2.Code)
	}

	// 正确凭证
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("GET", "/api/v1/parse", nil)
	req3.SetBasicAuth("testuser", "testpass")
	r.ServeHTTP(w3, req3)
	if w3.Code != 200 {
		t.Errorf("正确凭证应返回 200，实际: %d", w3.Code)
	}

	// 错误凭证
	w4 := httptest.NewRecorder()
	req4 := httptest.NewRequest("GET", "/api/v1/parse", nil)
	req4.SetBasicAuth("testuser", "wrongpass")
	r.ServeHTTP(w4, req4)
	if w4.Code != 401 {
		t.Errorf("错误凭证应返回 401，实际: %d", w4.Code)
	}
	if !containsJSONField(w4.Body.String(), "code", ErrUnauthorized) {
		t.Errorf("错误凭证 401 应返回 UNAUTHORIZED 错误码，实际: %s", w4.Body.String())
	}
}

func TestBasicAuthMiddlewareDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(basicAuthMiddleware("", "",
		map[string]bool{"/api/v1/health": true}))
	r.GET("/api/v1/parse", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/parse", nil)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("未启用认证时应直接放行，实际: %d", w.Code)
	}
}

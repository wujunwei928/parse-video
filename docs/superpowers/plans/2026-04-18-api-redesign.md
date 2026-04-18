# API 层重设计实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 parse-video 的 HTTP API 层升级为 RESTful v1 API，包含语义化状态码、CORS、速率限制、健康检查和平台列表端点，同时保留旧路由向后兼容。

**Architecture:** 在 `cmd/` 包中新增三个文件：`response.go`（响应类型和错误码）、`middleware.go`（Recovery/CORS/限速/日志/认证中间件）、`handlers.go`（v1 处理函数 + 旧路由适配器 + 平台元数据），然后重写 `serve.go` 将所有组件组装起来。parser 包零改动。

**Tech Stack:** Go 1.24, Gin, Cobra, `golang.org/x/time/rate`（新增）

---

### Task 1: 响应类型和错误码（response.go）

**Files:**
- Create: `cmd/response.go`
- Test: `cmd/response_test.go`

- [ ] **Step 1: 编写 response.go 测试**

创建 `cmd/response_test.go`：

```go
package cmd

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSendSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"key": "value"}
	sendSuccess(c, data)

	if w.Code != 200 {
		t.Errorf("状态码应为 200，实际: %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"status":"success"`) {
		t.Errorf("响应应包含 status:success，实际: %s", body)
	}
	if !strings.Contains(body, `"key":"value"`) {
		t.Errorf("响应应包含 data 内容，实际: %s", body)
	}
}

func TestSendError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sendError(c, 400, ErrMissingParameter, "url 参数缺失")

	if w.Code != 400 {
		t.Errorf("状态码应为 400，实际: %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	if resp["status"] != "error" {
		t.Errorf("status 应为 error，实际: %v", resp["status"])
	}
	errObj := resp["error"].(map[string]any)
	if errObj["code"] != "MISSING_PARAMETER" {
		t.Errorf("error.code 应为 MISSING_PARAMETER，实际: %v", errObj["code"])
	}
}

func TestErrorCodeConstants(t *testing.T) {
	codes := []string{
		ErrMissingParameter, ErrUnsupportedURL, ErrUnsupportedSource,
		ErrIDParseNotSupported, ErrParseFailed, ErrUnauthorized,
		ErrRateLimited, ErrInternal,
	}
	for _, code := range codes {
		if code == "" {
			t.Error("错误码不应为空")
		}
		if strings.Contains(code, " ") {
			t.Errorf("错误码不应包含空格: %s", code)
		}
	}
}

func TestSendErrorContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sendError(c, 422, ErrParseFailed, "解析失败")

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type 应为 application/json，实际: %s", ct)
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go test ./cmd/ -run "TestSendSuccess|TestSendError|TestErrorCodeConstants" -v`
Expected: 编译失败（函数和常量不存在）

- [ ] **Step 3: 实现 response.go**

创建 `cmd/response.go`：

```go
package cmd

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// v1 API 错误码
const (
	ErrMissingParameter    = "MISSING_PARAMETER"
	ErrUnsupportedURL      = "UNSUPPORTED_URL"
	ErrUnsupportedSource   = "UNSUPPORTED_SOURCE"
	ErrIDParseNotSupported = "ID_PARSE_NOT_SUPPORTED"
	ErrParseFailed         = "PARSE_FAILED"
	ErrUnauthorized        = "UNAUTHORIZED"
	ErrRateLimited         = "RATE_LIMITED"
	ErrInternal            = "INTERNAL_ERROR"
)

// apiResponse v1 成功响应
type apiResponse struct {
	Status string `json:"status"`
	Data   any    `json:"data"`
}

// apiErrorResponse v1 错误响应
type apiErrorResponse struct {
	Status string    `json:"status"`
	Error  apiError  `json:"error"`
}

// apiError 错误详情
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// sendSuccess 发送 v1 成功响应
func sendSuccess(c *gin.Context, data any) {
	c.JSON(http.StatusOK, apiResponse{Status: "success", Data: data})
}

// sendError 发送 v1 错误响应
func sendError(c *gin.Context, httpStatus int, code string, message string) {
	c.JSON(httpStatus, apiErrorResponse{
		Status: "error",
		Error:  apiError{Code: code, Message: message},
	})
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go test ./cmd/ -run "TestSendSuccess|TestSendError|TestErrorCodeConstants" -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add cmd/response.go cmd/response_test.go
git commit -m "feat(api): 添加 v1 响应类型和错误码定义"
```

---

### Task 2: Recovery 和 CORS 中间件（middleware.go）

**Files:**
- Create: `cmd/middleware.go`
- Create: `cmd/middleware_test.go`

- [ ] **Step 1: 编写 Recovery 和 CORS 测试**

创建 `cmd/middleware_test.go`（先只含 Recovery 和 CORS 部分）：

```go
package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
	return containsString(body, `"`+field+`":"`+value+`"`)
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go test ./cmd/ -run "TestRecoveryMiddleware|TestCORSMiddleware" -v`
Expected: 编译失败（函数不存在）

- [ ] **Step 3: 实现 Recovery 和 CORS 中间件**

创建 `cmd/middleware.go`：

```go
package cmd

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// recoveryMiddleware 捕获 panic，返回 500 INTERNAL_ERROR
func recoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				sendError(c, http.StatusInternalServerError, ErrInternal, "服务器内部错误")
				c.Abort()
			}
		}()
		c.Next()
	}
}

// corsMiddleware 处理跨域请求
func corsMiddleware(allowedOrigins string) gin.HandlerFunc {
	origins := parseOrigins(allowedOrigins)
	allowAll := len(origins) == 1 && origins[0] == "*"

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" && originInList(origin, origins) {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func parseOrigins(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func originInList(origin string, list []string) bool {
	for _, o := range list {
		if o == origin {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: 安装 x/time 依赖**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go get golang.org/x/time/rate`

- [ ] **Step 5: 运行测试验证通过**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go test ./cmd/ -run "TestRecoveryMiddleware|TestCORSMiddleware" -v`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add cmd/middleware.go cmd/middleware_test.go go.mod go.sum
git commit -m "feat(api): 添加 Recovery 和 CORS 中间件"
```

---

### Task 3: 速率限制中间件

**Files:**
- Modify: `cmd/middleware.go`
- Modify: `cmd/middleware_test.go`

- [ ] **Step 1: 编写速率限制测试**

在 `cmd/middleware_test.go` 末尾追加：

```go
func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// 2 次/分钟
	limiter := newIPRateLimiter(2)
	r := gin.New()
	r.Use(rateLimitMiddleware(limiter, "/api/v1/health"))
	r.GET("/api/v1/parse", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// 第一次请求应成功
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/api/v1/parse", nil)
	req1.RemoteAddr = "1.2.3.4:1234"
	r.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Errorf("第一次请求应成功，实际: %d", w1.Code)
	}

	// 第二次请求（超限）应返回 429
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
	if retryAfter == "" {
		t.Error("应设置 Retry-After header")
	}
}

func TestRateLimitExemptHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := newIPRateLimiter(1) // 1 次/分钟
	r := gin.New()
	r.Use(rateLimitMiddleware(limiter, "/api/v1/health"))
	r.GET("/api/v1/health", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// 连续请求 health 端点不受限
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

	// 不同 IP 各自独立计数
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
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go test ./cmd/ -run "TestRateLimit" -v`
Expected: 编译失败

- [ ] **Step 3: 实现速率限制**

在 `cmd/middleware.go` 末尾追加：

```go
// ipRateLimiter 基于 IP 的速率限制器
type ipRateLimiter struct {
	visitors sync.Map
	rate     rate.Limit
	burst    int
}

type visitorEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// newIPRateLimiter 创建 IP 速率限制器
func newIPRateLimiter(rpm int) *ipRateLimiter {
	rl := &ipRateLimiter{
		rate:  rate.Every(time.Minute / time.Duration(rpm)),
		burst: 1,
	}
	// 后台清理过期条目
	go rl.cleanup()
	return rl
}

func (l *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	now := time.Now()
	if v, ok := l.visitors.Load(ip); ok {
		entry := v.(*visitorEntry)
		entry.lastSeen = now
		return entry.limiter
	}
	entry := &visitorEntry{
		limiter:  rate.NewLimiter(l.rate, l.burst),
		lastSeen: now,
	}
	l.visitors.Store(ip, entry)
	return entry.limiter
}

func (l *ipRateLimiter) cleanup() {
	for {
		time.Sleep(10 * time.Minute)
		threshold := time.Now().Add(-30 * time.Minute)
		l.visitors.Range(func(key, value any) bool {
			entry := value.(*visitorEntry)
			if entry.lastSeen.Before(threshold) {
				l.visitors.Delete(key)
			}
			return true
		})
	}
}

func (l *ipRateLimiter) retryAfterSeconds() int {
	// 等待时间 = 60 / rpm，最少 1 秒
	s := int(60.0 / float64(l.rate))
	if s < 1 {
		s = 1
	}
	return s
}

// rateLimitMiddleware 基于 IP 的速率限制
func rateLimitMiddleware(limiter *ipRateLimiter, exemptPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 豁免路径
		if c.Request.URL.Path == exemptPath {
			c.Next()
			return
		}

		ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil {
			ip = c.Request.RemoteAddr
		}

		if !limiter.getLimiter(ip).Allow() {
			c.Header("Retry-After", fmt.Sprintf("%d", limiter.retryAfterSeconds()))
			sendError(c, http.StatusTooManyRequests, ErrRateLimited, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go test ./cmd/ -run "TestRateLimit" -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add cmd/middleware.go cmd/middleware_test.go
git commit -m "feat(api): 添加基于 IP 的速率限制中间件"
```

---

### Task 4: 请求日志和 Basic Auth 中间件

**Files:**
- Modify: `cmd/middleware.go`
- Modify: `cmd/middleware_test.go`

- [ ] **Step 1: 编写日志和认证测试**

在 `cmd/middleware_test.go` 末尾追加：

```go
func TestRequestLogMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(requestLogMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("请求应成功，实际: %d", w.Code)
	}
	// 日志中间件不影响响应内容，只验证请求正常通过
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

	// 无凭证访问受保护路由
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/api/v1/parse", nil)
	r.ServeHTTP(w1, req1)
	if w1.Code != 401 {
		t.Errorf("无凭证应返回 401，实际: %d", w1.Code)
	}

	// 豁免路由无需认证
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/v1/health", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Errorf("health 端点应无需认证，实际: %d", w2.Code)
	}

	// 正确凭证访问受保护路由
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
}

func TestBasicAuthMiddlewareDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// 空用户名密码 = 不启用
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
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go test ./cmd/ -run "TestRequestLogMiddleware|TestBasicAuthMiddleware" -v`
Expected: 编译失败

- [ ] **Step 3: 实现请求日志和 Basic Auth**

在 `cmd/middleware.go` 末尾追加：

```go
// requestLogMiddleware 结构化请求日志
func requestLogMiddleware() gin.HandlerFunc {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Printf("%s %s %d %s",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			time.Since(start).Round(time.Microsecond),
		)
	}
}

// basicAuthMiddleware 自定义 Basic Auth，返回 v1 错误格式
func basicAuthMiddleware(username, password string, exemptPaths map[string]bool) gin.HandlerFunc {
	if username == "" || password == "" {
		// 未配置认证，直接放行
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		// 豁免路径
		if exemptPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		user, pass, hasAuth := c.Request.BasicAuth()
		if !hasAuth || user != username || pass != password {
			// v1 路由返回 JSON 格式，旧路由由浏览器处理
			c.Header("WWW-Authenticate", `Basic realm="parse-video"`)
			sendError(c, http.StatusUnauthorized, ErrUnauthorized, "认证失败")
			c.Abort()
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go test ./cmd/ -run "TestRequestLogMiddleware|TestBasicAuthMiddleware" -v`
Expected: PASS

- [ ] **Step 5: 运行全部中间件测试**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go test ./cmd/ -run "TestRecoveryMiddleware|TestCORS|TestRateLimit|TestRequestLog|TestBasicAuth" -v`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add cmd/middleware.go cmd/middleware_test.go
git commit -m "feat(api): 添加请求日志和 Basic Auth 中间件"
```

---

### Task 5: v1 处理函数 — 健康检查和平台列表（handlers.go）

**Files:**
- Create: `cmd/handlers.go`
- Create: `cmd/handlers_test.go`

- [ ] **Step 1: 编写健康检查和平台列表测试**

创建 `cmd/handlers_test.go`：

```go
package cmd

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/health", nil)

	healthHandler(c)

	if w.Code != 200 {
		t.Errorf("health 应返回 200，实际: %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"status":"ok"`) {
		t.Errorf("health 应返回 status:ok，实际: %s", body)
	}
	if !strings.Contains(body, `"platforms"`) {
		t.Errorf("health 应包含 platforms 字段，实际: %s", body)
	}
}

func TestPlatformsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/platforms", nil)

	platformsHandler(c)

	if w.Code != 200 {
		t.Errorf("platforms 应返回 200，实际: %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	if resp["status"] != "success" {
		t.Errorf("status 应为 success，实际: %v", resp["status"])
	}
	data := resp["data"].([]any)
	if len(data) == 0 {
		t.Error("平台列表不应为空")
	}
	// 验证按字母序排列
	first := data[0].(map[string]any)
	if first["source"] != "acfun" {
		t.Errorf("第一个平台应为 acfun（字母序），实际: %v", first["source"])
	}
	// 验证 douyin 支持 ID 解析
	for _, p := range data {
		pm := p.(map[string]any)
		if pm["source"] == "douyin" {
			if pm["id_parse"] != true {
				t.Errorf("douyin 应支持 ID 解析")
			}
		}
		if pm["source"] == "kuaishou" {
			if pm["id_parse"] != false {
				t.Errorf("kuaishou 不应支持 ID 解析")
			}
		}
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go test ./cmd/ -run "TestHealthHandler|TestPlatformsHandler" -v`
Expected: 编译失败

- [ ] **Step 3: 实现 handlers.go（健康检查 + 平台列表 + 平台元数据）**

创建 `cmd/handlers.go`：

```go
package cmd

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wujunwei928/parse-video/parser"
	"github.com/wujunwei928/parse-video/utils"
)

// platformNames 平台显示名称映射（按 source 字母序）
var platformNames = map[string]string{
	"acfun":        "AcFun",
	"bilibili":     "哔哩哔哩",
	"doupai":       "逗拍",
	"douyin":       "抖音",
	"haokan":       "好看视频",
	"huoshan":      "火山",
	"huya":         "虎牙",
	"kuaishou":     "快手",
	"lishipin":     "梨视频",
	"lvzhou":       "绿洲",
	"meipai":       "美拍",
	"pipigaoxiao":  "皮皮搞笑",
	"pipixia":      "皮皮虾",
	"quanmin":      "度小视",
	"quanminkge":   "全民K歌",
	"redbook":      "小红书",
	"sixroom":      "六间房",
	"twitter":      "X/Twitter",
	"weibo":        "微博",
	"weishi":       "微视",
	"xigua":        "西瓜视频",
	"xinpianchang": "新片场",
	"zuiyou":       "最右",
}

// platformInfo 平台信息
type platformInfo struct {
	Source   string `json:"source"`
	Name     string `json:"name"`
	URLParse bool   `json:"url_parse"`
	IDParse  bool   `json:"id_parse"`
}

// healthHandler 健康检查
func healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "ok",
		"version":   Version,
		"platforms": len(parser.VideoSourceInfoMapping),
	})
}

// platformsHandler 支持平台列表
func platformsHandler(c *gin.Context) {
	platforms := make([]platformInfo, 0, len(parser.VideoSourceInfoMapping))
	for source := range parser.VideoSourceInfoMapping {
		info := parser.VideoSourceInfoMapping[source]
		name := source
		if n, ok := platformNames[source]; ok {
			name = n
		}
		platforms = append(platforms, platformInfo{
			Source:   source,
			Name:     name,
			URLParse: info.VideoShareUrlParser != nil,
			IDParse:  info.VideoIdParser != nil,
		})
	}
	sort.Slice(platforms, func(i, j int) bool {
		return platforms[i].Source < platforms[j].Source
	})
	sendSuccess(c, platforms)
}

// v1ParseURLHandler v1 分享链接解析
func v1ParseURLHandler(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		sendError(c, 400, ErrMissingParameter, "url 参数缺失")
		return
	}

	// URL 提取预验证
	extractedURL, err := utils.RegexpMatchUrlFromString(url)
	if err != nil {
		sendError(c, 400, ErrUnsupportedURL, "无法从输入中提取有效链接")
		return
	}

	// 平台域名匹配预验证
	if !matchPlatform(extractedURL) {
		sendError(c, 400, ErrUnsupportedURL, "该链接无法识别对应平台")
		return
	}

	info, err := parser.ParseVideoShareUrlByRegexp(url)
	if err != nil {
		sendError(c, 422, ErrParseFailed, err.Error())
		return
	}
	sendSuccess(c, info)
}

// v1ParseIDHandler v1 视频 ID 解析
func v1ParseIDHandler(c *gin.Context) {
	source := c.Param("source")
	videoID := c.Param("video_id")

	info, exists := parser.VideoSourceInfoMapping[source]
	if !exists {
		sendError(c, 400, ErrUnsupportedSource, "未知的平台: "+source)
		return
	}
	if info.VideoIdParser == nil {
		sendError(c, 400, ErrIDParseNotSupported, "该平台暂不支持视频 ID 解析")
		return
	}

	parseInfo, err := parser.ParseVideoId(source, videoID)
	if err != nil {
		sendError(c, 422, ErrParseFailed, err.Error())
		return
	}
	sendSuccess(c, parseInfo)
}

// matchPlatform 检查 URL 是否匹配已知平台域名
func matchPlatform(url string) bool {
	for _, sourceInfo := range parser.VideoSourceInfoMapping {
		for _, domain := range sourceInfo.VideoShareUrlDomain {
			if strings.Contains(url, domain) {
				return true
			}
		}
	}
	return false
}

// legacyParseURLHandler 旧路由适配器：分享链接解析
func legacyParseURLHandler(c *gin.Context) {
	url := c.Query("url")
	parseRes, err := parser.ParseVideoShareUrlByRegexp(url)
	if err != nil {
		c.JSON(200, gin.H{"code": 201, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 200, "msg": "解析成功", "data": parseRes})
}

// legacyParseIDHandler 旧路由适配器：视频 ID 解析
func legacyParseIDHandler(c *gin.Context) {
	source := c.Query("source")
	videoID := c.Query("video_id")
	parseRes, err := parser.ParseVideoId(source, videoID)
	if err != nil {
		c.JSON(200, gin.H{"code": 201, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 200, "msg": "解析成功", "data": parseRes})
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go test ./cmd/ -run "TestHealthHandler|TestPlatformsHandler" -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add cmd/handlers.go cmd/handlers_test.go
git commit -m "feat(api): 添加健康检查和平台列表处理函数"
```

---

### Task 6: v1 解析处理函数测试

**Files:**
- Modify: `cmd/handlers_test.go`

- [ ] **Step 1: 编写 v1 解析处理函数测试**

在 `cmd/handlers_test.go` 末尾追加：

```go
func TestV1ParseURLHandlerMissingURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/parse", nil)

	v1ParseURLHandler(c)

	if w.Code != 400 {
		t.Errorf("缺少 url 应返回 400，实际: %d", w.Code)
	}
	if !containsJSONField(w.Body.String(), "code", ErrMissingParameter) {
		t.Errorf("应返回 MISSING_PARAMETER，实际: %s", w.Body.String())
	}
}

func TestV1ParseURLHandlerInvalidURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/parse?url=not-a-url", nil)

	v1ParseURLHandler(c)

	if w.Code != 400 {
		t.Errorf("无效 URL 应返回 400，实际: %d", w.Code)
	}
	if !containsJSONField(w.Body.String(), "code", ErrUnsupportedURL) {
		t.Errorf("应返回 UNSUPPORTED_URL，实际: %s", w.Body.String())
	}
}

func TestV1ParseURLHandlerUnknownDomain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/parse?url=https://unknown-platform.com/video/123", nil)

	v1ParseURLHandler(c)

	if w.Code != 400 {
		t.Errorf("未知平台应返回 400，实际: %d", w.Code)
	}
	if !containsJSONField(w.Body.String(), "code", ErrUnsupportedURL) {
		t.Errorf("应返回 UNSUPPORTED_URL，实际: %s", w.Body.String())
	}
}

func TestV1ParseIDHandlerUnknownSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/parse/unknown/123", nil)
	c.Params = gin.Params{{Key: "source", Value: "unknown"}, {Key: "video_id", Value: "123"}}

	v1ParseIDHandler(c)

	if w.Code != 400 {
		t.Errorf("未知平台应返回 400，实际: %d", w.Code)
	}
	if !containsJSONField(w.Body.String(), "code", ErrUnsupportedSource) {
		t.Errorf("应返回 UNSUPPORTED_SOURCE，实际: %s", w.Body.String())
	}
}

func TestV1ParseIDHandlerNoIDParse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/parse/kuaishou/123", nil)
	c.Params = gin.Params{{Key: "source", Value: "kuaishou"}, {Key: "video_id", Value: "123"}}

	v1ParseIDHandler(c)

	if w.Code != 400 {
		t.Errorf("不支持 ID 解析应返回 400，实际: %d", w.Code)
	}
	if !containsJSONField(w.Body.String(), "code", ErrIDParseNotSupported) {
		t.Errorf("应返回 ID_PARSE_NOT_SUPPORTED，实际: %s", w.Body.String())
	}
}

func TestLegacyParseURLHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/video/share/url/parse?url=not-a-url", nil)

	legacyParseURLHandler(c)

	// 旧路由始终 200
	if w.Code != 200 {
		t.Errorf("旧路由应始终 200，实际: %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != float64(201) {
		t.Errorf("解析失败 code 应为 201，实际: %v", resp["code"])
	}
}
```

- [ ] **Step 2: 运行测试验证通过**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go test ./cmd/ -run "TestV1Parse|TestLegacyParse" -v`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add cmd/handlers_test.go
git commit -m "test(api): 添加 v1 解析和旧路由适配器测试"
```

---

### Task 7: 重写 serve.go — 组装路由和中间件

**Files:**
- Modify: `cmd/serve.go`
- Modify: `cmd/cmd_test.go`（可能需要调整）

- [ ] **Step 1: 重写 serve.go**

将 `cmd/serve.go` 完整替换为：

```go
package cmd

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动 HTTP 解析服务",
	RunE:  runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetString("port")
	addr := ":" + port

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// 中间件栈：Recovery → CORS → 日志 → 速率限制 → Basic Auth
	rateLimitRPM := getEnvInt("RATE_LIMIT_RPM", 60)
	corsOrigins := getEnvDefault("CORS_ORIGINS", "*")
	username := os.Getenv("PARSE_VIDEO_USERNAME")
	password := os.Getenv("PARSE_VIDEO_PASSWORD")

	exemptPaths := map[string]bool{
		"/api/v1/health":    true,
		"/api/v1/platforms": true,
		"/":                 true,
	}

	r.Use(recoveryMiddleware())
	r.Use(corsMiddleware(corsOrigins))
	r.Use(requestLogMiddleware())
	r.Use(rateLimitMiddleware(newIPRateLimiter(rateLimitRPM), "/api/v1/health"))
	r.Use(basicAuthMiddleware(username, password, exemptPaths))

	// Web UI
	if templateFS != nil {
		tmpl, err := template.ParseFS(templateFS, "*.html")
		if err != nil {
			return fmt.Errorf("模板加载失败: %w", err)
		}
		r.SetHTMLTemplate(tmpl)
		r.GET("/", func(c *gin.Context) {
			c.HTML(200, "index.html", gin.H{
				"title": "github.com/wujunwei928/parse-video Demo",
			})
		})
	}

	// v1 API 路由
	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", healthHandler)
		v1.GET("/platforms", platformsHandler)
		v1.GET("/parse", v1ParseURLHandler)
		v1.GET("/parse/:source/:video_id", v1ParseIDHandler)
	}

	// 旧路由（向后兼容）
	r.GET("/video/share/url/parse", legacyParseURLHandler)
	r.GET("/video/id/parse", legacyParseIDHandler)

	srv := &http.Server{Addr: addr, Handler: r}
	log.Printf("服务启动，监听端口 %s", addr)

	serveErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serveErr <- fmt.Errorf("端口 %s 已被占用: %w", addr, err)
			return
		}
		serveErr <- nil
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	select {
	case err := <-serveErr:
		return err
	case <-quit:
	}

	log.Println("Shutdown Server ...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("服务器关闭超时: %w", err)
	}
	log.Println("Server exiting")
	return nil
}

func getEnvDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
```

- [ ] **Step 2: 验证编译**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go build ./...`
Expected: 编译成功

- [ ] **Step 3: 运行已有测试确认不破坏**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go test ./cmd/ -v -timeout 60s`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add cmd/serve.go
git commit -m "feat(api): 重写 serve.go，注册 v1 路由 + 中间件 + 旧路由兼容"
```

---

### Task 8: API 集成测试

**Files:**
- Create: `cmd/api_test.go`

- [ ] **Step 1: 编写 API 集成测试**

创建 `cmd/api_test.go`：

```go
package cmd

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// setupTestRouter 创建完整配置的测试路由
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(recoveryMiddleware())
	r.Use(corsMiddleware("*"))

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", healthHandler)
		v1.GET("/platforms", platformsHandler)
		v1.GET("/parse", v1ParseURLHandler)
		v1.GET("/parse/:source/:video_id", v1ParseIDHandler)
	}

	r.GET("/video/share/url/parse", legacyParseURLHandler)
	r.GET("/video/id/parse", legacyParseIDHandler)
	return r
}

func TestIntegrationHealthEndpoint(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("health 应返回 200，实际: %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("health status 应为 ok，实际: %v", resp["status"])
	}
}

func TestIntegrationPlatformsEndpoint(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/platforms", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("platforms 应返回 200，实际: %d", w.Code)
	}
}

func TestIntegrationV1ParseMissingURL(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/parse", nil)
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("缺少 url 应返回 400，实际: %d", w.Code)
	}
}

func TestIntegrationV1ParseInvalidURL(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/parse?url=just-text-no-url", nil)
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("无有效 URL 应返回 400，实际: %d", w.Code)
	}
}

func TestIntegrationV1ParseUnknownPlatform(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/parse?url=https://example.com/video/123", nil)
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("未知平台应返回 400，实际: %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	errObj := resp["error"].(map[string]any)
	if errObj["code"] != "UNSUPPORTED_URL" {
		t.Errorf("错误码应为 UNSUPPORTED_URL，实际: %v", errObj["code"])
	}
}

func TestIntegrationV1ParseIDUnknownSource(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/parse/unknown_platform/123", nil)
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("未知 source 应返回 400，实际: %d", w.Code)
	}
}

func TestIntegrationV1ParseIDNoIDParse(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/parse/redbook/123", nil)
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("不支持 ID 解析应返回 400，实际: %d", w.Code)
	}
}

func TestIntegrationLegacyRouteFormat(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/video/share/url/parse?url=invalid", nil)
	r.ServeHTTP(w, req)

	// 旧路由始终 200
	if w.Code != 200 {
		t.Errorf("旧路由应始终 200，实际: %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["code"]; !ok {
		t.Error("旧路由应包含 code 字段")
	}
	if _, ok := resp["msg"]; !ok {
		t.Error("旧路由应包含 msg 字段")
	}
}

func TestIntegrationCORSHeaders(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/api/v1/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != 204 {
		t.Errorf("OPTIONS 应返回 204，实际: %d", w.Code)
	}
}
```

- [ ] **Step 2: 运行集成测试**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go test ./cmd/ -run "TestIntegration" -v -timeout 60s`
Expected: PASS

- [ ] **Step 3: 运行全量测试**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go test ./... -timeout 60s`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add cmd/api_test.go
git commit -m "test(api): 添加 API 集成测试"
```

---

### Task 9: OpenAPI 文档

**Files:**
- Create: `api/openapi.yaml`

- [ ] **Step 1: 创建 api 目录**

Run: `mkdir -p /code/parse-video/.worktrees/cli-refactor/api`

- [ ] **Step 2: 编写 OpenAPI 文档**

创建 `api/openapi.yaml`：

```yaml
openapi: "3.0.3"
info:
  title: parse-video API
  description: 视频解析服务，支持 20+ 中国社交平台去水印解析
  version: "1.0.0"

servers:
  - url: http://localhost:8080
    description: 本地开发

paths:
  /api/v1/health:
    get:
      summary: 健康检查
      operationId: health
      responses:
        "200":
          description: 服务健康
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                    example: ok
                  version:
                    type: string
                    example: "1.0.0"
                  platforms:
                    type: integer
                    example: 23

  /api/v1/platforms:
    get:
      summary: 支持平台列表
      operationId: platforms
      responses:
        "200":
          description: 平台列表
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                    example: success
                  data:
                    type: array
                    items:
                      $ref: "#/components/schemas/Platform"

  /api/v1/parse:
    get:
      summary: 解析分享链接
      operationId: parseURL
      parameters:
        - name: url
          in: query
          required: true
          description: 视频分享链接或包含链接的分享文案
          schema:
            type: string
      responses:
        "200":
          description: 解析成功
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                    example: success
                  data:
                    $ref: "#/components/schemas/VideoParseInfo"
        "400":
          description: 参数错误或平台不支持
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
        "422":
          description: 解析失败（平台接口异常）
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"

  /api/v1/parse/{source}/{video_id}:
    get:
      summary: 根据视频 ID 解析
      operationId: parseID
      parameters:
        - name: source
          in: path
          required: true
          description: 平台标识
          schema:
            type: string
        - name: video_id
          in: path
          required: true
          description: 视频 ID
          schema:
            type: string
      responses:
        "200":
          description: 解析成功
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                    example: success
                  data:
                    $ref: "#/components/schemas/VideoParseInfo"
        "400":
          description: 参数错误或平台不支持
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
        "422":
          description: 解析失败
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"

components:
  schemas:
    VideoParseInfo:
      type: object
      properties:
        author:
          type: object
          properties:
            uid:
              type: string
            name:
              type: string
            avatar:
              type: string
        title:
          type: string
        video_url:
          type: string
        music_url:
          type: string
        cover_url:
          type: string
        images:
          type: array
          nullable: true
          items:
            type: object
            properties:
              url:
                type: string
              live_photo_url:
                type: string

    Platform:
      type: object
      properties:
        source:
          type: string
        name:
          type: string
        url_parse:
          type: boolean
        id_parse:
          type: boolean

    ErrorResponse:
      type: object
      properties:
        status:
          type: string
          example: error
        error:
          type: object
          properties:
            code:
              type: string
              enum:
                - MISSING_PARAMETER
                - UNSUPPORTED_URL
                - UNSUPPORTED_SOURCE
                - ID_PARSE_NOT_SUPPORTED
                - PARSE_FAILED
                - UNAUTHORIZED
                - RATE_LIMITED
                - INTERNAL_ERROR
            message:
              type: string
```

- [ ] **Step 3: 提交**

```bash
git add api/openapi.yaml
git commit -m "docs: 添加 OpenAPI 3.0 文档"
```

---

### Task 10: 最终验证和清理

**Files:**
- 无新文件

- [ ] **Step 1: 运行全量测试**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go test ./... -v -timeout 60s`
Expected: 全部 PASS

- [ ] **Step 2: 验证编译**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go build -o /dev/null .`
Expected: 编译成功

- [ ] **Step 3: 验证 go vet**

Run: `cd /code/parse-video/.worktrees/cli-refactor && go vet ./...`
Expected: 无警告

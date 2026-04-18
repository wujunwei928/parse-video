package cmd

import (
	"fmt"
	"io"
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

// ipRateLimiter 基于 IP 的速率限制器
type ipRateLimiter struct {
	visitors sync.Map
	rate     rate.Limit
	burst    int
	rpm      int
}

type visitorEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newIPRateLimiter(rpm int) *ipRateLimiter {
	return newIPRateLimiterWithBurst(rpm, 1)
}

func newIPRateLimiterWithBurst(rpm, burst int) *ipRateLimiter {
	rl := &ipRateLimiter{
		rate:  rate.Every(time.Minute / time.Duration(rpm)),
		burst: burst,
		rpm:   rpm,
	}
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
		l.cleanupOnce(time.Now().Add(-30 * time.Minute))
	}
}

func (l *ipRateLimiter) cleanupOnce(threshold time.Time) {
	l.visitors.Range(func(key, value any) bool {
		entry := value.(*visitorEntry)
		if entry.lastSeen.Before(threshold) {
			l.visitors.Delete(key)
		}
		return true
	})
}

func (l *ipRateLimiter) retryAfterSeconds() int {
	s := int(float64(60)/float64(l.rpm) + 0.5)
	if s < 1 {
		s = 1
	}
	return s
}

func rateLimitMiddleware(limiter *ipRateLimiter, exemptPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
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

// requestLogMiddleware 结构化请求日志（输出到 stderr）
func requestLogMiddleware() gin.HandlerFunc {
	return requestLogMiddlewareWithWriter(os.Stderr)
}

// requestLogMiddlewareWithWriter 可指定输出目标的日志中间件（测试用）
func requestLogMiddlewareWithWriter(w io.Writer) gin.HandlerFunc {
	logger := log.New(w, "", log.LstdFlags)
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
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		if exemptPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		user, pass, hasAuth := c.Request.BasicAuth()
		if !hasAuth || user != username || pass != password {
			c.Header("WWW-Authenticate", `Basic realm="parse-video"`)
			sendError(c, http.StatusUnauthorized, ErrUnauthorized, "认证失败")
			c.Abort()
			return
		}
		c.Next()
	}
}

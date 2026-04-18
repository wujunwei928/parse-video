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
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

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
	stop     chan struct{}
}

type visitorEntry struct {
	limiter  *rate.Limiter
	lastSeen atomic.Int64
}

func newIPRateLimiter(rpm int) *ipRateLimiter {
	return newIPRateLimiterWithBurst(rpm, 1)
}

func newIPRateLimiterWithBurst(rpm, burst int) *ipRateLimiter {
	rl := &ipRateLimiter{
		rate:  rate.Every(time.Minute / time.Duration(rpm)),
		burst: burst,
		rpm:   rpm,
		stop:  make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

func (l *ipRateLimiter) getLimiter(ip string) *rate.Limiter {
	now := time.Now().UnixNano()
	if v, ok := l.visitors.Load(ip); ok {
		entry := v.(*visitorEntry)
		entry.lastSeen.Store(now)
		return entry.limiter
	}
	entry := &visitorEntry{
		limiter: rate.NewLimiter(l.rate, l.burst),
	}
	entry.lastSeen.Store(now)
	l.visitors.Store(ip, entry)
	return entry.limiter
}

func (l *ipRateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.cleanupOnce(time.Now().Add(-30 * time.Minute))
		case <-l.stop:
			return
		}
	}
}

func (l *ipRateLimiter) cleanupOnce(threshold time.Time) {
	thresholdNano := threshold.UnixNano()
	l.visitors.Range(func(key, value any) bool {
		entry := value.(*visitorEntry)
		if entry.lastSeen.Load() < thresholdNano {
			l.visitors.Delete(key)
		}
		return true
	})
}

// Stop 终止清理 goroutine
func (l *ipRateLimiter) Stop() {
	close(l.stop)
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

func requestLogMiddleware() gin.HandlerFunc {
	return requestLogMiddlewareWithWriter(os.Stderr)
}

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

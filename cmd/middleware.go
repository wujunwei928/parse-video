package cmd

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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

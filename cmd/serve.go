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
	if err != nil || n <= 0 {
		return defaultVal
	}
	return n
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

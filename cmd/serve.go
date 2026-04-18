package cmd

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"

	"github.com/wujunwei928/parse-video/parser"
)

type httpResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动 HTTP 解析服务",
	RunE:  runServe,
}

func runServe(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetString("port")
	addr := ":" + port

	r := gin.Default()

	username := os.Getenv("PARSE_VIDEO_USERNAME")
	password := os.Getenv("PARSE_VIDEO_PASSWORD")
	if username != "" && password != "" {
		r.Use(gin.BasicAuth(gin.Accounts{
			username: password,
		}))
	}

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

	r.GET("/video/share/url/parse", makeParseHandler(func(c *gin.Context) (any, error) {
		return parser.ParseVideoShareUrlByRegexp(c.Query("url"))
	}))

	r.GET("/video/id/parse", makeParseHandler(func(c *gin.Context) (any, error) {
		return parser.ParseVideoId(c.Query("source"), c.Query("video_id"))
	}))

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

func makeParseHandler(parseFunc func(c *gin.Context) (any, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		parseRes, err := parseFunc(c)
		if err != nil {
			c.JSON(http.StatusOK, httpResponse{Code: 201, Msg: err.Error()})
			return
		}
		c.JSON(http.StatusOK, httpResponse{Code: 200, Msg: "解析成功", Data: parseRes})
	}
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

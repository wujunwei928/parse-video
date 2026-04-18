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
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
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

	if os.Getenv("PARSE_VIDEO_USERNAME") != "" && os.Getenv("PARSE_VIDEO_PASSWORD") != "" {
		r.Use(gin.BasicAuth(gin.Accounts{
			os.Getenv("PARSE_VIDEO_USERNAME"): os.Getenv("PARSE_VIDEO_PASSWORD"),
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

	r.GET("/video/share/url/parse", func(c *gin.Context) {
		paramUrl := c.Query("url")
		parseRes, err := parser.ParseVideoShareUrlByRegexp(paramUrl)
		jsonRes := httpResponse{Code: 200, Msg: "解析成功", Data: parseRes}
		if err != nil {
			jsonRes = httpResponse{Code: 201, Msg: err.Error()}
		}
		c.JSON(http.StatusOK, jsonRes)
	})

	r.GET("/video/id/parse", func(c *gin.Context) {
		videoId := c.Query("video_id")
		source := c.Query("source")
		parseRes, err := parser.ParseVideoId(source, videoId)
		jsonRes := httpResponse{Code: 200, Msg: "解析成功", Data: parseRes}
		if err != nil {
			jsonRes = httpResponse{Code: 201, Msg: err.Error()}
		}
		c.JSON(200, jsonRes)
	})

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

func init() {
	rootCmd.AddCommand(serveCmd)
}

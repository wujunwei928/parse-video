package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"time"

	"github.com/wujunwei928/parse-video/parser"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/video/share/url/parse", func(c *gin.Context) {
		urlReg := regexp.MustCompile(`https?:\/\/\S+`)
		videoShareUrl := urlReg.FindString(c.Query("url"))

		parseRes, _ := parser.ParseVideoShareUrl(videoShareUrl)

		c.JSONP(http.StatusOK, parseRes)
	})

	r.GET("/video/id/parse", func(c *gin.Context) {
		videoId := c.Query("video_id")
		source := c.Query("source")

		parseRes, err := parser.ParseVideoId(videoId, source)
		if err != nil {
			c.JSON(500, err.Error())
		}

		c.JSON(200, parseRes)
	})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		// 服务连接
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// 等待中断信号以优雅地关闭服务器 (设置 5 秒的超时时间)
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
	log.Println("Server exiting")

}

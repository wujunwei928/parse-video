package main

import (
	"context"
	"embed"
	"flag"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wujunwei928/parse-video/mcp"
	"github.com/wujunwei928/parse-video/parser"
)

type HttpResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

//go:embed templates/*
var files embed.FS

func main() {
	// Parse command line flags
	mcpMode := flag.Bool("mcp", false, "Run as MCP server with stdio transport")
	mcpSSEMode := flag.Bool("mcp-sse", false, "Run as MCP server with SSE transport")
	mcpPort := flag.Int("mcp-port", 8081, "Port for MCP SSE server")
	httpMode := flag.Bool("http", false, "Run as HTTP server only")
	bothMode := flag.Bool("both", true, "Run both HTTP and MCP servers (default, uses SSE for MCP)")
	flag.Parse()

	// For mixed mode, default to SSE if not explicitly specified
	useSSEForMixed := *bothMode && !*mcpMode && !*mcpSSEMode

	// Determine run mode
	runMCP := *mcpMode || *mcpSSEMode || *bothMode
	runHTTP := *httpMode || *bothMode || (!runMCP && !*httpMode)

	// Start servers based on mode
	if runMCP && runHTTP {
		// Mixed mode: start both servers
		log.Println("Starting in mixed mode: both HTTP and MCP servers")

		// Start MCP server in background
		go func() {
			if *mcpSSEMode || useSSEForMixed {
				log.Printf("Starting MCP server with SSE transport on port %d", *mcpPort)
				if err := mcp.RunMCPServerWithSSE(*mcpPort); err != nil {
					log.Printf("MCP SSE server error: %v", err)
				}
			} else {
				log.Println("Starting MCP server with stdio transport")
				if err := mcp.RunMCPServerWithStdio(); err != nil {
					log.Printf("MCP stdio server error: %v", err)
				}
			}
		}()

		// Start HTTP server in foreground
		startHTTPServer()
	} else if runMCP {
		// MCP only mode
		if *mcpSSEMode {
			log.Printf("Starting MCP server with SSE transport on port %d", *mcpPort)
			if err := mcp.RunMCPServerWithSSE(*mcpPort); err != nil {
				log.Fatalf("Failed to start MCP SSE server: %v", err)
			}
		} else {
			log.Println("Starting MCP server with stdio transport")
			if err := mcp.RunMCPServerWithStdio(); err != nil {
				log.Fatalf("Failed to start MCP stdio server: %v", err)
			}
		}
	} else if runHTTP {
		// HTTP only mode
		startHTTPServer()
	}
}

func startHTTPServer() {
	r := gin.Default()

	// 根据相关环境变量，确定是否需要使用basic auth中间件验证用户
	if os.Getenv("PARSE_VIDEO_USERNAME") != "" && os.Getenv("PARSE_VIDEO_PASSWORD") != "" {
		r.Use(gin.BasicAuth(gin.Accounts{
			os.Getenv("PARSE_VIDEO_USERNAME"): os.Getenv("PARSE_VIDEO_PASSWORD"),
		}))
	}

	sub, err := fs.Sub(files, "templates")
	if err != nil {
		panic(err)
	}
	tmpl := template.Must(template.ParseFS(sub, "*.html"))
	r.SetHTMLTemplate(tmpl)
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{
			"title": "github.com/wujunwei928/parse-video Demo",
		})
	})

	r.GET("/video/share/url/parse", func(c *gin.Context) {
		paramUrl := c.Query("url")
		parseRes, err := parser.ParseVideoShareUrlByRegexp(paramUrl)
		jsonRes := HttpResponse{
			Code: 200,
			Msg:  "解析成功",
			Data: parseRes,
		}
		if err != nil {
			jsonRes = HttpResponse{
				Code: 201,
				Msg:  err.Error(),
			}
		}

		c.JSON(http.StatusOK, jsonRes)
	})

	r.GET("/video/id/parse", func(c *gin.Context) {
		videoId := c.Query("video_id")
		source := c.Query("source")

		parseRes, err := parser.ParseVideoId(source, videoId)
		jsonRes := HttpResponse{
			Code: 200,
			Msg:  "解析成功",
			Data: parseRes,
		}
		if err != nil {
			jsonRes = HttpResponse{
				Code: 201,
				Msg:  err.Error(),
			}
		}

		c.JSON(200, jsonRes)
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

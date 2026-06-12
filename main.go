package main

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/wujunwei928/parse-video/cmd"
)

//go:embed templates/* all:static
var assetsFS embed.FS

func main() {
	normalizeArgs()

	tmplSub, err := fs.Sub(assetsFS, "templates")
	if err != nil {
		log.Fatalf("模板子树加载失败: %v", err)
	}
	staticSub, err := fs.Sub(assetsFS, "static")
	if err != nil {
		log.Fatalf("静态资源子树加载失败: %v", err)
	}

	cmd.SetTemplates(tmplSub)
	cmd.SetStatic(staticSub)
	cmd.Execute()
}

func normalizeArgs() {
	for i, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") {
			name := strings.TrimPrefix(arg, "-")
			if name == "port" || name == "version" {
				os.Args[i+1] = "--" + name
			}
		}
	}
}

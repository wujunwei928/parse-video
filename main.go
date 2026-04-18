package main

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/wujunwei928/parse-video/cmd"
)

//go:embed templates/*
var templateFS embed.FS

func main() {
	normalizeArgs()
	sub, err := fs.Sub(templateFS, "templates")
	if err != nil {
		log.Fatal(err)
	}
	cmd.SetTemplates(sub)
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

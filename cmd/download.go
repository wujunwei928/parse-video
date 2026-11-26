package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/wujunwei928/parse-video/parser"
)

var invalidFileCharRe = regexp.MustCompile("[<>:\"/\\\\|?*\\x00-\\x1f]")

const douyinMediaUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/130.0.0.0 Safari/537.36"

// sanitizeFilename 清理文件名中的非法字符
func sanitizeFilename(name string) string {
	name = invalidFileCharRe.ReplaceAllString(name, "_")
	name = strings.Trim(strings.TrimSpace(name), " .")
	const maxNameBytes = 200
	if len(name) > maxNameBytes {
		end := maxNameBytes
		for !utf8.RuneStart(name[end]) {
			end--
		}
		name = strings.TrimRight(name[:end], " .")
	}
	if name == "" || !filepath.IsLocal(name+".mp4") {
		return "download"
	}
	return name
}

var allowedMediaExtensions = map[string][]string{
	".mp4": {".mp4", ".m4v", ".mov", ".mkv", ".webm", ".flv", ".avi", ".ts", ".m3u8", ".mpd"},
	".jpg": {".jpg", ".jpeg", ".png", ".gif", ".webp", ".avif", ".bmp", ".heic", ".heif"},
	".mp3": {".mp3", ".m4a", ".aac", ".wav", ".flac", ".ogg", ".opus", ".wma", ".webm"},
}

func mediaExtension(rawURL, fallback string) string {
	if target, err := url.Parse(rawURL); err == nil {
		ext := strings.ToLower(path.Ext(target.Path))
		if slices.Contains(allowedMediaExtensions[fallback], ext) {
			return ext
		}
	}
	return fallback
}

func downloadFile(ctx context.Context, client *http.Client, fileURL, savePath string) error {
	if _, err := os.Lstat(savePath); err == nil {
		return fmt.Errorf("目标文件已存在 %s: %w", filepath.Base(savePath), fs.ErrExist)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", parser.DefaultUserAgent)
	if target, err := url.Parse(fileURL); err == nil {
		if isDouyinMediaHost(target.Hostname()) {
			setDouyinMediaHeaders(request)
		}
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", response.StatusCode)
	}

	temp, err := os.CreateTemp(filepath.Dir(savePath), ".parse-video-*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		_ = temp.Close()
		_ = os.Remove(temp.Name())
	}()

	size, err := io.Copy(temp, response.Body)
	if err != nil {
		return fmt.Errorf("读取媒体失败: %w", err)
	}
	if size == 0 {
		return errors.New("上游返回空媒体文件")
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Linking publishes the finished file atomically and fails if the name is taken.
	if err := os.Link(temp.Name(), savePath); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("目标文件已存在 %s: %w", filepath.Base(savePath), fs.ErrExist)
		}
		return fmt.Errorf("无法发布下载文件: %w", err)
	}
	return nil
}

// downloadMedia 下载解析结果中的所有媒体文件
func downloadMedia(ctx context.Context, info *parser.VideoParseInfo, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	type mediaFile struct {
		label    string
		url      string
		filename string
		optional bool
	}
	var files []mediaFile
	var mediaURLs []string
	addFile := func(label, rawURL, name, fallback string, optional bool) {
		if rawURL != "" {
			files = append(files, mediaFile{
				label: label, url: rawURL, filename: name + mediaExtension(rawURL, fallback), optional: optional,
			})
			mediaURLs = append(mediaURLs, rawURL)
		}
	}

	baseName := sanitizeFilename(info.Title)
	addFile("视频", info.VideoUrl, baseName, ".mp4", false)
	for i, img := range info.Images {
		name := fmt.Sprintf("%s_%03d", baseName, i+1)
		addFile(fmt.Sprintf("图片 %d", i+1), img.Url, name, ".jpg", false)
		addFile("LivePhoto", img.LivePhotoUrl, name+"_live", ".mp4", true)
	}
	addFile("封面", info.CoverUrl, baseName+"_cover", ".jpg", true)
	addFile("音乐", info.MusicUrl, baseName+"_music", ".mp3", true)

	client := parser.NewMediaClient(mediaURLs)
	defer client.CloseIdleConnections()
	count := 0
	for _, file := range files {
		fmt.Fprintf(os.Stderr, "下载%s: %s\n", file.label, file.filename)
		if err := downloadFile(ctx, client, file.url, filepath.Join(outputDir, file.filename)); err != nil {
			if !file.optional || errors.Is(err, fs.ErrExist) || ctx.Err() != nil {
				return fmt.Errorf("%s下载失败: %w", file.label, err)
			}
			fmt.Fprintf(os.Stderr, "警告: %s下载失败: %v\n", file.label, err)
			continue
		}
		count++
	}

	if count > 0 {
		fmt.Fprintf(os.Stderr, "下载完成: %d 个文件保存到 %s\n", count, outputDir)
	} else {
		fmt.Fprintln(os.Stderr, "无可下载的媒体文件")
	}
	return nil
}

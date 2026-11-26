package cmd

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wujunwei928/parse-video/parser"
	"github.com/wujunwei928/parse-video/utils"
)

const (
	maxBatchItems            = 8
	maxBatchInputBytes       = 4096
	maxBatchJSONBodyBytes    = maxBatchItems*maxBatchInputBytes + 1024
	maxBatchFormBodyBytes    = 128 * 1024
	batchDownloadTimeout     = 10 * time.Minute
	batchParseContentType    = "application/x-ndjson; charset=utf-8"
	batchDownloadArchiveName = "parse-video-batch.zip"
)

type batchParseRequest struct {
	Items []string `json:"items"`
}

type batchParseItemResult struct {
	Index int                    `json:"index"`
	Data  *parser.VideoParseInfo `json:"data,omitempty"`
	Error string                 `json:"error,omitempty"`
}

func v1BatchParseHandler(c *gin.Context) {
	var request batchParseRequest
	if err := decodeBatchJSON(c, &request); err != nil {
		sendError(c, http.StatusBadRequest, ErrMissingParameter, "批量解析请求格式无效")
		return
	}
	if message := validateBatchInputs(request.Items); message != "" {
		sendError(c, http.StatusBadRequest, ErrUnsupportedURL, message)
		return
	}

	results := make(chan batchParseItemResult, len(request.Items))
	var workers sync.WaitGroup
	ctx := c.Request.Context()
	for index, input := range request.Items {
		workers.Add(1)
		go func(index int, input string) {
			defer workers.Done()
			results <- parseBatchItem(ctx, index, input)
		}(index, input)
	}
	go func() {
		workers.Wait()
		close(results)
	}()

	c.Header("Content-Type", batchParseContentType)
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusOK)
	encoder := json.NewEncoder(c.Writer)
	for result := range results {
		if err := encoder.Encode(result); err != nil {
			c.Abort()
			return
		}
		c.Writer.Flush()
	}
}

func v1BatchDownloadHandler(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBatchFormBodyBytes)
	if err := c.Request.ParseForm(); err != nil {
		sendError(c, http.StatusBadRequest, ErrUnsupportedURL, "批量下载请求格式无效")
		return
	}
	items := c.Request.PostForm["items"]
	if message := validateBatchInputs(items); message != "" {
		sendError(c, http.StatusBadRequest, ErrUnsupportedURL, message)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), batchDownloadTimeout)
	defer cancel()
	results := parseBatchItems(ctx, items)

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": batchDownloadArchiveName}))
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusOK)

	archive := zip.NewWriter(c.Writer)
	manifest := make([]string, 0, len(results)+1)
	manifest = append(manifest, "视频批量下载结果")
	usedNames := make(map[string]int, len(results))
	for _, result := range results {
		itemNumber := result.Index + 1
		if result.Error != "" || result.Data == nil {
			message := result.Error
			if message == "" {
				message = "解析失败，请确认链接后重试"
			}
			manifest = append(manifest, fmt.Sprintf("第 %d 项：失败（%s）", itemNumber, message))
			continue
		}
		if result.Data.VideoUrl == "" {
			manifest = append(manifest, fmt.Sprintf("第 %d 项：没有可下载的视频地址", itemNumber))
			continue
		}

		file, cleanup, err := downloadBatchVideo(ctx, result.Data.VideoUrl)
		if err != nil {
			manifest = append(manifest, fmt.Sprintf("第 %d 项：下载失败（%s）", itemNumber, batchDownloadErrorMessage(ctx, err)))
			continue
		}
		filename := nextBatchFilename(sanitizeFilename(result.Data.Title)+".mp4", usedNames)
		entry, err := archive.CreateHeader(&zip.FileHeader{Name: filename, Method: zip.Deflate, Modified: time.Now()})
		if err == nil {
			_, err = io.Copy(entry, file)
		}
		cleanup()
		if err != nil {
			manifest = append(manifest, fmt.Sprintf("第 %d 项：写入压缩包失败", itemNumber))
			continue
		}
		manifest = append(manifest, fmt.Sprintf("第 %d 项：已保存为 %s", itemNumber, filename))
	}

	manifestWriter, err := archive.Create("README.txt")
	if err == nil {
		_, err = io.WriteString(manifestWriter, strings.Join(manifest, "\n")+"\n")
	}
	if err == nil {
		err = archive.Close()
	} else {
		_ = archive.Close()
	}
	if err != nil {
		c.Abort()
	}
}

func decodeBatchJSON(c *gin.Context, destination any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBatchJSONBodyBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func validateBatchInputs(items []string) string {
	if len(items) == 0 {
		return "请至少提供一个视频链接"
	}
	if len(items) > maxBatchItems {
		return fmt.Sprintf("一次最多解析 %d 个视频链接", maxBatchItems)
	}
	for index, input := range items {
		if strings.TrimSpace(input) == "" {
			return fmt.Sprintf("第 %d 项视频链接不能为空", index+1)
		}
		if len(input) > maxBatchInputBytes {
			return fmt.Sprintf("第 %d 项分享文案不能超过 %d 字节", index+1, maxBatchInputBytes)
		}
	}
	return ""
}

func parseBatchItems(ctx context.Context, items []string) []batchParseItemResult {
	results := make([]batchParseItemResult, len(items))
	var workers sync.WaitGroup
	for index, input := range items {
		workers.Add(1)
		go func(index int, input string) {
			defer workers.Done()
			results[index] = parseBatchItem(ctx, index, input)
		}(index, input)
	}
	workers.Wait()
	return results
}

func parseBatchItem(ctx context.Context, index int, input string) batchParseItemResult {
	shareURL, err := utils.RegexpMatchUrlFromString(input)
	if err != nil || !parser.IsSupportedShareURL(shareURL) {
		return batchParseItemResult{Index: index, Error: "无法识别支持的视频分享链接"}
	}
	info, err := parseVideoShareURL(ctx, input)
	if err != nil {
		return batchParseItemResult{Index: index, Error: parseErrorMessage(err)}
	}
	if info == nil {
		return batchParseItemResult{Index: index, Error: "解析失败，请确认链接后重试"}
	}
	return batchParseItemResult{Index: index, Data: info}
}

func downloadBatchVideo(ctx context.Context, mediaURL string) (*os.File, func(), error) {
	target, err := url.Parse(mediaURL)
	if err != nil || target == nil || (target.Scheme != "http" && target.Scheme != "https") || target.Hostname() == "" || target.User != nil {
		return nil, func() {}, errors.New("无效的视频下载地址")
	}
	client := parser.NewMediaClient([]string{mediaURL})
	defer client.CloseIdleConnections()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return nil, func() {}, err
	}
	request.Header.Set("User-Agent", parser.DefaultUserAgent)
	if isDouyinMediaHost(target.Hostname()) {
		setDouyinMediaHeaders(request)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, func() {}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || strings.HasPrefix(strings.ToLower(response.Header.Get("Content-Type")), "text/html") {
		return nil, func() {}, errors.New("视频源拒绝了下载请求")
	}

	file, err := os.CreateTemp("", "parse-video-batch-*.mp4")
	if err != nil {
		return nil, func() {}, err
	}
	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}
	if _, err := io.Copy(file, response.Body); err != nil {
		cleanup()
		return nil, func() {}, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		cleanup()
		return nil, func() {}, err
	}
	return file, cleanup, nil
}

func batchDownloadErrorMessage(ctx context.Context, err error) string {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "下载超时，请稍后重试"
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return "下载已取消"
	}
	return "视频源暂时无法下载，请重试"
}

func nextBatchFilename(filename string, used map[string]int) string {
	extension := ".mp4"
	base := strings.TrimSuffix(filename, extension)
	for suffix := 1; ; suffix++ {
		candidate := filename
		if suffix > 1 {
			candidate = fmt.Sprintf("%s_%d%s", base, suffix, extension)
		}
		if used[candidate] == 0 {
			used[candidate] = 1
			return candidate
		}
	}
}

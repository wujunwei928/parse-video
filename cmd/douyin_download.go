package cmd

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wujunwei928/parse-video/parser"
	"github.com/wujunwei928/parse-video/utils"
)

const maxDouyinDownloadInputBytes = 4096

func v1DouyinDownloadHandler(c *gin.Context) {
	shareText := c.Query("url")
	if shareText == "" {
		sendError(c, http.StatusBadRequest, ErrMissingParameter, "url 参数缺失")
		return
	}
	if len(shareText) > maxDouyinDownloadInputBytes {
		sendError(c, http.StatusBadRequest, ErrUnsupportedURL, "分享文案过长")
		return
	}

	shareURL, err := utils.RegexpMatchUrlFromString(shareText)
	if err != nil || !parser.IsSupportedShareURL(shareURL) {
		sendError(c, http.StatusBadRequest, ErrUnsupportedURL, "无法识别抖音分享链接")
		return
	}
	parsedShareURL, err := url.Parse(shareURL)
	if err != nil || !isDouyinShareHost(parsedShareURL.Hostname()) {
		sendError(c, http.StatusBadRequest, ErrUnsupportedURL, "该下载接口仅支持抖音链接")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()
	info, err := parser.ParseVideoShareUrlByRegexpContext(ctx, shareText)
	if err != nil {
		status, code := classifyParseError(err)
		sendError(c, status, code, parseErrorMessage(err))
		return
	}
	mediaURL, err := url.Parse(info.VideoUrl)
	if err != nil || mediaURL == nil || mediaURL.Scheme != "https" || !isDouyinCDNHost(mediaURL.Hostname()) {
		sendError(c, http.StatusBadGateway, ErrDownloadFailed, "抖音没有返回有效的视频下载地址，请重新解析")
		return
	}

	// The parser resolves the short play URL first. Only stream from the final
	// Douyin CDN host, so the browser never receives an expiring source URL.
	client := parser.NewMediaClient([]string{info.VideoUrl, "https://douyinvod.com/"})
	defer client.CloseIdleConnections()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, info.VideoUrl, nil)
	if err != nil {
		sendError(c, http.StatusBadGateway, ErrDownloadFailed, "创建抖音下载请求失败")
		return
	}
	setDouyinMediaHeaders(request)
	response, err := client.Do(request)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			sendError(c, http.StatusGatewayTimeout, ErrDownloadFailed, "抖音视频下载超时，请重新尝试")
			return
		}
		sendError(c, http.StatusBadGateway, ErrDownloadFailed, "抖音 CDN 下载失败，请重新尝试")
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		sendError(c, http.StatusBadGateway, ErrDownloadFailed, "抖音 CDN 拒绝了下载请求，请重新解析后重试")
		return
	}
	if strings.HasPrefix(strings.ToLower(response.Header.Get("Content-Type")), "text/html") {
		sendError(c, http.StatusBadGateway, ErrDownloadFailed, "抖音 CDN 返回了错误页面，请重新解析后重试")
		return
	}

	filename := sanitizeFilename(info.Title) + ".mp4"
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	if length := response.Header.Get("Content-Length"); length != "" {
		c.Header("Content-Length", length)
	}
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, response.Body); err != nil {
		// The client may cancel a large download; avoid returning the signed URL
		// or transport details in logs or a response after streaming has started.
		c.Abort()
	}
}

func isDouyinShareHost(host string) bool {
	switch strings.ToLower(strings.TrimSuffix(host, ".")) {
	case "v.douyin.com", "www.douyin.com", "www.iesdouyin.com":
		return true
	default:
		return false
	}
}

func isDouyinMediaHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return isDouyinCDNHost(host) || host == "aweme.snssdk.com"
}

func isDouyinCDNHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return host == "douyinvod.com" || strings.HasSuffix(host, ".douyinvod.com")
}

func setDouyinMediaHeaders(request *http.Request) {
	request.Header.Set("User-Agent", douyinMediaUserAgent)
	request.Header.Set("Referer", "https://www.douyin.com/")
	request.Header.Set("Accept", "video/*,application/octet-stream;q=0.9,*/*;q=0.8")
}

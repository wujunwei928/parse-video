package parser

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

// cctvVideo 央视网视频解析器
type cctvVideo struct{}

// parseShareUrl 根据分享链接解析央视网视频信息
func (c cctvVideo) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	// 请求页面 HTML 提取视频 GUID
	guid, err := c.extractGuid(shareUrl)
	if err != nil {
		return nil, fmt.Errorf("提取视频GUID失败: %w", err)
	}

	return c.parseVideoID(guid)
}

// parseVideoID 根据视频GUID解析央视网视频信息
func (c cctvVideo) parseVideoID(videoId string) (*VideoParseInfo, error) {
	if len(videoId) == 0 {
		return nil, errors.New("视频GUID不能为空")
	}

	// 央视网视频信息 API
	apiUrl := fmt.Sprintf(
		"https://vdn.apps.cntv.cn/api/getHttpVideoInfo.do?pid=%s",
		videoId,
	)

	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		Get(apiUrl)
	if err != nil {
		return nil, fmt.Errorf("请求央视网视频API失败: %w", err)
	}

	jsonStr := string(res.Body())

	// 检查 API 状态
	status := gjson.Get(jsonStr, "status").String()
	if status != "001" {
		msg := gjson.Get(jsonStr, "title").String()
		return nil, fmt.Errorf("央视网视频API返回错误 (status: %s, title: %s)", status, msg)
	}

	// 提取 HLS 视频播放地址
	// 注：manifest 中的 h5e/enc/enc2 高码率流在 H.264 帧级加扰，播放花屏，仅 hls_url 可正常播放
	videoUrl := gjson.Get(jsonStr, "hls_url").String()
	if len(videoUrl) == 0 {
		return nil, errors.New("未找到视频播放地址")
	}

	// 提取视频元信息
	title := gjson.Get(jsonStr, "title").String()
	coverUrl := gjson.Get(jsonStr, "image").String()
	playChannel := gjson.Get(jsonStr, "play_channel").String()

	parseRes := &VideoParseInfo{
		Title:    title,
		VideoUrl: videoUrl,
		CoverUrl: coverUrl,
		Images:   make([]ImgInfo, 0),
	}
	parseRes.Author.Name = playChannel

	return parseRes, nil
}

// extractGuid 从页面 URL 请求并提取视频 GUID
func (c cctvVideo) extractGuid(pageUrl string) (string, error) {
	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		Get(pageUrl)
	if err != nil {
		return "", fmt.Errorf("请求页面失败: %w", err)
	}

	return c.extractGuidFromHTML(string(res.Body()))
}

// extractGuidFromHTML 从 HTML 字符串中提取视频 GUID
func (c cctvVideo) extractGuidFromHTML(html string) (string, error) {
	re := regexp.MustCompile(`var\s+guid\s*=\s*"([^"]+)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) >= 2 && len(matches[1]) > 0 {
		return matches[1], nil
	}

	return "", errors.New("页面中未找到视频GUID")
}

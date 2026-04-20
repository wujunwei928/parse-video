package parser

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

// qqVidPathRe 匹配腾讯视频页面路径中的视频 ID
var qqVidPathRe = regexp.MustCompile(`/x/(?:page|cover)/(?:[^/]+/)?(\w+)\.html`)

// qqVideo 腾讯视频解析器
type qqVideo struct{}

// parseShareUrl 根据分享链接解析腾讯视频信息
func (q qqVideo) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	vid, err := q.extractVid(shareUrl)
	if err != nil {
		return nil, fmt.Errorf("提取视频ID失败: %w", err)
	}

	return q.parseVideoID(vid)
}

// parseVideoID 根据视频ID解析腾讯视频信息
func (q qqVideo) parseVideoID(videoId string) (*VideoParseInfo, error) {
	if len(videoId) == 0 {
		return nil, errors.New("视频ID不能为空")
	}

	apiUrl := fmt.Sprintf(
		"https://vv.video.qq.com/getinfo?vids=%s&platform=101001&otype=json&defn=shd",
		videoId,
	)

	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		Get(apiUrl)
	if err != nil {
		return nil, fmt.Errorf("请求腾讯视频API失败: %w", err)
	}

	body := string(res.Body())

	// 去除 JSONP 前缀 QZOutputJson= 和尾部分号
	jsonStr := strings.TrimPrefix(body, "QZOutputJson=")
	jsonStr = strings.TrimSuffix(jsonStr, ";")

	// 检查 API 级别错误
	em := gjson.Get(jsonStr, "em").Int()
	if em != 0 {
		msg := gjson.Get(jsonStr, "msg").String()
		return nil, fmt.Errorf("腾讯视频API返回错误: %s (em: %d)", msg, em)
	}

	// 检查视频列表
	viResult := gjson.Get(jsonStr, "vl.vi.0")
	if !viResult.Exists() {
		return nil, errors.New("未找到视频信息，视频可能已被删除或设为私密")
	}

	// 提取 CDN 地址
	uiResult := viResult.Get("ul.ui.0")
	if !uiResult.Exists() {
		return nil, errors.New("未找到视频CDN地址")
	}

	baseUrl := uiResult.Get("url").String()
	fn := viResult.Get("fn").String()
	fvkey := viResult.Get("fvkey").String()
	if len(baseUrl) == 0 || len(fn) == 0 || len(fvkey) == 0 {
		return nil, errors.New("视频地址信息不完整")
	}

	// 构造视频播放地址
	videoUrl := fmt.Sprintf("%s%s?vkey=%s", baseUrl, fn, fvkey)

	// 提取视频元信息
	vid := viResult.Get("vid").String()
	title := viResult.Get("ti").String()
	coverUrl := fmt.Sprintf("https://puui.qpic.cn/vpic_cover/%s/%s_hz.jpg/496", vid, vid)

	parseRes := &VideoParseInfo{
		Title:    title,
		VideoUrl: videoUrl,
		CoverUrl: coverUrl,
		Images:   make([]ImgInfo, 0),
	}

	return parseRes, nil
}

// extractVid 从 URL 中提取腾讯视频 ID
func (q qqVideo) extractVid(rawUrl string) (string, error) {
	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		return "", fmt.Errorf("URL格式无效: %w", err)
	}

	host := parsedUrl.Host

	// 移动端播放页: m.v.qq.com/x/m/play?vid={vid}
	if strings.Contains(host, "m.v.qq.com") {
		vid := parsedUrl.Query().Get("vid")
		if len(vid) > 0 {
			return vid, nil
		}
		return "", errors.New("移动端链接中未找到vid参数")
	}

	// PC端页面: v.qq.com/x/page/{vid}.html 或 v.qq.com/x/cover/{cid}/{vid}.html
	if strings.Contains(host, "v.qq.com") {
		return q.extractVidFromPath(parsedUrl.Path)
	}

	return "", fmt.Errorf("不支持的腾讯视频域名: %s", host)
}

// extractVidFromPath 从 URL 路径中提取视频 ID
func (q qqVideo) extractVidFromPath(path string) (string, error) {
	matches := qqVidPathRe.FindStringSubmatch(path)
	if len(matches) >= 2 && len(matches[1]) > 0 {
		return matches[1], nil
	}

	return "", fmt.Errorf("无法从路径 %s 中提取视频ID", path)
}

package parser

import (
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

// sohuVideo 搜狐视频解析器
type sohuVideo struct{}

// parseShareUrl 根据分享链接解析搜狐视频信息
func (s sohuVideo) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	vid, err := s.extractVid(shareUrl)
	if err != nil {
		return nil, fmt.Errorf("提取视频ID失败: %w", err)
	}

	return s.parseVideoID(vid)
}

// parseVideoID 根据视频ID解析搜狐视频信息
func (s sohuVideo) parseVideoID(videoId string) (*VideoParseInfo, error) {
	if len(videoId) == 0 {
		return nil, errors.New("视频ID不能为空")
	}

	// 搜狐视频信息API
	apiUrl := fmt.Sprintf(
		"https://api.tv.sohu.com/v4/video/info/%s.json?site=2&api_key=9854b2afa779e1a6bcdd07b217417549&sver=6.2.0",
		videoId,
	)

	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		Get(apiUrl)
	if err != nil {
		return nil, fmt.Errorf("请求搜狐视频API失败: %w", err)
	}

	jsonStr := string(res.Body())

	// 检查API状态
	status := gjson.Get(jsonStr, "status").Int()
	if status != 200 {
		msg := gjson.Get(jsonStr, "statusText").String()
		return nil, fmt.Errorf("搜狐视频API返回错误: %s (status: %d)", msg, status)
	}

	data := gjson.Get(jsonStr, "data")
	if !data.Exists() {
		return nil, errors.New("API响应中未找到视频数据")
	}

	// 提取视频播放地址
	videoUrl := data.Get("url_high_mp4").String()
	if len(videoUrl) == 0 {
		videoUrl = data.Get("download_url").String()
	}
	if len(videoUrl) == 0 {
		return nil, errors.New("未找到视频播放地址")
	}

	// 提取视频元信息
	title := data.Get("video_name").String()
	coverUrl := data.Get("originalCutCover").String()
	authorUid := data.Get("user.user_id").String()
	authorName := data.Get("user.nickname").String()
	authorAvatar := data.Get("user.small_pic").String()

	parseRes := &VideoParseInfo{
		Title:    title,
		VideoUrl: videoUrl,
		CoverUrl: coverUrl,
		Images:   make([]ImgInfo, 0),
	}
	parseRes.Author.Uid = authorUid
	parseRes.Author.Name = authorName
	parseRes.Author.Avatar = authorAvatar

	return parseRes, nil
}

// extractVid 从 URL 中提取搜狐视频 ID
func (s sohuVideo) extractVid(rawUrl string) (string, error) {
	// 匹配 tv.sohu.com/v/{base64}.html 格式
	// base64 编码部分解码后为 us/{uid}/{vid}.shtml
	base64Re := regexp.MustCompile(`/v/([A-Za-z0-9+/=]+)\.html`)
	if matches := base64Re.FindStringSubmatch(rawUrl); len(matches) >= 2 {
		decoded, err := base64.StdEncoding.DecodeString(matches[1])
		if err != nil {
			return "", fmt.Errorf("base64解码失败: %w", err)
		}
		// 从解码后的路径 us/{uid}/{vid}.shtml 中提取 vid
		return s.extractVidFromPath(string(decoded))
	}

	// 匹配 my.tv.sohu.com/us/{uid}/{vid}.shtml 格式
	if strings.Contains(rawUrl, "my.tv.sohu.com") || strings.Contains(rawUrl, "tv.sohu.com/us/") {
		// 从路径中提取 vid
		vidRe := regexp.MustCompile(`/us/\d+/(\d+)\.shtml`)
		if matches := vidRe.FindStringSubmatch(rawUrl); len(matches) >= 2 {
			return matches[1], nil
		}
	}

	return "", errors.New("不是有效的搜狐视频链接")
}

// extractVidFromPath 从解码后的路径中提取视频 ID
func (s sohuVideo) extractVidFromPath(path string) (string, error) {
	re := regexp.MustCompile(`/?us/\d+/(\d+)\.shtml`)
	matches := re.FindStringSubmatch(path)
	if len(matches) >= 2 && len(matches[1]) > 0 {
		return matches[1], nil
	}
	return "", fmt.Errorf("无法从路径 %s 中提取视频ID", path)
}

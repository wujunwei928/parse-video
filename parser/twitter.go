package parser

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

type twitter struct{}

func (t twitter) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	client := resty.New()

	// 处理 t.co 短链: 需要先跟随重定向获取真实URL
	if strings.Contains(shareUrl, "t.co/") {
		client.SetRedirectPolicy(resty.NoRedirectPolicy())
		res, err := client.R().
			SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
			Get(shareUrl)
		if !errors.Is(err, resty.ErrAutoRedirectDisabled) {
			return nil, fmt.Errorf("请求 t.co 短链失败: %v", err)
		}
		locationRes, err := res.RawResponse.Location()
		if err != nil {
			return nil, fmt.Errorf("获取 t.co 重定向地址失败: %v", err)
		}
		shareUrl = locationRes.String()
	}

	// 从 URL 中提取 tweet ID
	tweetId, err := t.extractTweetId(shareUrl)
	if err != nil {
		return nil, err
	}

	return t.parseVideoID(tweetId)
}

func (t twitter) parseVideoID(videoId string) (*VideoParseInfo, error) {
	token := t.getToken(videoId)
	apiUrl := fmt.Sprintf("https://cdn.syndication.twimg.com/tweet-result?id=%s&token=%s", videoId, token)

	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36").
		SetHeader("Accept", "application/json").
		SetHeader("Referer", "https://platform.twitter.com/").
		Get(apiUrl)
	if err != nil {
		return nil, fmt.Errorf("请求 Twitter syndication API 失败: %v", err)
	}

	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("Twitter API 返回错误状态码: %d", res.StatusCode())
	}

	jsonBytes := res.Body()

	// 提取作者信息
	authorName := gjson.GetBytes(jsonBytes, "user.name").String()
	authorScreenName := gjson.GetBytes(jsonBytes, "user.screen_name").String()
	authorAvatar := gjson.GetBytes(jsonBytes, "user.profile_image_url_https").String()
	authorId := gjson.GetBytes(jsonBytes, "user.id_str").String()

	// 提取推文文本作为标题
	title := gjson.GetBytes(jsonBytes, "text").String()

	// 提取视频信息: 从 mediaDetails 中查找 video 类型
	var videoUrl string
	var coverUrl string

	mediaDetails := gjson.GetBytes(jsonBytes, "mediaDetails")
	if mediaDetails.Exists() {
		for _, media := range mediaDetails.Array() {
			mediaType := media.Get("type").String()

			if mediaType == "video" || mediaType == "animated_gif" {
				// 获取封面图
				coverUrl = media.Get("media_url_https").String()

				// 从 variants 中选取最高码率的 mp4
				var maxBitrate int64
				for _, variant := range media.Get("video_info.variants").Array() {
					contentType := variant.Get("content_type").String()
					if contentType != "video/mp4" {
						continue
					}
					bitrate := variant.Get("bitrate").Int()
					url := variant.Get("url").String()
					if bitrate > maxBitrate || videoUrl == "" {
						maxBitrate = bitrate
						videoUrl = url
					}
				}
				break // 只取第一个视频
			}
		}
	}

	// 如果没有视频, 尝试提取图集
	images := make([]ImgInfo, 0)
	if len(videoUrl) == 0 {
		// 尝试从顶层 video 字段获取
		topVideoVariants := gjson.GetBytes(jsonBytes, "video.variants")
		if topVideoVariants.Exists() {
			coverUrl = gjson.GetBytes(jsonBytes, "video.poster").String()
			var maxBitrate int64
			for _, variant := range topVideoVariants.Array() {
				contentType := variant.Get("content_type").String()
				if contentType != "video/mp4" {
					continue
				}
				bitrate := variant.Get("bitrate").Int()
				url := variant.Get("url").String()
				if bitrate > maxBitrate || videoUrl == "" {
					maxBitrate = bitrate
					videoUrl = url
				}
			}
		}

		// 如果还是没有视频, 提取图片
		if len(videoUrl) == 0 && mediaDetails.Exists() {
			for _, media := range mediaDetails.Array() {
				if media.Get("type").String() == "photo" {
					imageUrl := media.Get("media_url_https").String()
					if len(imageUrl) > 0 {
						images = append(images, ImgInfo{
							Url: imageUrl,
						})
					}
				}
			}
			// 如果有图片, 用第一张作为封面
			if len(images) > 0 {
				coverUrl = images[0].Url
			}
		}
	}

	if len(videoUrl) == 0 && len(images) == 0 {
		return nil, errors.New("该推文中没有找到视频或图片")
	}

	// 使用 screen_name 作为作者名称的补充
	displayName := authorName
	if len(displayName) == 0 {
		displayName = authorScreenName
	}

	parseInfo := &VideoParseInfo{
		Title:    title,
		VideoUrl: videoUrl,
		CoverUrl: coverUrl,
		Images:   images,
	}
	parseInfo.Author.Uid = authorId
	parseInfo.Author.Name = displayName
	parseInfo.Author.Avatar = authorAvatar

	return parseInfo, nil
}

// extractTweetId 从推文 URL 中提取 tweet ID
// 支持格式:
// https://x.com/user/status/1234567890
// https://twitter.com/user/status/1234567890
// https://mobile.twitter.com/user/status/1234567890
func (t twitter) extractTweetId(shareUrl string) (string, error) {
	re := regexp.MustCompile(`(?:twitter\.com|x\.com)/[^/]+/status(?:es)?/(\d+)`)
	matches := re.FindStringSubmatch(shareUrl)
	if len(matches) < 2 {
		return "", fmt.Errorf("无法从 URL 中提取推文 ID: %s", shareUrl)
	}
	return matches[1], nil
}

// getToken 计算 syndication API 所需的 token
// 算法: (tweetId / 1e15 * π) 转换为字符串, 去除 "0" 和 "."
func (t twitter) getToken(id string) string {
	num, _ := strconv.ParseFloat(id, 64)
	token := (num / 1e15) * math.Pi
	tokenStr := strings.ReplaceAll(strconv.FormatFloat(token, 'f', -1, 64), "0", "")
	return strings.ReplaceAll(tokenStr, ".", "")
}

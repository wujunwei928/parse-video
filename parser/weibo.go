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

type weiBo struct {
}

func (w weiBo) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	urlInfo, err := url.Parse(shareUrl)
	if err != nil {
		return nil, errors.New("parse share url fail")
	}

	// Handle video URLs
	if strings.Contains(shareUrl, "show?fid=") {
		if len(urlInfo.Query()["fid"]) <= 0 {
			return nil, errors.New("can not parse video id from share url")
		}
		videoId := urlInfo.Query()["fid"][0]
		return w.parseVideoID(videoId)
	} else if strings.Contains(shareUrl, "/tv/show/") {
		videoId := strings.ReplaceAll(urlInfo.Path, "/tv/show/", "")
		return w.parseVideoID(videoId)
	} else {
		// Handle regular post URLs (potential image albums)
		// Extract post ID from URLs like https://weibo.com/2543858012/Q9pcJ4S21
		pathParts := strings.Split(strings.Trim(urlInfo.Path, "/"), "/")
		if len(pathParts) >= 2 {
			postId := pathParts[len(pathParts)-1]
			return w.parsePostUrl(postId, shareUrl)
		}
	}

	return nil, errors.New("unsupported weibo url format")
}

func (w weiBo) parseVideoID(videoId string) (*VideoParseInfo, error) {
	reqUrl := fmt.Sprintf("https://h5.video.weibo.com/api/component?page=/show/%s", videoId)
	client := resty.New()
	videoRes, err := client.R().
		SetHeader(HttpHeaderCookie, "login_sid_t=6b652c77c1a4bc50cb9d06b24923210d; cross_origin_proto=SSL; WBStorage=2ceabba76d81138d|undefined; _s_tentry=passport.weibo.com; Apache=7330066378690.048.1625663522444; SINAGLOBAL=7330066378690.048.1625663522444; ULV=1625663522450:1:1:1:7330066378690.048.1625663522444:; TC-V-WEIBO-G0=35846f552801987f8c1e8f7cec0e2230; SUB=_2AkMXuScYf8NxqwJRmf8RzmnhaoxwzwDEieKh5dbDJRMxHRl-yT9jqhALtRB6PDkJ9w8OaqJAbsgjdEWtIcilcZxHG7rw; SUBP=0033WrSXqPxfM72-Ws9jqgMF55529P9D9W5Qx3Mf.RCfFAKC3smW0px0; XSRF-TOKEN=JQSK02Ijtm4Fri-YIRu0-vNj").
		SetHeader(HttpHeaderReferer, "https://h5.video.weibo.com/show/"+videoId).
		SetHeader(HttpHeaderContentType, "application/x-www-form-urlencoded").
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		SetBody([]byte(`data={"Component_Play_Playinfo":{"oid":"` + videoId + `"}}`)).
		Post(reqUrl)
	if err != nil {
		return nil, err
	}
	data := gjson.GetBytes(videoRes.Body(), "data.Component_Play_Playinfo")
	var videoUrl string
	data.Get("urls").ForEach(func(key, value gjson.Result) bool {
		if len(videoUrl) == 0 {
			// 第一条码率最高
			videoUrl = "https:" + value.String()
		}
		return true
	})
	parseInfo := &VideoParseInfo{
		Title:    data.Get("title").String(),
		VideoUrl: videoUrl,
		CoverUrl: "https:" + data.Get("cover_image").String(),
	}
	parseInfo.Author.Name = data.Get("author").String()
	parseInfo.Author.Avatar = "https:" + data.Get("avatar").String()

	return parseInfo, nil
}

func (w weiBo) parsePostUrl(postId string, originalUrl string) (*VideoParseInfo, error) {
	// Try mobile API first
	reqUrl := fmt.Sprintf("https://m.weibo.cn/statuses/show?id=%s", postId)
	client := resty.New()

	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		SetHeader(HttpHeaderReferer, "https://m.weibo.cn/").
		SetHeader(HttpHeaderContentType, "application/json;charset=UTF-8").
		SetHeader("X-Requested-With", "XMLHttpRequest").
		Get(reqUrl)
	if err == nil {
		data := gjson.GetBytes(res.Body(), "data")
		if data.Exists() {
			return w.parseMobileApiData(data)
		}
	}

	// Fallback to desktop page parsing using the original URL
	res, err = client.R().
		SetHeader(HttpHeaderUserAgent, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36").
		Get(originalUrl)
	if err != nil {
		return nil, err
	}

	return w.parseHtmlPage(res.Body())
}

func (w weiBo) parseMobileApiData(data gjson.Result) (*VideoParseInfo, error) {
	// Extract basic info
	title := data.Get("text").String()
	authorName := data.Get("user.screen_name").String()
	authorAvatar := data.Get("user.avatar_large").String()

	// Get images
	images := make([]ImgInfo, 0)
	picsData := data.Get("pics")
	if picsData.Exists() {
		picsArray := picsData.Array()
		for _, pic := range picsArray {
			// Get the largest image URL available
			largePicUrl := pic.Get("large.url").String()
			if largePicUrl == "" {
				largePicUrl = pic.Get("original.url").String()
			}
			if largePicUrl == "" {
				largePicUrl = pic.Get("bmiddle.url").String()
			}
			if largePicUrl == "" {
				largePicUrl = pic.Get("url").String()
			}

			// 微博示例：https://weibo.com/6871895822/5211295285513194
			// 获取 live photo URL
			livePhotoUrl := pic.Get("videoSrc").String()

			if largePicUrl != "" {
				images = append(images, ImgInfo{
					Url:          largePicUrl,
					LivePhotoUrl: livePhotoUrl,
				})
			}
		}
	}

	parseInfo := &VideoParseInfo{
		Title:    w.cleanText(title),
		VideoUrl: "", // Regular posts don't have videos
		CoverUrl: "",
		Images:   images,
	}
	parseInfo.Author.Name = authorName
	parseInfo.Author.Avatar = authorAvatar

	return parseInfo, nil
}

func (w weiBo) parseHtmlPage(htmlBody []byte) (*VideoParseInfo, error) {
	// Try to extract data from $render_data script
	re := regexp.MustCompile(`\$render_data\s*=\s*(.*?)\[0\]`)
	findRes := re.FindSubmatch(htmlBody)
	if len(findRes) < 2 {
		return nil, errors.New("parse weibo html page fail")
	}

	jsonStr := string(findRes[1]) + "[0]"
	data := gjson.Parse(jsonStr)

	// Extract basic info
	title := data.Get("status.text").String()
	authorName := data.Get("status.user.screen_name").String()
	authorAvatar := data.Get("status.user.avatar_large").String()

	// Get images
	images := make([]ImgInfo, 0)
	picsData := data.Get("status.pics")
	if picsData.Exists() {
		picsArray := picsData.Array()
		for _, pic := range picsArray {
			// Get the largest image URL available
			largePicUrl := pic.Get("large.url").String()
			if largePicUrl == "" {
				largePicUrl = pic.Get("original.url").String()
			}
			if largePicUrl == "" {
				largePicUrl = pic.Get("bmiddle.url").String()
			}
			if largePicUrl == "" {
				largePicUrl = pic.Get("url").String()
			}

			if largePicUrl != "" {
				images = append(images, ImgInfo{
					Url: largePicUrl,
				})
			}
		}
	}

	parseInfo := &VideoParseInfo{
		Title:    w.cleanText(title),
		VideoUrl: "", // Regular posts don't have videos
		CoverUrl: "",
		Images:   images,
	}
	parseInfo.Author.Name = authorName
	parseInfo.Author.Avatar = authorAvatar

	return parseInfo, nil
}

// cleanText removes HTML tags from text
func (w weiBo) cleanText(text string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	cleaned := re.ReplaceAllString(text, "")
	return strings.TrimSpace(cleaned)
}

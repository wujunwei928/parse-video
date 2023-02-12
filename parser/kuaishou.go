package parser

import (
	"errors"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/go-resty/resty/v2"
)

type kuaiShou struct{}

func (k kuaiShou) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	client := resty.New()
	client.SetRedirectPolicy(resty.NoRedirectPolicy())
	res, _ := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		Get(shareUrl)
	//这里会返回err, auto redirect is disabled

	locationRes, err := res.RawResponse.Location()
	if err != nil {
		return nil, err
	}

	// 分享的中间跳转链接不太一样, 有些是 /fw/long-video , 有些 /fw/photo
	referUri := strings.ReplaceAll(locationRes.String(), "v.m.chenzhongtech.com/fw/long-video", "m.gifshow.com/fw/photo")
	referUri = strings.ReplaceAll(referUri, "v.m.chenzhongtech.com/fw/photo", "m.gifshow.com/fw/photo")

	videoId := strings.ReplaceAll(strings.Trim(locationRes.Path, "/"), "fw/long-video/", "")
	videoId = strings.ReplaceAll(videoId, "fw/photo/", "")
	if len(videoId) <= 0 {
		return nil, errors.New("parse video id from share url fail")
	}

	postData := map[string]interface{}{
		"photoId":     videoId,
		"isLongVideo": false,
	}
	videoRes, err := client.R().
		SetHeader(HttpHeaderCookie, "_did=web_4611110883127BC1; did=web_9a0b966fb1674f6c9a4886a504bee5e5").
		SetHeader("Origin", "https://m.gifshow.com").
		SetHeader(HttpHeaderReferer, strings.ReplaceAll(referUri, "m.gifshow.com/fw/photo", "m.gifshow.com/fw/photo")).
		SetHeader(HttpHeaderContentType, "application/json").
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		SetBody(postData).
		Post("https://m.gifshow.com/rest/wd/photo/info?kpn=KUAISHOU&captchaToken=")

	data := gjson.GetBytes(videoRes.Body(), "photo")
	avatar := data.Get("headUrl").String()
	author := data.Get("userName").String()
	title := data.Get("caption").String()
	videoUrl := data.Get("mainMvUrls.0.url").String()
	cover := data.Get("coverUrls.0.url").String()

	parseRes := &VideoParseInfo{
		Title:    title,
		VideoUrl: videoUrl,
		CoverUrl: cover,
	}
	parseRes.Author.Name = author
	parseRes.Author.Avatar = avatar

	return parseRes, nil
}

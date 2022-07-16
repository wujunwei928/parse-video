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
	referUri := strings.ReplaceAll(locationRes.String(), "v.m.chenzhongtech.com/fw/long-video", "video.kuaishou.com/video")

	videoId := strings.ReplaceAll(strings.Trim(locationRes.Path, "/"), "fw/long-video/", "")
	if len(videoId) <= 0 {
		return nil, errors.New("parse video id from share url fail")
	}

	postData := map[string]interface{}{
		"photoId":     videoId,
		"isLongVideo": false,
	}
	videoRes, err := client.R().
		SetHeader(HttpHeaderCookie, "did=web_9bceee20fa5d4a968535a27e538bf51b; didv=1655992503000;").
		SetHeader(HttpHeaderReferer, referUri).
		SetHeader(HttpHeaderContentType, "application/json").
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		SetBody(postData).
		Post("https://v.m.chenzhongtech.com/rest/wd/photo/info")

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

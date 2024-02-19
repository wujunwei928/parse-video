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
		SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7").
		Get(shareUrl)
	//这里会返回err, auto redirect is disabled

	// 获取 cookies： did，didv
	cookies := res.RawResponse.Cookies()
	if len(cookies) <= 0 {
		return nil, errors.New("get cookies from share url fail")
	}

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
		"fid":               "0",
		"shareResourceType": "PHOTO_OTHER",
		"shareChannel":      "share_copylink",
		"kpn":               "KUAISHOU",
		"subBiz":            "BROWSE_SLIDE_PHOTO",
		"env":               "SHARE_VIEWER_ENV_TX_TRICK",
		"h5Domain":          "m.gifshow.com",
		"photoId":           videoId,
		"isLongVideo":       false,
	}
	videoRes, err := client.R().
		SetHeader("Origin", "https://m.gifshow.com").
		SetHeader(HttpHeaderReferer, strings.ReplaceAll(referUri, "m.gifshow.com/fw/photo", "m.gifshow.com/fw/photo")).
		SetHeader(HttpHeaderContentType, "application/json").
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		SetCookies(cookies).
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

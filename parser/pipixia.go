package parser

import (
	"errors"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

type piPiXia struct {
}

func (p piPiXia) parseVideoID(videoId string) (*VideoParseInfo, error) {
	reqUrl := "https://is.snssdk.com/bds/cell/detail/?cell_type=1&aid=1319&app_name=super&cell_id=" + videoId
	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		Get(reqUrl)
	if err != nil {
		return nil, err
	}

	data := gjson.GetBytes(res.Body(), "data.data.item")
	author := data.Get("author.name").String()
	avatar := data.Get("author.avatar.download_list.0.url").String()
	title := data.Get("share.title").String()
	videoUrl := data.Get("origin_video_download.url_list.0.url").String()
	cover := data.Get("cover.url_list.0.url").String()

	parseRes := &VideoParseInfo{
		Title:    title,
		VideoUrl: videoUrl,
		CoverUrl: cover,
	}
	parseRes.Author.Name = author
	parseRes.Author.Avatar = avatar

	return parseRes, nil
}

func (p piPiXia) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	client := resty.New()
	// disable redirects in the HTTP client, get params before redirects
	client.SetRedirectPolicy(resty.NoRedirectPolicy())
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		Get(shareUrl)
	// 非 resty.ErrAutoRedirectDisabled 错误时，返回错误
	if !errors.Is(err, resty.ErrAutoRedirectDisabled) {
		return nil, err
	}

	locationRes, err := res.RawResponse.Location()
	if err != nil {
		return nil, err
	}

	videoId := strings.ReplaceAll(strings.Trim(locationRes.Path, "/"), "item/", "")
	if len(videoId) <= 0 {
		return nil, errors.New("parse video id from share url fail")
	}

	return p.parseVideoID(videoId)
}

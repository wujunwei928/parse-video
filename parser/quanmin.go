package parser

import (
	"errors"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

type quanMin struct{}

func (q quanMin) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	urlRes, err := url.Parse(shareUrl)
	if err != nil {
		return nil, err
	}

	videoId := urlRes.Query().Get("vid")
	if len(videoId) <= 0 {
		return nil, errors.New("parse video_id from share url fail")
	}

	return q.parseVideoID(videoId)
}

func (q quanMin) parseVideoID(videoId string) (*VideoParseInfo, error) {
	reqUrl := "https://quanmin.hao222.com/wise/growth/api/sv/immerse?source=share-h5&pd=qm_share_mvideo&_format=json&vid=" + videoId
	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		Get(reqUrl)
	if err != nil {
		return nil, err
	}

	data := gjson.GetBytes(res.Body(), "data")
	author := data.Get("author.name").String()
	avatar := data.Get("author.icon").String()
	title := data.Get("meta.title").String()
	videoUrl := data.Get("meta.video_info.clarityUrl.1.url").String()
	cover := data.Get("meta.image").String()

	parseRes := &VideoParseInfo{
		Title:    title,
		VideoUrl: videoUrl,
		CoverUrl: cover,
	}
	parseRes.Author.Name = author
	parseRes.Author.Avatar = avatar

	return parseRes, nil
}

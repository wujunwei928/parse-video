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

	// 接口返回错误
	if gjson.GetBytes(res.Body(), "errno").Int() != 0 {
		return nil, errors.New(gjson.GetBytes(res.Body(), "error").String())
	}
	// 视频状态错误
	metaStatusText := gjson.GetBytes(res.Body(), "data.meta.statusText").String()
	if len(metaStatusText) > 0 {
		return nil, errors.New(metaStatusText)
	}

	data := gjson.GetBytes(res.Body(), "data")
	author := data.Get("author.name").String()
	avatar := data.Get("author.icon").String()
	videoUrl := data.Get("meta.video_info.clarityUrl.1.url").String()
	cover := data.Get("meta.image").String()
	// 获取视频标题，如果没有则使用分享标题
	title := data.Get("meta.title").String()
	if len(title) <= 0 {
		title = data.Get("shareInfo.title").String()
	}

	parseRes := &VideoParseInfo{
		Title:    title,
		VideoUrl: videoUrl,
		CoverUrl: cover,
	}
	parseRes.Author.Uid = data.Get("author.id").String()
	parseRes.Author.Name = author
	parseRes.Author.Avatar = avatar

	return parseRes, nil
}

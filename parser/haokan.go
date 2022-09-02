package parser

import (
	"errors"
	"net/url"

	"github.com/tidwall/gjson"

	"github.com/go-resty/resty/v2"
)

type haoKan struct {
}

func (h haoKan) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	urlInfo, err := url.Parse(shareUrl)
	if err != nil {
		return nil, errors.New("parse share url fail")
	}
	if len(urlInfo.Query()["vid"]) <= 0 {
		return nil, errors.New("can not parse video id from share url")
	}
	return h.parseVideoID(urlInfo.Query()["vid"][0])
}

func (h haoKan) parseVideoID(videoId string) (*VideoParseInfo, error) {
	reqUrl := "https://haokan.baidu.com/v?_format=json&vid=" + videoId
	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		Get(reqUrl)
	if err != nil {
		return nil, err
	}

	data := gjson.GetBytes(res.Body(), "data.apiData.curVideoMeta")
	title := data.Get("title").String()
	videoUrl := data.Get("playurl").String()
	cover := data.Get("poster").String()

	parseRes := &VideoParseInfo{
		Title:    title,
		VideoUrl: videoUrl,
		CoverUrl: cover,
	}

	return parseRes, nil
}

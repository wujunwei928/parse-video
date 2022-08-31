package parser

import (
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/go-resty/resty/v2"
)

type quanMinKGe struct {
}

func (q quanMinKGe) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	urlInfo, err := url.Parse(shareUrl)
	if err != nil {
		return nil, errors.New("parse share url fail")
	}
	if len(urlInfo.Query()["s"]) <= 0 {
		return nil, errors.New("can not parse video id from share url")
	}
	return q.parseVideoID(urlInfo.Query()["s"][0])
}

func (q quanMinKGe) parseVideoID(videoId string) (*VideoParseInfo, error) {
	reqUrl := "https://kg.qq.com/node/play?s=" + videoId
	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.5112.102 Safari/537.36 Edg/104.0.1293.70").
		Get(reqUrl)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`window.__DATA__ = (.*?);`)
	findRes := re.FindSubmatch(res.Body())
	if len(findRes) < 2 {
		return nil, errors.New("parse video json info from html fail")
	}

	data := gjson.GetBytes([]byte(strings.TrimSpace(string(findRes[1]))), "detail")
	parseInfo := &VideoParseInfo{
		Title:    data.Get("content").String(),
		VideoUrl: data.Get("playurl_video").String(),
		CoverUrl: data.Get("cover").String(),
	}
	parseInfo.Author.Name = data.Get("nick").String()
	parseInfo.Author.Avatar = data.Get("avatar").String()

	return parseInfo, nil
}

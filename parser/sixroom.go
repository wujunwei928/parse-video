package parser

import (
	"errors"
	"net/url"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

type sixRoom struct {
}

func (s sixRoom) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	urlInfo, err := url.Parse(shareUrl)
	if err != nil {
		return nil, errors.New("parse share url fail")
	}
	var videoId string
	if strings.Contains(shareUrl, "watchMini.php?vid=") {
		if len(urlInfo.Query()["vid"]) <= 0 {
			return nil, errors.New("can not parse video id from share url")
		}
		videoId = urlInfo.Query()["vid"][0]
	} else {
		videoId = strings.ReplaceAll(urlInfo.Path, "/v/", "")
	}
	return s.parseVideoID(videoId)
}

func (s sixRoom) parseVideoID(videoId string) (*VideoParseInfo, error) {
	reqUrl := "https://v.6.cn/coop/mobile/index.php?padapi=minivideo-watchVideo.php&av=3.0&encpass=&logiuid=&isnew=1&from=0&vid=" + videoId
	client := resty.New()
	videoRes, err := client.R().
		SetHeader(HttpHeaderReferer, "https://m.6.cn/v/"+videoId).
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		Get(reqUrl)
	if err != nil {
		return nil, err
	}

	data := gjson.GetBytes(videoRes.Body(), "content")
	parseInfo := &VideoParseInfo{
		Title:    data.Get("title").String(),
		VideoUrl: data.Get("playurl").String(),
		CoverUrl: data.Get("picurl").String(),
	}
	parseInfo.Author.Name = data.Get("alias").String()
	parseInfo.Author.Avatar = data.Get("picuser").String()

	return parseInfo, nil
}

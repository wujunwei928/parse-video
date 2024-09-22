package parser

import (
	"errors"
	"net/url"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

type zuiYou struct{}

func (z zuiYou) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	urlInfo, err := url.Parse(shareUrl)
	if err != nil {
		return nil, errors.New("parse share url fail")
	}
	if len(urlInfo.Query()["pid"]) <= 0 {
		return nil, errors.New("can not parse video id from share url")
	}
	pid := urlInfo.Query()["pid"][0]
	intPid, err := strconv.Atoi(pid)
	if err != nil {
		return nil, err
	}
	postData := map[string]interface{}{
		"h_av": "5.2.13.011",
		"pid":  intPid,
	}

	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		SetBody(postData).
		Post("https://share.xiaochuankeji.cn/planck/share/post/detail_h5")
	if err != nil {
		return nil, err
	}

	data := gjson.GetBytes(res.Body(), "data.post")
	videoKey := data.Get("imgs.0.id").String()
	videoPlayAddr := data.Get("videos." + videoKey + ".url").String()
	videoCover := data.Get("videos." + videoKey + ".cover_urls.0").String()
	title := data.Get("content").String()
	userName := data.Get("member.name").String()
	userAvatar := data.Get("member.avatar_urls.origin.urls.0").String()

	parseRes := &VideoParseInfo{
		Title:    title,
		VideoUrl: videoPlayAddr,
		CoverUrl: videoCover,
	}
	parseRes.Author.Name = userName
	parseRes.Author.Avatar = userAvatar

	return parseRes, nil
}

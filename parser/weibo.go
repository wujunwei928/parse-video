package parser

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

type weiBo struct {
}

func (w weiBo) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	urlInfo, err := url.Parse(shareUrl)
	if err != nil {
		return nil, errors.New("parse share url fail")
	}
	var videoId string
	if strings.Contains(shareUrl, "show?fid=") {
		if len(urlInfo.Query()["fid"]) <= 0 {
			return nil, errors.New("can not parse video id from share url")
		}
		videoId = urlInfo.Query()["fid"][0]
	} else {
		videoId = strings.ReplaceAll(urlInfo.Path, "/tv/show/", "")
	}
	return w.parseVideoID(videoId)
}

func (w weiBo) parseVideoID(videoId string) (*VideoParseInfo, error) {
	reqUrl := fmt.Sprintf("https://h5.video.weibo.com/api/component?page=/show/%s", videoId)
	client := resty.New()
	videoRes, err := client.R().
		SetHeader(HttpHeaderCookie, "login_sid_t=6b652c77c1a4bc50cb9d06b24923210d; cross_origin_proto=SSL; WBStorage=2ceabba76d81138d|undefined; _s_tentry=passport.weibo.com; Apache=7330066378690.048.1625663522444; SINAGLOBAL=7330066378690.048.1625663522444; ULV=1625663522450:1:1:1:7330066378690.048.1625663522444:; TC-V-WEIBO-G0=35846f552801987f8c1e8f7cec0e2230; SUB=_2AkMXuScYf8NxqwJRmf8RzmnhaoxwzwDEieKh5dbDJRMxHRl-yT9jqhALtRB6PDkJ9w8OaqJAbsgjdEWtIcilcZxHG7rw; SUBP=0033WrSXqPxfM72-Ws9jqgMF55529P9D9W5Qx3Mf.RCfFAKC3smW0px0; XSRF-TOKEN=JQSK02Ijtm4Fri-YIRu0-vNj").
		SetHeader(HttpHeaderReferer, "https://h5.video.weibo.com/show/"+videoId).
		SetHeader(HttpHeaderContentType, "application/x-www-form-urlencoded").
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		SetBody([]byte(`data={"Component_Play_Playinfo":{"oid":"` + videoId + `"}}`)).
		Post(reqUrl)
	if err != nil {
		return nil, err
	}
	data := gjson.GetBytes(videoRes.Body(), "data.Component_Play_Playinfo")
	var videoUrl string
	data.Get("urls").ForEach(func(key, value gjson.Result) bool {
		if len(videoUrl) == 0 {
			// 第一条码率最高
			videoUrl = "https:" + value.String()
		}
		return true
	})
	parseInfo := &VideoParseInfo{
		Title:    data.Get("title").String(),
		VideoUrl: videoUrl,
		CoverUrl: "https:" + data.Get("cover_image").String(),
	}
	parseInfo.Author.Name = data.Get("author").String()
	parseInfo.Author.Avatar = "https:" + data.Get("avatar").String()

	return parseInfo, nil
}

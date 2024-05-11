package parser

import (
	"errors"
	"net/url"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

type piPiGaoXiao struct {
}

func (p piPiGaoXiao) parseVideoID(videoId string) (*VideoParseInfo, error) {
	reqUrl := "https://share.ippzone.com/ppapi/share/fetch_content"
	headers := map[string]string{
		HttpHeaderReferer:   reqUrl,
		HttpHeaderUserAgent: "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36",
	}
	postData := "{\"pid\":" + videoId + ",\"type\":\"post\",\"mid\":null}"

	client := resty.New()
	res, err := client.R().
		SetHeaders(headers).
		SetBody([]byte(postData)).
		Post(reqUrl)
	if err != nil {
		return nil, err
	}

	// 接口返回错误
	apiErr := gjson.GetBytes(res.Body(), "msg")
	if apiErr.Exists() {
		return nil, errors.New(apiErr.String())
	}

	data := gjson.GetBytes(res.Body(), "data.post")
	title := data.Get("content").String()
	id := data.Get("imgs.0.id").String()
	videoUrl := data.Get("videos." + id + ".url").String()
	cover := "https://file.ippzone.com/img/view/id/" + id

	parseRes := &VideoParseInfo{
		Title:    title,
		VideoUrl: videoUrl,
		CoverUrl: cover,
	}

	return parseRes, nil
}

func (p piPiGaoXiao) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	urlRes, err := url.Parse(shareUrl)
	if err != nil {
		return nil, err
	}

	videoId := strings.ReplaceAll(urlRes.Path, "/pp/post/", "")
	if len(videoId) <= 0 {
		return nil, errors.New("parse video_id from share url fail")
	}

	return p.parseVideoID(videoId)
}

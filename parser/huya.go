package parser

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

type huYa struct {
}

func (h huYa) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	re := regexp.MustCompile(`\/(\d+).html`)

	findRes := re.FindSubmatch([]byte(shareUrl))
	if len(findRes) < 2 {
		return nil, errors.New("parse video from share url fail")
	}

	return h.parseVideoID(string(findRes[1]))
}

func (h huYa) parseVideoID(videoId string) (*VideoParseInfo, error) {
	reqUrl := "https://liveapi.huya.com/moment/getMomentContent?videoId=" + videoId
	headers := map[string]string{
		HttpHeaderUserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/85.0.4183.102 Safari/537.36",
		HttpHeaderReferer:   "https://v.huya.com/",
	}

	client := resty.New()
	res, err := client.R().
		SetHeaders(headers).
		Get(reqUrl)
	if err != nil {
		return nil, err
	}
	videoData := gjson.GetBytes(res.Body(), "data.moment.videoInfo")
	fmt.Println(string(res.Body()))
	parseRes := &VideoParseInfo{
		Title:    videoData.Get("videoTitle").String(),
		VideoUrl: videoData.Get("definitions.0.url").String(),
		CoverUrl: videoData.Get("videoCover").String(),
	}
	parseRes.Author.Avatar = videoData.Get("actorAvatarUrl").String()
	parseRes.Author.Name = videoData.Get("actorNick").String()

	return parseRes, nil
}

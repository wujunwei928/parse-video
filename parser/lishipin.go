package parser

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/go-resty/resty/v2"
)

type liShiPin struct {
}

func (l liShiPin) parseVideoID(videoId string) (*VideoParseInfo, error) {
	reqUrl := fmt.Sprintf("https://www.pearvideo.com/videoStatus.jsp?contId=%s&mrd=%d", videoId, time.Now().Unix())
	headers := map[string]string{
		HttpHeaderReferer:   fmt.Sprintf("https://www.pearvideo.com/detail_%s", videoId),
		HttpHeaderUserAgent: "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36",
	}

	client := resty.New()
	res, err := client.R().
		SetHeaders(headers).
		Get(reqUrl)
	if err != nil {
		return nil, err
	}

	data := gjson.GetBytes(res.Body(), "videoInfo")
	videoSrcUrl := data.Get("videos.srcUrl").String()
	timer := gjson.GetBytes(res.Body(), "systemTime").String()
	videoUrl := strings.ReplaceAll(videoSrcUrl, timer, "cont-"+videoId)
	cover := data.Get("video_image").String()

	parseRes := &VideoParseInfo{
		VideoUrl: videoUrl,
		CoverUrl: cover,
	}

	return parseRes, nil
}

func (l liShiPin) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	urlRes, err := url.Parse(shareUrl)
	if err != nil {
		return nil, err
	}

	videoId := strings.ReplaceAll(urlRes.Path, "/detail_", "")
	if len(videoId) <= 0 {
		return nil, errors.New("parse video_id from share url fail")
	}

	return l.parseVideoID(videoId)
}

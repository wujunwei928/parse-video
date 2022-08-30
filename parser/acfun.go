package parser

import (
	"regexp"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/go-resty/resty/v2"
)

type acFun struct {
}

func (a acFun) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, "User-Agent:Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1").
		Get(shareUrl)
	if err != nil {
		return nil, err
	}

	parseInfo := &VideoParseInfo{}

	videoInfoRe := regexp.MustCompile(`var videoInfo =\s(.*?);`)
	if findRes := videoInfoRe.FindSubmatch(res.Body()); len(findRes) >= 2 {
		jsonStr := strings.TrimSpace(string(findRes[1]))
		parseInfo.Title = gjson.Get(jsonStr, "title").String()
		parseInfo.CoverUrl = gjson.Get(jsonStr, "cover").String()
	}
	playInfoRe := regexp.MustCompile(`var playInfo =\s(.*?);`)
	if findRes := playInfoRe.FindSubmatch(res.Body()); len(findRes) >= 2 {
		jsonStr := strings.TrimSpace(string(findRes[1]))
		parseInfo.VideoUrl = gjson.Get(jsonStr, "streams.0.playUrls.0").String()
		// 视频地址是m3u8, 可以使用网站 https://tools.thatwind.com/tool/m3u8downloader 下载
	}

	return parseInfo, nil
}

func (a acFun) parseVideoID(videoId string) (*VideoParseInfo, error) {
	// acid, 格式: ac36935385
	reqUrl := "https://www.acfun.cn/v/" + videoId
	return a.parseShareUrl(reqUrl)
}

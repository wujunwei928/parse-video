package parser

import (
	"bytes"

	"github.com/PuerkitoBio/goquery"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

type xinPianChang struct {
}

func (x xinPianChang) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.125 Safari/537.3").
		//SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		SetHeader("Upgrade-Insecure-Requests", "1").
		SetHeader(HttpHeaderReferer, "https://www.xinpianchang.com/").
		Get(shareUrl)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(res.Body()))
	if err != nil {
		return nil, err
	}
	videoJson := doc.Find("#__NEXT_DATA__").Text()
	//fmt.Println(videoJson)

	data := gjson.Get(videoJson, "props.pageProps.detail")
	avatar := data.Get("author.userinfo.avatar").String()
	author := data.Get("author.userinfo.username").String()
	title := data.Get("title").String()
	videoUrl := data.Get("video.content.progressive.0.url").String()
	cover := data.Get("cover").String()

	parseRes := &VideoParseInfo{
		Title:    title,
		VideoUrl: videoUrl,
		CoverUrl: cover,
	}
	parseRes.Author.Name = author
	parseRes.Author.Avatar = avatar

	return parseRes, nil
}

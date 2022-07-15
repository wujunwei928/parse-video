package parser

import (
	"bytes"
	"errors"

	"github.com/PuerkitoBio/goquery"

	"github.com/go-resty/resty/v2"
)

type zuiYou struct{}

func (z zuiYou) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	if len(shareUrl) <= 0 {
		return nil, errors.New("video share url is empty")
	}

	client := resty.New()
	res, err := client.R().
		EnableTrace().
		Get(shareUrl)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(res.Body()))
	if err != nil {
		return nil, err
	}

	videoPlayAddr, _ := doc.Find("video").Attr("src")
	title := doc.Find(".SharePostCard__content h1").Text()

	parseRes := &VideoParseInfo{
		Desc:          title,
		VideoPlayAddr: videoPlayAddr,
	}

	return parseRes, nil
}

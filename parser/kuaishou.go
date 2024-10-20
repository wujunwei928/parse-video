package parser

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

type kuaiShou struct{}

func (k kuaiShou) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7").
		Get(shareUrl)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`window.INIT_STATE\s*=\s*(.*?)</script>`)
	findRes := re.FindSubmatch(res.Body())
	if len(findRes) < 2 {
		return nil, errors.New("parse video json info from html fail")
	}
	jsonBytes := bytes.TrimSpace(findRes[1])

	var (
		videoRes   gjson.Result
		isFindInfo bool
	)

	for _, jsonItem := range gjson.ParseBytes(jsonBytes).Map() {
		jsonItemMap := jsonItem.Map()
		_, hasResult := jsonItemMap["result"]
		_, hasPhoto := jsonItemMap["photo"]
		if hasResult && hasPhoto {
			videoRes = jsonItem
			isFindInfo = true
			break
		}
	}

	if !isFindInfo {
		return nil, errors.New("parse video json fail")
	}

	if resultCode := videoRes.Get("result").Int(); resultCode != 1 {
		return nil, fmt.Errorf("获取作品信息失败:result=%d", resultCode)
	}

	data := videoRes.Get("photo")
	avatar := data.Get("headUrl").String()
	author := data.Get("userName").String()
	title := data.Get("caption").String()
	videoUrl := data.Get("mainMvUrls.0.url").String()
	cover := data.Get("coverUrls.0.url").String()

	// 获取图集
	imageCdnHost := data.Get("ext_params.atlas.cdn.0").String()
	imagesObjArr := data.Get("ext_params.atlas.list").Array()
	images := make([]string, 0, len(imagesObjArr))
	if len(imageCdnHost) > 0 && len(imagesObjArr) > 0 {
		for _, imageItem := range imagesObjArr {
			imageUrl := fmt.Sprintf("https://%s/%s", imageCdnHost, imageItem.String())
			images = append(images, imageUrl)
		}
	}

	parseRes := &VideoParseInfo{
		Title:    title,
		VideoUrl: videoUrl,
		CoverUrl: cover,
		Images:   images,
	}
	parseRes.Author.Name = author
	parseRes.Author.Avatar = avatar

	return parseRes, nil
}

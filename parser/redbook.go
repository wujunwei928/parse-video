package parser

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"

	"github.com/tidwall/gjson"

	"github.com/go-resty/resty/v2"
)

type redBook struct{}

func (r redBook) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	client := resty.New()
	videoRes, err := client.R().
		SetHeader(HttpHeaderUserAgent, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36 Edg/129.0.0.0").
		Get(shareUrl)
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`window.__INITIAL_STATE__\s*=\s*(.*?)</script>`)
	findRes := re.FindSubmatch(videoRes.Body())
	if len(findRes) < 2 {
		return nil, errors.New("parse video json info from html fail")
	}

	jsonBytes := bytes.TrimSpace(findRes[1])

	nodeId := gjson.GetBytes(jsonBytes, "note.currentNoteId").String()
	data := gjson.GetBytes(jsonBytes, fmt.Sprintf("note.noteDetailMap.%s.note", nodeId))

	videoUrl := data.Get("video.media.stream.h264.0.masterUrl").String()

	// 获取图集图片地址
	imagesObjArr := data.Get("imageList").Array()
	images := make([]string, 0, len(imagesObjArr))
	if len(videoUrl) <= 0 {
		for _, imageItem := range imagesObjArr {
			imageUrl := imageItem.Get("urlDefault").String()
			if len(imageUrl) > 0 {
				images = append(images, imageUrl)
			}
		}
	}

	parseInfo := &VideoParseInfo{
		Title:    data.Get("title").String(),
		VideoUrl: data.Get("video.media.stream.h264.0.masterUrl").String(),
		CoverUrl: data.Get("imageList.0.urlDefault").String(),
		Images:   images,
	}
	parseInfo.Author.Uid = data.Get("user.userId").String()
	parseInfo.Author.Name = data.Get("user.nickname").String()
	parseInfo.Author.Avatar = data.Get("user.avatar").String()

	return parseInfo, nil
}

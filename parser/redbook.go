package parser

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
)

type redBook struct{}

func (r redBook) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	client := newClient()
	// 桌面 UA 的无登录请求会被服务端 302 到登录页，必须使用移动端 UA
	videoRes, err := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		Get(shareUrl)
	if err != nil {
		return nil, err
	}

	return parseNoteInfoFromHtml(videoRes.Body())
}

// parseNoteInfoFromHtml 从笔记页 HTML 的 __INITIAL_STATE__ 中提取笔记数据。
// 兼容两种页面结构：移动端 noteData.data.noteData（当前返回），
// 桌面端 note.noteDetailMap.<noteId>.note（旧结构，保留兼容）。
func parseNoteInfoFromHtml(body []byte) (*VideoParseInfo, error) {
	// (?s) 使 . 匹配换行，兼容带换行缩进的页面
	re := regexp.MustCompile(`(?s)window\.__INITIAL_STATE__\s*=\s*(.*?)</script>`)
	findRes := re.FindSubmatch(body)
	if len(findRes) < 2 {
		return nil, errors.New("parse video json info from html fail")
	}

	jsonBytes := bytes.TrimSpace(findRes[1])

	data := gjson.GetBytes(jsonBytes, "noteData.data.noteData")
	if !data.IsObject() {
		nodeId := gjson.GetBytes(jsonBytes, "note.currentNoteId").String()
		data = gjson.GetBytes(jsonBytes, fmt.Sprintf("note.noteDetailMap.%s.note", nodeId))
	}
	if !data.IsObject() {
		return nil, errors.New("parse note info from html fail")
	}

	// 小红书为 h264 分享流烧录了水印（作者昵称+小红书 logo），h265 转码流无水印，
	// 故优先取 h265；老页面可能只有 h264 流，回退兼容
	videoUrl := data.Get("video.media.stream.h265.0.masterUrl").String()
	if len(videoUrl) <= 0 {
		videoUrl = data.Get("video.media.stream.h264.0.masterUrl").String()
	}

	// 获取图集图片地址
	imagesObjArr := data.Get("imageList").Array()
	images := make([]ImgInfo, 0, len(imagesObjArr))
	if len(videoUrl) <= 0 {
		for _, imageItem := range imagesObjArr {
			// 桌面端结构用 urlDefault，移动端结构用 url
			imageUrl := imageItem.Get("urlDefault").String()
			if len(imageUrl) <= 0 {
				imageUrl = imageItem.Get("url").String()
			}
			if len(imageUrl) <= 0 {
				continue
			}
			imgId := strings.Split(imageUrl[strings.LastIndex(imageUrl, "/")+1:], "!")[0]
			// 如果链接中带有 spectrum/ , 替换域名时需要带上
			spectrumStr := ""
			if strings.Contains(imageUrl, "spectrum") {
				spectrumStr = "spectrum/"
			}
			newUrl := fmt.Sprintf("https://ci.xiaohongshu.com/notes_pre_post/%s%s?imageView2/format/jpg", spectrumStr, imgId)
			imgInfo := ImgInfo{
				Url: newUrl,
			}
			// 如果原图片网址中没有 notes_pre_post 关键字，不支持替换域名，使用原域名
			if !strings.Contains(imageUrl, "notes_pre_post") {
				imgInfo.Url = imageUrl
			}
			if imageItem.Get("livePhoto").Bool() {
				for _, livePhotoItem := range imageItem.Get("stream.h264").Array() {
					if livePhotoUrl := livePhotoItem.Get("masterUrl").String(); len(livePhotoUrl) > 0 {
						imgInfo.LivePhotoUrl = livePhotoUrl
					}
				}
			}
			images = append(images, imgInfo)
		}
	}

	// 无视频且无图片说明页面是空壳（如被风控重定向到登录页），按解析失败处理
	if len(videoUrl) <= 0 && len(images) <= 0 {
		return nil, errors.New("parse note info from html fail")
	}

	parseInfo := &VideoParseInfo{
		Title:    data.Get("title").String(),
		VideoUrl: videoUrl,
		CoverUrl: data.Get("imageList.0.urlDefault").String(),
		Images:   images,
	}
	if parseInfo.CoverUrl == "" {
		parseInfo.CoverUrl = data.Get("imageList.0.url").String()
	}
	parseInfo.Author.Uid = data.Get("user.userId").String()
	// 桌面端字段为 nickname，移动端为 nickName
	parseInfo.Author.Name = data.Get("user.nickname").String()
	if parseInfo.Author.Name == "" {
		parseInfo.Author.Name = data.Get("user.nickName").String()
	}
	parseInfo.Author.Avatar = data.Get("user.avatar").String()

	return parseInfo, nil
}

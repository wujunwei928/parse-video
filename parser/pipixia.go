package parser

import (
	"errors"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

type piPiXia struct {
}

func (p piPiXia) parseVideoID(videoId string) (*VideoParseInfo, error) {
	reqUrl := "https://api.pipix.com/bds/cell/cell_comment/?offset=0&cell_type=1&api_version=1&cell_id=" + videoId + "&ac=wifi&channel=huawei_1319_64&aid=1319&app_name=super"
	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		Get(reqUrl)
	if err != nil {
		return nil, err
	}
	// data['data']['cell_comments'][0]['comment_info']['item']
	data := gjson.GetBytes(res.Body(), "data.cell_comments.0.comment_info.item")
	authorId := data.Get("author.id").String()

	// 获取图集图片地址
	imagesObjArr := data.Get("note.multi_image").Array()
	images := make([]ImgInfo, 0, len(imagesObjArr))
	for _, imageItem := range imagesObjArr {
		imageUrl := imageItem.Get("url_list.0.url").String()
		if len(imageUrl) > 0 {
			images = append(images, ImgInfo{
				Url: imageUrl,
			})
		}
	}

	videoUrl := data.Get("video.video_high.url_list.0.url").String() // 备用视频地址, 可能有水印
	// comments中可能带有不带水印视频, comments可能为空, 尝试获取
	for _, comment := range data.Get("comments").Array() {
		commentVideoUrl := comment.Get("item.video.video_high.url_list.0.url").String()
		if comment.Get("item.author.id").String() == authorId && len(commentVideoUrl) > 0 {
			videoUrl = commentVideoUrl
			break
		}
	}

	author := data.Get("author.name").String()
	avatar := data.Get("author.avatar.download_list.0.url").String()
	// 该接口标题为content
	title := data.Get("content").String()
	cover := data.Get("cover.url_list.0.url").String()

	parseRes := &VideoParseInfo{
		Title:    title,
		VideoUrl: videoUrl,
		CoverUrl: cover,
	}
	parseRes.Author.Name = author
	parseRes.Author.Avatar = avatar
	parseRes.Images = images

	return parseRes, nil
}

func (p piPiXia) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	client := resty.New()
	// disable redirects in the HTTP client, get params before redirects
	client.SetRedirectPolicy(resty.NoRedirectPolicy())
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		Get(shareUrl)
	// 非 resty.ErrAutoRedirectDisabled 错误时，返回错误
	if !errors.Is(err, resty.ErrAutoRedirectDisabled) {
		return nil, err
	}

	locationRes, err := res.RawResponse.Location()
	if err != nil {
		return nil, err
	}

	videoId := strings.ReplaceAll(strings.Trim(locationRes.Path, "/"), "item/", "")
	if len(videoId) <= 0 {
		return nil, errors.New("parse video id from share url fail")
	}

	return p.parseVideoID(videoId)
}

package parser

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/go-resty/resty/v2"
)

type douYin struct{}

func (d douYin) parseVideoID(videoId string) (*VideoParseInfo, error) {

	reqUrl, err := d.getDetailUrlByVideoId(videoId)
	if err != nil {
		return nil, err
	}

	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36").
		SetHeader(HttpHeaderReferer, "https://www.douyin.com/").
		SetHeader(HttpHeaderCookie, fmt.Sprintf(`msToken=%s;odin_tt=324fb4ea4a89c0c05827e18a1ed9cf9bf8a17f7705fcc793fec935b637867e2a5a9b8168c885554d029919117a18ba69; ttwid=1%7CWBuxH_bhbuTENNtACXoesI5QHV2Dt9-vkMGVHSRRbgY%7C1677118712%7C1d87ba1ea2cdf05d80204aea2e1036451dae638e7765b8a4d59d87fa05dd39ff; bd_ticket_guard_client_data=eyJiZC10aWNrZXQtZ3VhcmQtdmVyc2lvbiI6MiwiYmQtdGlja2V0LWd1YXJkLWNsaWVudC1jc3IiOiItLS0tLUJFR0lOIENFUlRJRklDQVRFIFJFUVVFU1QtLS0tLVxyXG5NSUlCRFRDQnRRSUJBREFuTVFzd0NRWURWUVFHRXdKRFRqRVlNQllHQTFVRUF3d1BZbVJmZEdsamEyVjBYMmQxXHJcbllYSmtNRmt3RXdZSEtvWkl6ajBDQVFZSUtvWkl6ajBEQVFjRFFnQUVKUDZzbjNLRlFBNUROSEcyK2F4bXAwNG5cclxud1hBSTZDU1IyZW1sVUE5QTZ4aGQzbVlPUlI4NVRLZ2tXd1FJSmp3Nyszdnc0Z2NNRG5iOTRoS3MvSjFJc3FBc1xyXG5NQ29HQ1NxR1NJYjNEUUVKRGpFZE1Cc3dHUVlEVlIwUkJCSXdFSUlPZDNkM0xtUnZkWGxwYmk1amIyMHdDZ1lJXHJcbktvWkl6ajBFQXdJRFJ3QXdSQUlnVmJkWTI0c0RYS0c0S2h3WlBmOHpxVDRBU0ROamNUb2FFRi9MQnd2QS8xSUNcclxuSURiVmZCUk1PQVB5cWJkcytld1QwSDZqdDg1czZZTVNVZEo5Z2dmOWlmeTBcclxuLS0tLS1FTkQgQ0VSVElGSUNBVEUgUkVRVUVTVC0tLS0tXHJcbiJ9`, d.randSeq(107))).
		Get(reqUrl)
	if err != nil {
		return nil, err
	}

	data := gjson.GetBytes(res.Body(), "aweme_detail")

	videoInfo := &VideoParseInfo{
		Title:    data.Get("desc").String(),
		VideoUrl: data.Get("video.play_addr.url_list.0").String(),
		MusicUrl: data.Get("music.play_url.url_list.0").String(),
		CoverUrl: data.Get("video.origin_cover.url_list.0").String(),
	}
	videoInfo.Author.Uid = data.Get("author.unique_id").String()
	videoInfo.Author.Name = data.Get("author.nickname").String()
	videoInfo.Author.Avatar = data.Get("music.avatar_large.url_list.0").String()

	return videoInfo, nil
}

func (d douYin) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	client := resty.New()
	client.SetRedirectPolicy(resty.NoRedirectPolicy())
	res, _ := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		Get(shareUrl)
	// 这里会返回err, auto redirect is disabled

	locationRes, err := res.RawResponse.Location()
	if err != nil {
		return nil, err
	}

	videoId := strings.ReplaceAll(strings.Trim(locationRes.Path, "/"), "share/video/", "")
	if len(videoId) <= 0 {
		return nil, errors.New("parse video id from share url fail")
	}

	return d.parseVideoID(videoId)
}

func (d douYin) getDetailUrlByVideoId(videoId string) (string, error) {
	postData := map[string]interface{}{
		"url":        fmt.Sprintf("https://www.douyin.com/aweme/v1/web/aweme/detail/?aweme_id=%s&aid=1128&version_name=23.5.0&device_platform=android&os_version=2333", videoId),
		"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36",
	}

	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderContentType, "application/json").
		SetBody(postData).
		Post("https://tiktok.iculture.cc/X-Bogus")
	if err != nil {
		return "", err
	}

	return gjson.GetBytes(res.Body(), "param").String(), nil
}

func (d douYin) randSeq(n int) string {
	letters := []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

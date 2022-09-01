package parser

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
)

type meiPai struct {
}

func (m meiPai) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.88 Safari/537.36").
		Get(shareUrl)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(res.Body()))
	if err != nil {
		return nil, err
	}
	videoBs64, ok := doc.Find("#shareMediaBtn").Attr("data-video")
	if !ok {
		return nil, errors.New("parse video base64 from share url fail")
	}
	videoUrl, err := m.parseVideoBs64(videoBs64)
	if err != nil {
		return nil, errors.New("parse video play url fail")
	}
	coverUrl, _ := doc.Find("#detailVideo img").Attr("src")
	userName, _ := doc.Find(".detail-avatar").Attr("alt")
	userAvatar, _ := doc.Find(".detail-avatar").Attr("src")

	parseInfo := &VideoParseInfo{
		Title:    doc.Find(".detail-cover-title").Text(),
		VideoUrl: videoUrl,
		CoverUrl: coverUrl,
	}
	parseInfo.Author.Name = userName
	parseInfo.Author.Avatar = "https:" + userAvatar

	return parseInfo, nil

}

func (m meiPai) parseVideoID(videoId string) (*VideoParseInfo, error) {
	reqUrl := "https://www.meipai.com/video/" + videoId
	return m.parseShareUrl(reqUrl)
}

func (m meiPai) parseVideoBs64(videoBs64 string) (string, error) {
	hex := m.getHex(videoBs64)
	dec, err := m.getDec(hex["hex_1"])
	if err != nil {
		return "", err
	}
	d, err := m.subStr(hex["str_1"], dec["pre"])
	if err != nil {
		return "", err
	}
	p, err := m.getPos(d, dec["tail"])
	if err != nil {
		return "", err
	}
	kk, err := m.subStr(d, p)
	if err != nil {
		return "", err
	}
	decodeBs64, err := base64.StdEncoding.DecodeString(kk)
	if err != nil {
		return "", err
	}
	videoUrl := "https:" + string(decodeBs64)
	return videoUrl, nil
}

func (m meiPai) getHex(s string) map[string]string {
	length := len(s)
	hex := s[0:4]
	str := s[4:length]
	return map[string]string{
		"hex_1": m.reverseString(hex),
		"str_1": str,
	}
}

func (m meiPai) getDec(hex string) (map[string][]int, error) {
	n, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return nil, err
	}
	intN := int(n)
	length := len(strconv.Itoa(intN))
	pre := make([]int, 0, 2)
	tail := make([]int, 0, length-2)
	// 从后往前截取int
	// 9812
	// 2 981 : 9812%10=2 ; (9812-2)/10 = 981
	// 1 98  : 981%10=1  ; (981-1)/10 = 98
	// 8 9   : 98%10=8   ; (98-8)/10=9
	// 9 0   : 9%10=9
	for i, tmpN := 0, intN; i <= length-1; i++ {
		tmp := tmpN % 10
		tmpN = (tmpN - tmp) / 10
		if i >= length-2 {
			pre = append([]int{tmp}, pre...)
		} else {
			tail = append([]int{tmp}, tail...)
		}
	}
	return map[string][]int{
		"pre":  pre,
		"tail": tail,
	}, nil
}

func (m meiPai) subStr(s string, b []int) (string, error) {
	if len(b) < 2 {
		return "", errors.New("substr param b length is not correct")
	}
	length := len(s)
	c := s[0:b[0]]
	d := s[b[0] : b[0]+b[1]]
	temp := strings.ReplaceAll(s[b[0]:length], d, "")
	return c + temp, nil
}

func (m meiPai) getPos(s string, b []int) ([]int, error) {
	if len(b) < 2 {
		return nil, errors.New("getpos param b length is not correct")
	}
	b[0] = len(s) - b[0] - b[1]
	return b, nil
}

func (m meiPai) reverseString(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

package parser

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"
)

type kuaiShou struct{}

type LogRedirects struct {
	Transport http.RoundTripper
}

func (l LogRedirects) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	t := l.Transport
	if t == nil {
		t = http.DefaultTransport
	}
	resp, err = t.RoundTrip(req)
	if err != nil {
		return
	}
	switch resp.StatusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect:
		log.Println("Request for", req.URL, "redirected with status", resp.StatusCode)
	}
	return
}

func (k kuaiShou) parseVideoID(videoId string) (*VideoParseInfo, error) {
	return nil, nil
}

func (k kuaiShou) parseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	if len(shareUrl) <= 0 {
		return nil, errors.New("video share url is empty")
	}

	client := resty.New()
	client.SetRedirectPolicy(resty.NoRedirectPolicy())
	res, _ := client.R().Get(shareUrl)
	//这里会返回err, auto redirect is disabled

	locationRes, err := res.RawResponse.Location()
	if err != nil {
		return nil, err
	}

	videoId := strings.ReplaceAll(strings.Trim(locationRes.Path, "/"), "fw/long-video/", "")
	if len(videoId) <= 0 {
		return nil, errors.New("parse video id from share url fail")
	}

	//res, err := client.R().
	//	//SetHeader("Cookie", "did=web_9bceee20fa5d4a968535a27e538bf51b; didv=1655992503000;").
	//	//SetHeader("Referer", "https://video.kuaishou.com/video/3x8hpv6t9zjm9bu").
	//	SetHeader("Content-Type", "application/json").
	//	SetBody(postBytes).
	//	Post("https://v.m.chenzhongtech.com/rest/wd/photo/info")

	return k.parseVideoID(videoId)
}

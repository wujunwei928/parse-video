package parser

import (
	"errors"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/go-resty/resty/v2"
)

type douYin struct{}

func (d douYin) parseVideoID(videoId string) (*VideoParseInfo, error) {
	reqUrl := "https://www.iesdouyin.com/aweme/v1/web/aweme/detail/?aweme_id=" + videoId
	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderCookie, "ttwid=1%7Cg3mkQ4zWpDIHPZhitKABkAlgY7wYIjXaL-dKPKq9Gik%7C1676131262%7C4393576ad4aae2ab8091a2fb6ad679c882851c313c3f1f18418dacd779d34fd5; __ac_nonce=063e8a51e00246bd7927b; __ac_signature=_02B4Z6wo00f0130dUGgAAIDCiGK69KnZR.d9PVTAALy72e; msToken=Pjd8tqkk_5DcDHCu1SPENat04F4ReGou6-ooN4L9Zxw-rnq0JzMj-OnNZ90k2e79Ccu1Zr_FagheEFg8GULPVXBFHNVtFO_1bZbExqyo0Ic=; s_v_web_id=verify_le153mts_CjtZAjYN_VDN0_42E1_BONz_W9Q42yHSntOm; msToken=0e9rBOId2Tk1RhlLE0W0v5w2Tmw0WX_Fsea1MfUBW2E6G8kerGIOVO8VzTreSlf1SPgsVqRNZ7PEHPEmZzRykwQElZKH90zxwGUh05dmB20al3UFlfBV; ttcid=0886122963064dd2a054ba4dcedf61c537; tt_scid=1sT1CdTVVMLIJePCKOxMPLvSFDeEmNLPvDHT14XwEoNDPwjZO2GA97Oqq5Eipkk2aa1c").
		SetHeader(HttpHeaderUserAgent, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.0.0 Safari/537.36 Edg/110.0.1587.41").
		SetHeader("authority", "www.iesdouyin.com").
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

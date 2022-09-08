package parser

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"github.com/go-resty/resty/v2"
)

type douYinRes struct {
	StatusCode int `json:"status_code"`
	ItemList   []struct {
		Author struct {
			UniqueId     string `json:"unique_id"`
			Nickname     string `json:"nickname"`
			AvatarLarger struct {
				UrlList []string `json:"url_list"`
				Uri     string   `json:"uri"`
			} `json:"avatar_larger"`
		} `json:"author"`
		Video struct {
			PlayAddr struct {
				UrlList []string `json:"url_list"`
				Uri     string   `json:"uri"`
			} `json:"play_addr"`
			OriginCover struct {
				Uri     string   `json:"uri"`
				UrlList []string `json:"url_list"`
			} `json:"origin_cover"`
		} `json:"video"`
		Music struct {
			Title   string `json:"title"`
			PlayUrl struct {
				Uri     string   `json:"uri"`
				UrlList []string `json:"url_list"`
			} `json:"play_url"`
		} `json:"music"`
		Desc      string `json:"desc"`
		AwemeId   string `json:"aweme_id"`
		ShareInfo struct {
			ShareWeiboDesc string `json:"share_weibo_desc"`
			ShareDesc      string `json:"share_desc"`
			ShareTitle     string `json:"share_title"`
		} `json:"share_info"`
	} `json:"item_list"`
}

type douYin struct{}

func (d douYin) parseVideoID(videoId string) (*VideoParseInfo, error) {
	parseMap, err := d.batchParseVideoID([]string{videoId})
	if err != nil {
		return nil, err
	}
	if _, ok := parseMap[videoId]; !ok {
		return nil, errors.New("has no parse info")
	}

	return parseMap[videoId].ParseInfo, nil
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

func (d douYin) batchParseVideoID(videoIds []string) (map[string]BatchParseItem, error) {
	// 支持多个videoId批量获取, 用逗号隔开
	itemIds := strings.Join(videoIds, ",")
	reqUrl := "https://www.iesdouyin.com/web/api/v2/aweme/iteminfo/?item_ids=" + itemIds
	client := resty.New()
	res, err := client.R().
		SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
		Get(reqUrl)
	if err != nil {
		return nil, err
	}

	douYinRes := &douYinRes{}
	json.Unmarshal(res.Body(), douYinRes)

	parseList := make(map[string]BatchParseItem, len(douYinRes.ItemList))
	for _, item := range douYinRes.ItemList {
		if len(item.Video.PlayAddr.UrlList) <= 0 {
			continue
		}
		videoPlayAddr := strings.ReplaceAll(item.Video.PlayAddr.UrlList[0], "/playwm/", "/play/")
		parseItem := &VideoParseInfo{
			Title:    item.ShareInfo.ShareWeiboDesc,
			VideoUrl: videoPlayAddr,
			MusicUrl: item.Music.PlayUrl.Uri,
			CoverUrl: item.Video.OriginCover.UrlList[0],
		}
		parseItem.Author.Uid = item.Author.UniqueId
		parseItem.Author.Name = item.Author.Nickname
		parseItem.Author.Avatar = item.Author.AvatarLarger.UrlList[0]

		parseList[item.AwemeId] = BatchParseItem{
			ParseInfo: parseItem,
		}
	}

	d.getRedirectUrl(parseList)

	return parseList, nil
}

// 获取重定向后的链接
func (d douYin) getRedirectUrl(videoInfoMap map[string]BatchParseItem) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	for id, item := range videoInfoMap {
		wg.Add(1)
		videoId := id
		go func(strUrl string) {
			defer wg.Done()

			client := resty.New()
			client.SetRedirectPolicy(resty.NoRedirectPolicy())
			res2, _ := client.R().
				SetHeader(HttpHeaderUserAgent, DefaultUserAgent).
				Get(strUrl)
			locationRes, _ := res2.RawResponse.Location()
			if locationRes != nil {
				mu.Lock()
				videoInfoMap[videoId].ParseInfo.VideoUrl = locationRes.String()
				mu.Unlock()
			}
		}(item.ParseInfo.VideoUrl)
	}

	wg.Wait()
}

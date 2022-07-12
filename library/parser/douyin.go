package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
)

type DouYin struct {
	VideoId  string
	ShareUrl string
}

type DouYinRes struct {
	FilterList []interface{} `json:"filter_list"`
	Extra      struct {
		Now   int64  `json:"now"`
		Logid string `json:"logid"`
	} `json:"extra"`
	StatusCode int `json:"status_code"`
	ItemList   []struct {
		AwemeType  int         `json:"aweme_type"`
		Promotions interface{} `json:"promotions"`
		AnchorInfo struct {
			Type int    `json:"type"`
			Id   string `json:"id"`
			Name string `json:"name"`
		} `json:"anchor_info,omitempty"`
		ShareUrl     string `json:"share_url"`
		IsPreview    int    `json:"is_preview"`
		AuthorUserId int64  `json:"author_user_id"`
		Video        struct {
			HasWatermark bool `json:"has_watermark"`
			PlayAddr     struct {
				UrlList []string `json:"url_list"`
				Uri     string   `json:"uri"`
			} `json:"play_addr"`
			Cover struct {
				Uri     string   `json:"uri"`
				UrlList []string `json:"url_list"`
			} `json:"cover"`
			Height       int `json:"height"`
			Width        int `json:"width"`
			DynamicCover struct {
				UrlList []string `json:"url_list"`
				Uri     string   `json:"uri"`
			} `json:"dynamic_cover"`
			OriginCover struct {
				Uri     string   `json:"uri"`
				UrlList []string `json:"url_list"`
			} `json:"origin_cover"`
			Ratio    string      `json:"ratio"`
			BitRate  interface{} `json:"bit_rate"`
			Vid      string      `json:"vid"`
			Duration int         `json:"duration"`
		} `json:"video"`
		ShareInfo struct {
			ShareWeiboDesc string `json:"share_weibo_desc"`
			ShareDesc      string `json:"share_desc"`
			ShareTitle     string `json:"share_title"`
		} `json:"share_info"`
		CommentList interface{} `json:"comment_list"`
		GroupIdStr  string      `json:"group_id_str"`
		Author      struct {
			Nickname        string      `json:"nickname"`
			FollowStatus    int         `json:"follow_status"`
			FollowersDetail interface{} `json:"followers_detail"`
			AvatarLarger    struct {
				Uri     string   `json:"uri"`
				UrlList []string `json:"url_list"`
			} `json:"avatar_larger"`
			PolicyVersion interface{} `json:"policy_version"`
			CardEntries   interface{} `json:"card_entries"`
			Uid           string      `json:"uid"`
			AvatarMedium  struct {
				Uri     string   `json:"uri"`
				UrlList []string `json:"url_list"`
			} `json:"avatar_medium"`
			PlatformSyncInfo interface{} `json:"platform_sync_info"`
			Geofencing       interface{} `json:"geofencing"`
			MixInfo          interface{} `json:"mix_info"`
			ShortId          string      `json:"short_id"`
			Signature        string      `json:"signature"`
			AvatarThumb      struct {
				Uri     string   `json:"uri"`
				UrlList []string `json:"url_list"`
			} `json:"avatar_thumb"`
			UniqueId  string      `json:"unique_id"`
			TypeLabel interface{} `json:"type_label"`
		} `json:"author"`
		VideoText  interface{} `json:"video_text"`
		GroupId    int64       `json:"group_id"`
		CreateTime int         `json:"create_time"`
		Statistics struct {
			PlayCount    int    `json:"play_count"`
			ShareCount   int    `json:"share_count"`
			AwemeId      string `json:"aweme_id"`
			CommentCount int    `json:"comment_count"`
			DiggCount    int    `json:"digg_count"`
		} `json:"statistics"`
		Duration     int         `json:"duration"`
		LabelTopText interface{} `json:"label_top_text"`
		ForwardId    string      `json:"forward_id"`
		AwemePoiInfo struct {
			TypeName string `json:"type_name"`
			Tag      string `json:"tag"`
			Icon     struct {
				Uri     string   `json:"uri"`
				UrlList []string `json:"url_list"`
			} `json:"icon"`
			PoiName string `json:"poi_name"`
		} `json:"aweme_poi_info,omitempty"`
		AwemeId string `json:"aweme_id"`
		Music   struct {
			Title   string `json:"title"`
			CoverHd struct {
				Uri     string   `json:"uri"`
				UrlList []string `json:"url_list"`
			} `json:"cover_hd"`
			PlayUrl struct {
				Uri     string   `json:"uri"`
				UrlList []string `json:"url_list"`
			} `json:"play_url"`
			Duration   int    `json:"duration"`
			Status     int    `json:"status"`
			Mid        string `json:"mid"`
			Author     string `json:"author"`
			CoverLarge struct {
				Uri     string   `json:"uri"`
				UrlList []string `json:"url_list"`
			} `json:"cover_large"`
			CoverMedium struct {
				Uri     string   `json:"uri"`
				UrlList []string `json:"url_list"`
			} `json:"cover_medium"`
			CoverThumb struct {
				Uri     string   `json:"uri"`
				UrlList []string `json:"url_list"`
			} `json:"cover_thumb"`
			Position interface{} `json:"position"`
			Id       int64       `json:"id"`
		} `json:"music"`
		IsLiveReplay bool        `json:"is_live_replay"`
		Images       interface{} `json:"images"`
		Desc         string      `json:"desc"`
		TextExtra    []struct {
			Start       int    `json:"start"`
			End         int    `json:"end"`
			Type        int    `json:"type"`
			HashtagName string `json:"hashtag_name"`
			HashtagId   int64  `json:"hashtag_id"`
		} `json:"text_extra"`
		ImageInfos interface{} `json:"image_infos"`
		Geofencing interface{} `json:"geofencing"`
		LongVideo  interface{} `json:"long_video"`
		ChaList    []struct {
			ConnectMusic interface{} `json:"connect_music"`
			CoverItem    struct {
				UrlList []string `json:"url_list"`
				Uri     string   `json:"uri"`
			} `json:"cover_item"`
			HashTagProfile string `json:"hash_tag_profile"`
			ChaName        string `json:"cha_name"`
			Desc           string `json:"desc"`
			UserCount      int    `json:"user_count"`
			IsCommerce     bool   `json:"is_commerce"`
			Cid            string `json:"cid"`
			Type           int    `json:"type"`
			ViewCount      int    `json:"view_count"`
		} `json:"cha_list"`
		RiskInfos struct {
			Content          string `json:"content"`
			ReflowUnplayable int    `json:"reflow_unplayable"`
			Warn             bool   `json:"warn"`
			Type             int    `json:"type"`
		} `json:"risk_infos"`
		MixInfo struct {
			NextInfo struct {
				MixName  string `json:"mix_name"`
				Desc     string `json:"desc"`
				CoverUrl struct {
					Uri     string   `json:"uri"`
					UrlList []string `json:"url_list"`
				} `json:"cover_url"`
			} `json:"next_info"`
			MixId    string `json:"mix_id"`
			MixName  string `json:"mix_name"`
			CoverUrl struct {
				Uri     string   `json:"uri"`
				UrlList []string `json:"url_list"`
			} `json:"cover_url"`
			Desc       string `json:"desc"`
			CreateTime int    `json:"create_time"`
			Status     struct {
				Status      int `json:"status"`
				IsCollected int `json:"is_collected"`
			} `json:"status"`
			Statis struct {
				CurrentEpisode   int `json:"current_episode"`
				UpdatedToEpisode int `json:"updated_to_episode"`
				PlayVv           int `json:"play_vv"`
				CollectVv        int `json:"collect_vv"`
			} `json:"statis"`
			Extra string `json:"extra"`
		} `json:"mix_info,omitempty"`
		VideoLabels interface{} `json:"video_labels"`
		Timer       struct {
			Status     int `json:"status"`
			PublicTime int `json:"public_time"`
		} `json:"timer,omitempty"`
	} `json:"item_list"`
}

func (d *DouYin) Parse() ([]*VideoParseInfo, error) {
	if len(d.VideoId) > 0 {
		return d.ParseByVideoID()
	} else if len(d.ShareUrl) > 0 {
		return d.ParseByShareUrl()
	}

	return nil, errors.New("video id and share url can not both empty")
}

func (d *DouYin) ParseByVideoID() ([]*VideoParseInfo, error) {
	if len(d.VideoId) <= 0 {
		return nil, errors.New("video id is empty")
	}

	// 支持多个videoId批量获取, 用逗号隔开
	reqUrl := "https://www.iesdouyin.com/web/api/v2/aweme/iteminfo/?item_ids=" + d.VideoId
	client := resty.New()
	res, err := client.R().Get(reqUrl)
	if err != nil {
		return nil, err
	}

	douYinRes := &DouYinRes{}
	json.Unmarshal(res.Body(), douYinRes)

	parseList := make([]*VideoParseInfo, 0, len(douYinRes.ItemList))
	for _, item := range douYinRes.ItemList {
		parseItem := &VideoParseInfo{
			Desc:          item.Desc,
			VideoPlayAddr: fmt.Sprintf("https://aweme.snssdk.com/aweme/v1/playwm/?video_id=%s&ratio=720p&line=0", item.Video.PlayAddr.Uri),
			MusicPlayAddr: item.Music.PlayUrl.Uri,
		}
		parseList = append(parseList, parseItem)
	}

	return parseList, nil
}

func (d *DouYin) ParseByShareUrl() ([]*VideoParseInfo, error) {
	if len(d.ShareUrl) <= 0 {
		return nil, errors.New("video share url is empty")
	}

	client := resty.New()
	client.SetRedirectPolicy(resty.NoRedirectPolicy())
	res, _ := client.R().EnableTrace().Get(d.ShareUrl)
	// 这里会返回err, auto redirect is disabled

	locationRes, err := res.RawResponse.Location()
	if err != nil {
		return nil, err
	}

	videoId := strings.ReplaceAll(strings.Trim(locationRes.Path, "/"), "share/video/", "")
	fmt.Println(locationRes.Path, videoId)
	if len(videoId) <= 0 {
		return nil, errors.New("parse video id from share url fail")
	}

	d.VideoId = videoId
	return d.ParseByVideoID()
}

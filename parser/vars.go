package parser

const (
	SourceDouYin   = "douyin"
	SourceKuaiShou = "kuaishou"
	SourcePiPiXia  = "pipixia"
	SourceHuoShan  = "huoshan"
	SourceWeiBo    = "weibo"
	SourceWeiShi   = "weishi"
	SourceLvZhou   = "lvzhou"
	SourceZuiYou   = "zuiyou"
	SourceBBQ      = "bbq"
	SourceQuanMin  = "quanmin"
	SourceXiGua    = "xigua"
	SourceLiShiPin = "lishipin"
)

// videoShareUrlParser 根据视频分享地址解析
type videoShareUrlParser interface {
	parseShareUrl(shareUrl string) (*VideoParseInfo, error)
}

// videoIdParser 根据视频ID解析
type videoIdParser interface {
	parseVideoID(videoId string) (*VideoParseInfo, error)
}

// VideoParseInfo 视频解析信息
type VideoParseInfo struct {
	Author struct {
		Uid    string `json:"uid"`    // 作者id
		Name   string `json:"title"`  // 作者名称
		Avatar string `json:"avatar"` // 作者头像
	} `json:"author"`
	Title    string `json:"title"`     // 描述
	VideoUrl string `json:"video_url"` // 视频播放地址
	MusicUrl string `json:"music_url"` // 音乐播放地址
	CoverUrl string `json:"cover_url"` // 视频封面地址
}

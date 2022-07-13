package parser

const (
	SourceDouYin   = "douyin"
	SourceKuaiShou = "kuaishou"
	SourcePiPiXia  = "pipixia"
	SourceHuoShan  = "huoshan"
	SourceWeiBo    = "weibo"
	SourceLvZhou   = "lvzhou"
	SourceZuiYou   = "zuiyou"
	SourceBBQ      = "bbq"
	SourceQuanMin  = "quanmin"
)

// videoIdParser 根据视频ID解析
type videoIdParser interface {
	parseVideoID(videoId string) ([]*VideoParseInfo, error)
}

// videoShareUrlParser 根据视频分享地址解析
type videoShareUrlParser interface {
	parseShareUrl(shareUrl string) ([]*VideoParseInfo, error)
}

// VideoParseInfo 视频解析信息
type VideoParseInfo struct {
	Desc          string `json:"desc"`            // 描述
	VideoPlayAddr string `json:"video_play_addr"` // 视频播放地址
	MusicPlayAddr string `json:"music_play_addr"` // 音乐播放地址
}

package parser

// VideoIdParser 根据视频ID解析
type VideoIdParser interface {
	ParseByVideoID() (*VideoParseInfo, error)
}

// VideoShareUrlParser 根据视频分享地址解析
type VideoShareUrlParser interface {
	ParseByShareUrl() (*VideoParseInfo, error)
}

// VideoParseInfo 视频解析信息
type VideoParseInfo struct {
	Desc          string `json:"desc"`            // 描述
	VideoPlayAddr string `json:"video_play_addr"` // 视频播放地址
	MusicPlayAddr string `json:"music_play_addr"` // 音乐播放地址
}

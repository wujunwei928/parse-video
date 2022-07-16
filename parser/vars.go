package parser

const (
	SourceDouYin      = "douyin"
	SourceKuaiShou    = "kuaishou"
	SourcePiPiXia     = "pipixia"
	SourceHuoShan     = "huoshan"
	SourceWeiBo       = "weibo"
	SourceWeiShi      = "weishi"
	SourceLvZhou      = "lvzhou"
	SourceZuiYou      = "zuiyou"
	SourceBBQ         = "bbq"
	SourceQuanMin     = "quanmin"
	SourceXiGua       = "xigua"
	SourceLiShiPin    = "lishipin"
	SourcePiPiGaoXiao = "pipigaoxiao"
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

// 视频渠道信息
type videoSourceInfo struct {
	VideoShareUrlDomain string              // 视频分享地址域名
	VideoShareUrlParser videoShareUrlParser // 视频分享地址解析方法
	VideoIdParser       videoIdParser       // 视频id解析方法, 有些渠道可能没有id解析方法
}

// 视频渠道映射信息
var videoSourceInfoMapping = map[string]videoSourceInfo{
	SourceDouYin: {
		VideoShareUrlDomain: "v.douyin.com",
		VideoShareUrlParser: douYin{},
		VideoIdParser:       douYin{},
	},
	SourceKuaiShou: {
		VideoShareUrlDomain: "v.kuaishou.com",
		VideoShareUrlParser: kuaiShou{},
	},
	SourceZuiYou: {
		VideoShareUrlDomain: "share.xiaochuankeji.cn",
		VideoShareUrlParser: zuiYou{},
	},
	SourceXiGua: {
		VideoShareUrlDomain: "v.ixigua.com",
		VideoShareUrlParser: xiGua{},
		VideoIdParser:       xiGua{},
	},
	SourcePiPiXia: {
		VideoShareUrlDomain: "h5.pipix.com",
		VideoShareUrlParser: piPiXia{},
		VideoIdParser:       piPiXia{},
	},
	SourceWeiShi: {
		VideoShareUrlDomain: "isee.weishi.qq.com",
		VideoShareUrlParser: weiShi{},
		VideoIdParser:       weiShi{},
	},
	SourceHuoShan: {
		VideoShareUrlDomain: "share.huoshan.com",
		VideoShareUrlParser: huoShan{},
		VideoIdParser:       huoShan{},
	},
	SourceLiShiPin: {
		VideoShareUrlDomain: "www.pearvideo.com",
		VideoShareUrlParser: liShiPin{},
		VideoIdParser:       liShiPin{},
	},
	SourcePiPiGaoXiao: {
		VideoShareUrlDomain: "h5.pipigx.com",
		VideoShareUrlParser: piPiGaoXiao{},
		VideoIdParser:       piPiGaoXiao{},
	},
}

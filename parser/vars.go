package parser

// 视频渠道来源
const (
	SourceDouYin       = "douyin"       // 抖音
	SourceKuaiShou     = "kuaishou"     // 快手
	SourcePiPiXia      = "pipixia"      // 皮皮虾
	SourceHuoShan      = "huoshan"      // 火山
	SourceWeiBo        = "weibo"        // 微博
	SourceWeiShi       = "weishi"       // 微视
	SourceLvZhou       = "lvzhou"       // 绿洲
	SourceZuiYou       = "zuiyou"       // 最右
	SourceQuanMin      = "quanmin"      // 度小视(原 全民小视频)
	SourceXiGua        = "xigua"        // 西瓜
	SourceLiShiPin     = "lishipin"     // 梨视频
	SourcePiPiGaoXiao  = "pipigaoxiao"  // 皮皮搞笑
	SourceHuYa         = "huya"         // 虎牙
	SourceAcFun        = "acfun"        // A站
	SourceDouPai       = "doupai"       // 逗拍
	SourceMeiPai       = "meipai"       // 美拍
	SourceQuanMinKGe   = "quanminkge"   // 全民K歌
	SourceSixRoom      = "sixroom"      // 六间房
	SourceXinPianChang = "xinpianchang" // 新片场
	SourceHaoKan       = "haokan"       // 好看视频
	SourceRedBook      = "redbook"      // 小红书
)

// http 相关
const (
	HttpHeaderUserAgent   = "User-Agent" //http header
	HttpHeaderReferer     = "Referer"
	HttpHeaderContentType = "Content-Type"
	HttpHeaderCookie      = "Cookie"

	// DefaultUserAgent 默认UserAgent
	DefaultUserAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1"
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
		Name   string `json:"name"`   // 作者名称
		Avatar string `json:"avatar"` // 作者头像
	} `json:"author"`
	Title    string   `json:"title"`     // 描述
	VideoUrl string   `json:"video_url"` // 视频播放地址
	MusicUrl string   `json:"music_url"` // 音乐播放地址
	CoverUrl string   `json:"cover_url"` // 视频封面地址
	Images   []string `json:"images"`    // 图集图片地址列表
}

// BatchParseItem 批量解析时, 单条解析格式
type BatchParseItem struct {
	ParseInfo *VideoParseInfo // 视频解析信息
	Error     error           // 错误, 如果单条解析失败时, 记录error信息
}

// 视频渠道信息
type videoSourceInfo struct {
	VideoShareUrlDomain []string            // 视频分享地址域名
	VideoShareUrlParser videoShareUrlParser // 视频分享地址解析方法
	VideoIdParser       videoIdParser       // 视频id解析方法, 有些渠道可能没有id解析方法
}

// 视频渠道映射信息
var videoSourceInfoMapping = map[string]videoSourceInfo{
	SourceDouYin: {
		VideoShareUrlDomain: []string{"v.douyin.com", "www.iesdouyin.com", "www.douyin.com"},
		VideoShareUrlParser: douYin{},
		VideoIdParser:       douYin{},
	},
	SourceKuaiShou: {
		VideoShareUrlDomain: []string{"v.kuaishou.com"},
		VideoShareUrlParser: kuaiShou{},
	},
	SourceZuiYou: {
		VideoShareUrlDomain: []string{"share.xiaochuankeji.cn"},
		VideoShareUrlParser: zuiYou{},
	},
	SourceXiGua: {
		VideoShareUrlDomain: []string{"v.ixigua.com"},
		VideoShareUrlParser: xiGua{},
		VideoIdParser:       xiGua{},
	},
	SourcePiPiXia: {
		VideoShareUrlDomain: []string{"h5.pipix.com"},
		VideoShareUrlParser: piPiXia{},
		VideoIdParser:       piPiXia{},
	},
	SourceWeiShi: {
		VideoShareUrlDomain: []string{"isee.weishi.qq.com"},
		VideoShareUrlParser: weiShi{},
		VideoIdParser:       weiShi{},
	},
	SourceHuoShan: {
		VideoShareUrlDomain: []string{"share.huoshan.com"},
		VideoShareUrlParser: huoShan{},
		VideoIdParser:       huoShan{},
	},
	SourceLiShiPin: {
		VideoShareUrlDomain: []string{"www.pearvideo.com"},
		VideoShareUrlParser: liShiPin{},
		VideoIdParser:       liShiPin{},
	},
	SourcePiPiGaoXiao: {
		VideoShareUrlDomain: []string{"h5.pipigx.com"},
		VideoShareUrlParser: piPiGaoXiao{},
		VideoIdParser:       piPiGaoXiao{},
	},
	SourceQuanMin: {
		VideoShareUrlDomain: []string{"xspshare.baidu.com"},
		VideoShareUrlParser: quanMin{},
		VideoIdParser:       quanMin{},
	},
	SourceHuYa: {
		VideoShareUrlDomain: []string{"v.huya.com"},
		VideoShareUrlParser: huYa{},
		VideoIdParser:       huYa{},
	},
	SourceAcFun: {
		VideoShareUrlDomain: []string{"www.acfun.cn"},
		VideoShareUrlParser: acFun{},
		VideoIdParser:       acFun{},
	},
	SourceWeiBo: {
		VideoShareUrlDomain: []string{"weibo.com"},
		VideoShareUrlParser: weiBo{},
		VideoIdParser:       weiBo{},
	},
	SourceLvZhou: {
		VideoShareUrlDomain: []string{"weibo.cn"},
		VideoShareUrlParser: lvZhou{},
		VideoIdParser:       lvZhou{},
	},
	SourceMeiPai: {
		VideoShareUrlDomain: []string{"meipai.com"},
		VideoShareUrlParser: meiPai{},
		VideoIdParser:       meiPai{},
	},
	SourceDouPai: {
		VideoShareUrlDomain: []string{"doupai.cc"},
		VideoShareUrlParser: douPai{},
		VideoIdParser:       douPai{},
	},
	SourceQuanMinKGe: {
		VideoShareUrlDomain: []string{"kg.qq.com"},
		VideoShareUrlParser: quanMinKGe{},
		VideoIdParser:       quanMinKGe{},
	},
	SourceSixRoom: {
		VideoShareUrlDomain: []string{"6.cn"},
		VideoShareUrlParser: sixRoom{},
		VideoIdParser:       sixRoom{},
	},
	SourceXinPianChang: {
		VideoShareUrlDomain: []string{"xinpianchang.com"},
		VideoShareUrlParser: xinPianChang{},
	},
	SourceHaoKan: {
		VideoShareUrlDomain: []string{
			"haokan.baidu.com",
			"haokan.hao123.com",
		},
		VideoShareUrlParser: haoKan{},
		VideoIdParser:       haoKan{},
	},
	SourceRedBook: {
		VideoShareUrlDomain: []string{
			"www.xiaohongshu.com",
			"xhslink.com",
		},
		VideoShareUrlParser: redBook{},
	},
}

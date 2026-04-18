package cmd

import (
	"net/url"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wujunwei928/parse-video/parser"
	"github.com/wujunwei928/parse-video/utils"
)

var (
	parseVideoShareURL = parser.ParseVideoShareUrlByRegexp
	parseVideoID       = parser.ParseVideoId
)

// platformNames 平台显示名称映射（按 source 字母序）
var platformNames = map[string]string{
	"acfun":        "AcFun",
	"bilibili":     "哔哩哔哩",
	"doupai":       "逗拍",
	"douyin":       "抖音",
	"haokan":       "好看视频",
	"huoshan":      "火山",
	"huya":         "虎牙",
	"kuaishou":     "快手",
	"lishipin":     "梨视频",
	"lvzhou":       "绿洲",
	"meipai":       "美拍",
	"pipigaoxiao":  "皮皮搞笑",
	"pipixia":      "皮皮虾",
	"quanmin":      "度小视",
	"quanminkge":   "全民K歌",
	"redbook":      "小红书",
	"sixroom":      "六间房",
	"twitter":      "X/Twitter",
	"weibo":        "微博",
	"weishi":       "微视",
	"xigua":        "西瓜视频",
	"xinpianchang": "新片场",
	"zuiyou":       "最右",
}

// platformInfo 平台信息
type platformInfo struct {
	Source   string `json:"source"`
	Name     string `json:"name"`
	URLParse bool   `json:"url_parse"`
	IDParse  bool   `json:"id_parse"`
}

// healthHandler 健康检查
func healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "ok",
		"version":   Version,
		"platforms": len(parser.VideoSourceInfoMapping),
	})
}

// platformsHandler 支持平台列表
func platformsHandler(c *gin.Context) {
	platforms := make([]platformInfo, 0, len(parser.VideoSourceInfoMapping))
	for source := range parser.VideoSourceInfoMapping {
		info := parser.VideoSourceInfoMapping[source]
		name := source
		if n, ok := platformNames[source]; ok {
			name = n
		}
		platforms = append(platforms, platformInfo{
			Source:   source,
			Name:     name,
			URLParse: info.VideoShareUrlParser != nil,
			IDParse:  info.VideoIdParser != nil,
		})
	}
	sort.Slice(platforms, func(i, j int) bool {
		return platforms[i].Source < platforms[j].Source
	})
	sendSuccess(c, platforms)
}

// v1ParseURLHandler v1 分享链接解析
func v1ParseURLHandler(c *gin.Context) {
	rawURL := c.Query("url")
	if rawURL == "" {
		sendError(c, 400, ErrMissingParameter, "url 参数缺失")
		return
	}

	// URL 提取预验证
	extractedURL, err := utils.RegexpMatchUrlFromString(rawURL)
	if err != nil {
		sendError(c, 400, ErrUnsupportedURL, "无法从输入中提取有效链接")
		return
	}

	// 平台域名匹配预验证
	if !matchPlatform(extractedURL) {
		sendError(c, 400, ErrUnsupportedURL, "该链接无法识别对应平台")
		return
	}

	info, err := parseVideoShareURL(rawURL)
	if err != nil {
		status, code := classifyParseError(err)
		sendError(c, status, code, err.Error())
		return
	}
	sendSuccess(c, info)
}

// v1ParseIDHandler v1 视频 ID 解析
func v1ParseIDHandler(c *gin.Context) {
	source := c.Param("source")
	videoID := c.Param("video_id")

	info, exists := parser.VideoSourceInfoMapping[source]
	if !exists {
		sendError(c, 400, ErrUnsupportedSource, "未知的平台: "+source)
		return
	}
	if info.VideoIdParser == nil {
		sendError(c, 400, ErrIDParseNotSupported, "该平台暂不支持视频 ID 解析")
		return
	}

	parseInfo, err := parseVideoID(source, videoID)
	if err != nil {
		status, code := classifyParseError(err)
		sendError(c, status, code, err.Error())
		return
	}
	sendSuccess(c, parseInfo)
}

// matchPlatform 检查 URL 的 host 是否匹配已知平台域名
func matchPlatform(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	for _, sourceInfo := range parser.VideoSourceInfoMapping {
		for _, domain := range sourceInfo.VideoShareUrlDomain {
			domain = strings.ToLower(domain)
			if host == domain || strings.HasSuffix(host, "."+domain) {
				return true
			}
		}
	}
	return false
}

// legacyParseURLHandler 旧路由适配器：分享链接解析
func legacyParseURLHandler(c *gin.Context) {
	rawURL := c.Query("url")
	parseRes, err := parser.ParseVideoShareUrlByRegexp(rawURL)
	if err != nil {
		c.JSON(200, gin.H{"code": 201, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 200, "msg": "解析成功", "data": parseRes})
}

// legacyParseIDHandler 旧路由适配器：视频 ID 解析
func legacyParseIDHandler(c *gin.Context) {
	source := c.Query("source")
	videoID := c.Query("video_id")
	parseRes, err := parser.ParseVideoId(source, videoID)
	if err != nil {
		c.JSON(200, gin.H{"code": 201, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 200, "msg": "解析成功", "data": parseRes})
}

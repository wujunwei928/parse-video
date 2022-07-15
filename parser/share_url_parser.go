package parser

import (
	"fmt"
	"strings"
)

var parseShareUrlMapping = map[string]videoShareUrlParser{
	SourceDouYin:      douYin{},
	SourceKuaiShou:    kuaiShou{},
	SourceZuiYou:      zuiYou{},
	SourceXiGua:       xiGua{},
	SourcePiPiXia:     piPiXia{},
	SourceWeiShi:      weiShi{},
	SourceHuoShan:     huoShan{},
	SourceLiShiPin:    liShiPin{},
	SourcePiPiGaoXiao: piPiGaoXiao{},
}

// 分享链接中, 域名和来源映射信息
var shareUrlSourceDomainMapping = map[string]string{
	SourceDouYin:      "v.douyin.com",
	SourceKuaiShou:    "v.kuaishou.com",
	SourceZuiYou:      "share.xiaochuankeji.cn",
	SourceXiGua:       "v.ixigua.com",
	SourcePiPiXia:     "h5.pipix.com",
	SourceWeiShi:      "isee.weishi.qq.com",
	SourceHuoShan:     "share.huoshan.com",
	SourceLiShiPin:    "www.pearvideo.com",
	SourcePiPiGaoXiao: "h5.pipigx.com",
}

func ParseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	// 根据url判断source
	source := ""
	for itemSource, itemDomain := range shareUrlSourceDomainMapping {
		if strings.Contains(shareUrl, itemDomain) {
			source = itemSource
			break
		}
	}
	fmt.Println(source)

	urlParser, ok := parseShareUrlMapping[source]
	if !ok {
		return nil, fmt.Errorf("source %s has no video id parser", source)
	}

	return urlParser.parseShareUrl(shareUrl)
}

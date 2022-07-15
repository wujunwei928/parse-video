package parser

import (
	"errors"
	"fmt"
	"strings"
)

var parseShareUrlMapping = map[string]videoShareUrlParser{
	SourceDouYin:   douYin{},
	SourceKuaiShou: kuaiShou{},
	SourceZuiYou:   zuiYou{},
	SourceXiGua:    xiGua{},
}

// 分享链接中, 域名和来源映射信息
var shareUrlSourceDomainMapping = map[string]string{
	SourceDouYin:   "douyin.com",
	SourceKuaiShou: "kuaishou.com",
	SourceZuiYou:   "xiaochuankeji.cn",
	SourceXiGua:    "v.ixigua.com",
}

func ParseShareUrl(shareUrl string) (*VideoParseInfo, error) {
	if len(shareUrl) <= 0 {
		return nil, errors.New("video id or source is empty")
	}

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

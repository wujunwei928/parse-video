package parser

import (
	"errors"
	"fmt"
	"strings"
)

var parseShareUrlMapping = map[string]videoShareUrlParser{
	SourceDouYin: douYin{},
}

func ParseShareUrl(shareUrl string) ([]*VideoParseInfo, error) {
	if len(shareUrl) <= 0 {
		return nil, errors.New("video id or source is empty")
	}

	// 根据url判断source
	source := ""
	switch {
	case strings.Contains(shareUrl, "douyin.com"):
		source = SourceDouYin
	case strings.Contains(shareUrl, "kuaishou.com"):
		source = SourceKuaiShou
	}

	urlParser, ok := parseShareUrlMapping[source]
	if !ok {
		return nil, fmt.Errorf("source %s has no video id parser", source)
	}

	return urlParser.parseShareUrl(shareUrl)
}

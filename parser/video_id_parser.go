package parser

import (
	"errors"
	"fmt"
)

// 视频来源与对应解析方法映射
// 不是所有的来源都支持id: 目前仅支持抖音
var parseVideoIdFunMapping = map[string]videoIdParser{
	SourceDouYin: douYin{},
	SourceXiGua:  xiGua{},
}

func ParseVideoId(source, videoId string) (*VideoParseInfo, error) {
	if len(videoId) <= 0 || len(source) <= 0 {
		return nil, errors.New("video id or source is empty")
	}

	idParser, ok := parseVideoIdFunMapping[source]
	if !ok {
		return nil, fmt.Errorf("source %s has no video id parser", source)
	}

	return idParser.parseVideoID(videoId)
}

func BatchParseVideoId(source string, videoIds []string) ([]*VideoParseInfo, error) {
	if len(videoIds) <= 0 || len(source) <= 0 {
		return nil, errors.New("videos id or source is empty")
	}

	switch source {
	case SourceDouYin:
		return douYin{}.MultiParseVideoID(videoIds)
	}

	return nil, errors.New("source not support batch parse video id")
}

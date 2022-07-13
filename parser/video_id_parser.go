package parser

import (
	"errors"
	"fmt"
)

var parseVideoIdFunMapping = map[string]videoIdParser{
	SourceDouYin: douYin{},
}

func ParseVideoId(videoId, source string) ([]*VideoParseInfo, error) {
	if len(videoId) <= 0 || len(source) <= 0 {
		return nil, errors.New("video id or source is empty")
	}

	idParser, ok := parseVideoIdFunMapping[source]
	if !ok {
		return nil, fmt.Errorf("source %s has no video id parser", source)
	}

	return idParser.parseVideoID(videoId)
}

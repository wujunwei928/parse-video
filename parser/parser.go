package parser

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ParseVideoShareUrl 根据视频分享链接解析视频信息
func ParseVideoShareUrl(shareUrl string) (*VideoParseInfo, error) {
	// 根据分享url判断source
	source := ""
	for itemSource, itemSourceInfo := range videoSourceInfoMapping {
		if strings.Contains(shareUrl, itemSourceInfo.VideoShareUrlDomain) {
			source = itemSource
			break
		}
	}

	// 没有找到对应source
	if len(source) <= 0 {
		return nil, fmt.Errorf("share url [%s] not have source config", shareUrl)
	}

	// 没有对应的视频链接解析方法
	urlParser := videoSourceInfoMapping[source].VideoShareUrlParser
	if urlParser == nil {
		return nil, fmt.Errorf("source %s has no video share url parser", source)
	}

	return urlParser.parseShareUrl(shareUrl)
}

// ParseVideoId 根据视频id解析视频信息
func ParseVideoId(source, videoId string) (*VideoParseInfo, error) {
	if len(videoId) <= 0 || len(source) <= 0 {
		return nil, errors.New("video id or source is empty")
	}

	idParser := videoSourceInfoMapping[source].VideoIdParser
	if idParser == nil {
		return nil, fmt.Errorf("source %s has no video id parser", source)
	}

	return idParser.parseVideoID(videoId)
}

// BatchParseVideoId 根据视频id批量解析视频信息
func BatchParseVideoId(source string, videoIds []string) (map[string]BatchParseItem, error) {
	if len(videoIds) <= 0 || len(source) <= 0 {
		return nil, errors.New("videos id or source is empty")
	}

	idParser := videoSourceInfoMapping[source].VideoIdParser
	if idParser == nil {
		return nil, fmt.Errorf("source %s has no video id parser", source)
	}

	switch source {
	case SourceDouYin:
		// 抖音接口支持多id请求
		return douYin{}.batchParseVideoID(videoIds)
	default:
		// 其他平台接口不支持id批量, 使用协程方式请求
		var wg sync.WaitGroup
		var mu sync.Mutex
		parseMap := make(map[string]BatchParseItem, len(videoIds))
		for _, v := range videoIds {
			wg.Add(1)
			videoId := v
			go func(videoId string) {
				defer wg.Done()

				parseInfo, parseErr := ParseVideoId(source, videoId)
				mu.Lock()
				parseMap[videoId] = BatchParseItem{
					ParseInfo: parseInfo,
					Error:     parseErr,
				}
				mu.Unlock()
			}(videoId)
		}
		wg.Wait()

		return parseMap, nil
	}

	return nil, errors.New("source not support batch parse video id")
}

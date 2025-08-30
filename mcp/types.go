package mcp

import (
	"github.com/wujunwei928/parse-video/parser"
)

// MCP specific types and constants

// ParseVideoShareURLRequest represents the request for parsing video share URL
type ParseVideoShareURLRequest struct {
	URL string `json:"url" mcp:"description=The video share URL to parse"`
}

// ParseVideoIDRequest represents the request for parsing video by ID
type ParseVideoIDRequest struct {
	Source  string `json:"source" mcp:"description=The video platform source (e.g., douyin, kuaishou)"`
	VideoID string `json:"video_id" mcp:"description=The video ID to parse"`
}

// BatchParseVideoIDRequest represents the request for batch parsing videos by ID
type BatchParseVideoIDRequest struct {
	Source   string   `json:"source" mcp:"description=The video platform source (e.g., douyin, kuaishou)"`
	VideoIDs []string `json:"video_ids" mcp:"description=List of video IDs to parse"`
}

// ExtractURLRequest represents the request for extracting URL from text
type ExtractURLRequest struct {
	Text string `json:"text" mcp:"description=The text containing URLs to extract"`
}

// PlatformInfo represents information about a supported platform
type PlatformInfo struct {
	Source              string   `json:"source"`
	Name                string   `json:"name"`
	Domains             []string `json:"domains"`
	SupportsShareURL    bool     `json:"supports_share_url"`
	SupportsVideoID     bool     `json:"supports_video_id"`
	SupportsBatchParse  bool     `json:"supports_batch_parse"`
}

// ConvertVideoParseInfo converts parser.VideoParseInfo to MCP compatible format
func ConvertVideoParseInfo(info *parser.VideoParseInfo) map[string]interface{} {
	if info == nil {
		return nil
	}

	result := map[string]interface{}{
		"title":      info.Title,
		"video_url":  info.VideoUrl,
		"music_url":  info.MusicUrl,
		"cover_url":  info.CoverUrl,
		"author": map[string]interface{}{
			"uid":    info.Author.Uid,
			"name":   info.Author.Name,
			"avatar": info.Author.Avatar,
		},
	}

	// Add images if present
	if len(info.Images) > 0 {
		images := make([]map[string]interface{}, len(info.Images))
		for i, img := range info.Images {
			images[i] = map[string]interface{}{
				"url":             img.Url,
				"live_photo_url":  img.LivePhotoUrl,
			}
		}
		result["images"] = images
	}

	return result
}

// ConvertBatchParseResult converts parser batch parse result to MCP compatible format
func ConvertBatchParseResult(results map[string]parser.BatchParseItem) map[string]interface{} {
	mcpResults := make(map[string]interface{})
	
	for videoID, item := range results {
		if item.Error != nil {
			mcpResults[videoID] = map[string]interface{}{
				"error": item.Error.Error(),
			}
		} else {
			mcpResults[videoID] = ConvertVideoParseInfo(item.ParseInfo)
		}
	}
	
	return mcpResults
}

// GetPlatformInfo returns platform information for a specific source
func GetPlatformInfo(source string) *PlatformInfo {
	info, exists := parser.VideoSourceInfoMapping[source]
	if !exists {
		return nil
	}

	return &PlatformInfo{
		Source:             source,
		Name:               getPlatformName(source),
		Domains:            info.VideoShareUrlDomain,
		SupportsShareURL:   info.VideoShareUrlParser != nil,
		SupportsVideoID:    info.VideoIdParser != nil,
		SupportsBatchParse: info.VideoIdParser != nil,
	}
}

// GetAllPlatformInfo returns information for all supported platforms
func GetAllPlatformInfo() []PlatformInfo {
	var platforms []PlatformInfo
	
	for source := range parser.VideoSourceInfoMapping {
		if info := GetPlatformInfo(source); info != nil {
			platforms = append(platforms, *info)
		}
	}
	
	return platforms
}

// getPlatformName returns the display name for a platform source
func getPlatformName(source string) string {
	switch source {
	case parser.SourceDouYin:
		return "抖音"
	case parser.SourceKuaiShou:
		return "快手"
	case parser.SourcePiPiXia:
		return "皮皮虾"
	case parser.SourceHuoShan:
		return "火山"
	case parser.SourceWeiBo:
		return "微博"
	case parser.SourceWeiShi:
		return "微视"
	case parser.SourceLvZhou:
		return "绿洲"
	case parser.SourceZuiYou:
		return "最右"
	case parser.SourceQuanMin:
		return "度小视"
	case parser.SourceXiGua:
		return "西瓜视频"
	case parser.SourceLiShiPin:
		return "梨视频"
	case parser.SourcePiPiGaoXiao:
		return "皮皮搞笑"
	case parser.SourceHuYa:
		return "虎牙"
	case parser.SourceAcFun:
		return "A站"
	case parser.SourceDouPai:
		return "逗拍"
	case parser.SourceMeiPai:
		return "美拍"
	case parser.SourceQuanMinKGe:
		return "全民K歌"
	case parser.SourceSixRoom:
		return "六间房"
	case parser.SourceXinPianChang:
		return "新片场"
	case parser.SourceHaoKan:
		return "好看视频"
	case parser.SourceRedBook:
		return "小红书"
	case parser.SourceBiliBili:
		return "哔哩哔哩"
	default:
		return source
	}
}
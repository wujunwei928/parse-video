package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/wujunwei928/parse-video/parser"
	"github.com/wujunwei928/parse-video/utils"
)

// RegisterTools registers all MCP tools for video parsing
func RegisterTools(s *server.MCPServer) {
	// Tool: Parse video share URL
	parseShareURLTool := mcp.NewTool("parse_video_share_url",
		mcp.WithDescription("Parse video information from a share URL"),
		mcp.WithString("url",
			mcp.Description("The video share URL to parse (can be a full URL or text containing the URL)"),
			mcp.Required(),
		),
	)
	s.AddTool(parseShareURLTool, handleParseShareURL)

	// Tool: Parse video by ID
	parseVideoIDTool := mcp.NewTool("parse_video_id",
		mcp.WithDescription("Parse video information by platform source and video ID"),
		mcp.WithString("source",
			mcp.Description("The video platform source (e.g., douyin, kuaishou, bilibili)"),
			mcp.Required(),
		),
		mcp.WithString("video_id",
			mcp.Description("The video ID to parse"),
			mcp.Required(),
		),
	)
	s.AddTool(parseVideoIDTool, handleParseVideoID)

	// Tool: Batch parse videos by ID
	batchParseTool := mcp.NewTool("batch_parse_video_id",
		mcp.WithDescription("Parse multiple videos by platform source and video IDs"),
		mcp.WithString("source",
			mcp.Description("The video platform source (e.g., douyin, kuaishou, bilibili)"),
			mcp.Required(),
		),
		mcp.WithArray("video_ids",
			mcp.Description("List of video IDs to parse"),
			mcp.Required(),
		),
	)
	s.AddTool(batchParseTool, handleBatchParseVideoID)

	// Tool: Extract URL from text
	extractURLTool := mcp.NewTool("extract_url_from_text",
		mcp.WithDescription("Extract URLs from text content"),
		mcp.WithString("text",
			mcp.Description("The text containing URLs to extract"),
			mcp.Required(),
		),
	)
	s.AddTool(extractURLTool, handleExtractURL)

	// Tool: Get supported platforms
	getPlatformsTool := mcp.NewTool("get_supported_platforms",
		mcp.WithDescription("Get list of all supported video platforms"),
	)
	s.AddTool(getPlatformsTool, handleGetSupportedPlatforms)
}

// handleParseShareURL handles the parse_video_share_url tool
func handleParseShareURL(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url := req.GetString("url", "")
	if url == "" {
		return mcp.NewToolResultError("URL parameter is required"), nil
	}

	// Try to extract URL from text if needed
	videoURL, err := utils.RegexpMatchUrlFromString(url)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to extract URL: %v", err)), nil
	}

	// Parse the video
	result, err := parser.ParseVideoShareUrl(videoURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse video: %v", err)), nil
	}

	// Convert to MCP format
	mcpResult := ConvertVideoParseInfo(result)

	return mcp.NewToolResultText(fmt.Sprintf("Successfully parsed video:\n%v", mcpResult)), nil
}

// handleParseVideoID handles the parse_video_id tool
func handleParseVideoID(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	source := req.GetString("source", "")
	videoID := req.GetString("video_id", "")

	if source == "" || videoID == "" {
		return mcp.NewToolResultError("Both source and video_id parameters are required"), nil
	}

	// Validate source
	if _, exists := parser.VideoSourceInfoMapping[source]; !exists {
		return mcp.NewToolResultError(fmt.Sprintf("Unsupported platform source: %s", source)), nil
	}

	// Parse the video
	result, err := parser.ParseVideoId(source, videoID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse video: %v", err)), nil
	}

	// Convert to MCP format
	mcpResult := ConvertVideoParseInfo(result)

	return mcp.NewToolResultText(fmt.Sprintf("Successfully parsed video:\n%v", mcpResult)), nil
}

// handleBatchParseVideoID handles the batch_parse_video_id tool
func handleBatchParseVideoID(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	source := req.GetString("source", "")
	videoIDs := req.GetStringSlice("video_ids", []string{})

	if source == "" || len(videoIDs) == 0 {
		return mcp.NewToolResultError("Both source and video_ids parameters are required"), nil
	}

	// Validate source
	if _, exists := parser.VideoSourceInfoMapping[source]; !exists {
		return mcp.NewToolResultError(fmt.Sprintf("Unsupported platform source: %s", source)), nil
	}

	// Batch parse videos
	results, err := parser.BatchParseVideoId(source, videoIDs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to batch parse videos: %v", err)), nil
	}

	// Convert to MCP format
	mcpResults := ConvertBatchParseResult(results)

	return mcp.NewToolResultText(fmt.Sprintf("Batch parse results:\n%v", mcpResults)), nil
}

// handleExtractURL handles the extract_url_from_text tool
func handleExtractURL(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	text := req.GetString("text", "")
	if text == "" {
		return mcp.NewToolResultError("Text parameter is required"), nil
	}

	// Extract URLs from text
	url, err := utils.RegexpMatchUrlFromString(text)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to extract URL: %v", err)), nil
	}

	// Find platform for the URL
	var platform string
	for source, info := range parser.VideoSourceInfoMapping {
		for _, domain := range info.VideoShareUrlDomain {
			if strings.Contains(url, domain) {
				platform = source
				break
			}
		}
		if platform != "" {
			break
		}
	}

	result := map[string]interface{}{
		"extracted_url": url,
		"platform":      platform,
		"platform_name": getPlatformName(platform),
	}

	return mcp.NewToolResultText(fmt.Sprintf("Extracted URL: %v", result)), nil
}

// handleGetSupportedPlatforms handles the get_supported_platforms tool
func handleGetSupportedPlatforms(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	platforms := GetAllPlatformInfo()

	return mcp.NewToolResultText(fmt.Sprintf("Supported platforms:\n%v", platforms)), nil
}

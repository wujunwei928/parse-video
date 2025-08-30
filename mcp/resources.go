package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterResources registers all MCP resources for platform information
func RegisterResources(s *server.MCPServer) {
	// Resource: List all platforms
	platformsResource := mcp.NewResource(
		"platforms://",
		"all-platforms",
		mcp.WithResourceDescription("List of all supported video platforms"),
		mcp.WithMIMEType("application/json"),
	)
	s.AddResource(platformsResource, handlePlatformsResource)

	// Resource: Individual platform information
	platformResourceTemplate := mcp.NewResourceTemplate(
		"platforms://{source}",
		"platform-info",
		mcp.WithTemplateDescription("Detailed information about a specific video platform"),
		mcp.WithTemplateMIMEType("application/json"),
	)
	s.AddResourceTemplate(platformResourceTemplate, handlePlatformResource)
}

// handlePlatformsResource handles the platforms:// resource
func handlePlatformsResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	platforms := GetAllPlatformInfo()
	
	platformsData := map[string]interface{}{
		"total_platforms": len(platforms),
		"platforms":       platforms,
	}
	
	jsonData, err := json.MarshalIndent(platformsData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal platforms data: %v", err)
	}

	return []mcp.ResourceContents{
		&mcp.TextResourceContents{
			URI:      "platforms://",
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// handlePlatformResource handles the platforms://{source} resource
func handlePlatformResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract source from URI
	uri := req.Params.URI
	if !isPlatformResourceURI(uri) {
		return nil, fmt.Errorf("invalid platform resource URI: %s", uri)
	}

	source := extractSourceFromURI(uri)
	if source == "" {
		return nil, fmt.Errorf("cannot extract source from URI: %s", uri)
	}

	platformInfo := GetPlatformInfo(source)
	if platformInfo == nil {
		return nil, fmt.Errorf("platform not found: %s", source)
	}

	jsonData, err := json.MarshalIndent(platformInfo, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal platform data: %v", err)
	}

	return []mcp.ResourceContents{
		&mcp.TextResourceContents{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// isPlatformResourceURI checks if the URI matches the platform resource pattern
func isPlatformResourceURI(uri string) bool {
	return uri == "platforms://" || (len(uri) > len("platforms://") && uri[:len("platforms://")] == "platforms://")
}

// extractSourceFromURI extracts the platform source from a resource URI
func extractSourceFromURI(uri string) string {
	if uri == "platforms://" {
		return ""
	}
	
	// Remove "platforms://" prefix
	if len(uri) > len("platforms://") {
		return uri[len("platforms://"):]
	}
	
	return ""
}
package parser

import (
	"testing"
)

// TestSohuExtractVid_Base64URL 测试从 base64 编码 URL 提取视频 ID
func TestSohuExtractVid_Base64URL(t *testing.T) {
	s := sohuVideo{}
	tests := []struct {
		name     string
		url      string
		expected string
		hasError bool
	}{
		{
			name:     "base64编码URL",
			url:      "https://tv.sohu.com/v/dXMvMzM1OTQyMjE0LzM5OTU3MTYxMi5zaHRtbA==.html",
			expected: "399571612",
		},
		{
			name:     "直接路径URL",
			url:      "http://my.tv.sohu.com/us/335942214/399571612.shtml",
			expected: "399571612",
		},
		{
			name:     "无效URL",
			url:      "https://tv.sohu.com/v/invalid.html",
			hasError: true,
		},
		{
			name:     "非搜狐链接",
			url:      "https://www.douyin.com/video/123",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vid, err := s.extractVid(tt.url)
			if tt.hasError {
				if err == nil {
					t.Errorf("期望返回错误，但得到了 vid: %s", vid)
				}
				return
			}
			if err != nil {
				t.Errorf("不期望返回错误: %v", err)
				return
			}
			if vid != tt.expected {
				t.Errorf("期望 vid=%s, 实际 vid=%s", tt.expected, vid)
			}
		})
	}
}

// TestSohuExtractVidFromPath 测试从解码路径提取视频 ID
func TestSohuExtractVidFromPath(t *testing.T) {
	s := sohuVideo{}
	tests := []struct {
		name     string
		path     string
		expected string
		hasError bool
	}{
		{
			name:     "标准路径",
			path:     "us/335942214/399571612.shtml",
			expected: "399571612",
		},
		{
			name:     "带斜杠前缀",
			path:     "/us/335942214/399571612.shtml",
			expected: "399571612",
		},
		{
			name:     "无效路径",
			path:     "invalid/path",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vid, err := s.extractVidFromPath(tt.path)
			if tt.hasError {
				if err == nil {
					t.Errorf("期望返回错误，但得到了 vid: %s", vid)
				}
				return
			}
			if err != nil {
				t.Errorf("不期望返回错误: %v", err)
				return
			}
			if vid != tt.expected {
				t.Errorf("期望 vid=%s, 实际 vid=%s", tt.expected, vid)
			}
		})
	}
}

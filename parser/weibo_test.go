package parser

import (
	"testing"
)

func TestWeiBo_parseShareUrl(t *testing.T) {
	w := weiBo{}

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "Image album URL",
			url:     "https://weibo.com/2543858012/Q9pcJ4S21",
			wantErr: false,
		},
		{
			name:    "Video URL with fid parameter",
			url:     "https://video.weibo.com/show?fid=1034:4808181919187090",
			wantErr: false,
		},
		{
			name:    "TV show URL",
			url:     "https://weibo.com/tv/show/1034:4808181919187090",
			wantErr: false,
		},
		{
			name:    "Invalid URL",
			url:     "https://example.com/invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := w.parseShareUrl(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("weiBo.parseShareUrl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("weiBo.parseShareUrl() returned nil result without error")
				return
			}
			if !tt.wantErr && got != nil {
				if got.Title == "" && got.VideoUrl == "" && len(got.Images) == 0 {
					t.Error("weiBo.parseShareUrl() returned empty result (no title, video, or images)")
				}
			}
		})
	}
}

func TestWeiBo_cleanText(t *testing.T) {
	w := weiBo{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Text with HTML tags",
			input:    "<span class=\"text\">Hello World</span>",
			expected: "Hello World",
		},
		{
			name:     "Text with multiple tags",
			input:    "<div><p>Hello <strong>World</strong></p></div>",
			expected: "Hello World",
		},
		{
			name:     "Plain text",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "Text with whitespace",
			input:    "  Hello World  ",
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.cleanText(tt.input)
			if got != tt.expected {
				t.Errorf("weiBo.cleanText() = %v, want %v", got, tt.expected)
			}
		})
	}
}
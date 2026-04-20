package parser

import (
	"testing"
)

// TestCCTVExtractGuid 从模拟 HTML 中提取 GUID
func TestCCTVExtractGuid(t *testing.T) {
	tests := []struct {
		name    string
		html    string
		want    string
		wantErr bool
	}{
		{
			name: "标准格式",
			html: `<script>var guid = "68c27e1af8cc47f79000ca944432b0e6";</script>`,
			want: "68c27e1af8cc47f79000ca944432b0e6",
		},
		{
			name: "无空格",
			html: `<script>var guid="abc123def456";</script>`,
			want: "abc123def456",
		},
		{
			name: "多空格",
			html: `<script>var  guid  =  "multiSpaceGuid";</script>`,
			want: "multiSpaceGuid",
		},
		{
			name:    "页面无GUID",
			html:    `<html><body>no guid here</body></html>`,
			wantErr: true,
		},
		{
			name:    "空GUID",
			html:    `<script>var guid = "";</script>`,
			wantErr: true,
		},
	}

	c := cctvVideo{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 使用正则直接测试提取逻辑
			got, err := c.extractGuidFromHTML(tt.html)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractGuidFromHTML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractGuidFromHTML() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCCTVSourceMapping 验证 CCTV 平台在映射中正确注册
func TestCCTVSourceMapping(t *testing.T) {
	info, ok := videoSourceInfoMapping[SourceCCTV]
	if !ok {
		t.Fatal("CCTV 平台未在 videoSourceInfoMapping 中注册")
	}

	if len(info.VideoShareUrlDomain) == 0 {
		t.Fatal("CCTV 平台未配置域名")
	}

	expectedDomains := []string{"tv.cctv.cn", "tv.cctv.com"}
	for _, domain := range expectedDomains {
		found := false
		for _, d := range info.VideoShareUrlDomain {
			if d == domain {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("域名 %s 未在 CCTV 域名列表中注册", domain)
		}
	}

	if info.VideoShareUrlParser == nil {
		t.Fatal("CCTV 平台未配置 URL 解析器")
	}

	if info.VideoIdParser == nil {
		t.Fatal("CCTV 平台未配置 ID 解析器")
	}
}

// TestCCTVVideoIDEmpty 验证空 GUID 返回错误
func TestCCTVVideoIDEmpty(t *testing.T) {
	c := cctvVideo{}
	_, err := c.parseVideoID("")
	if err == nil {
		t.Error("空 GUID 应返回错误")
	}
}

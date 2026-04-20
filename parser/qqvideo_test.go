package parser

import (
	"testing"
)

func Test_qqVideo_extractVid(t *testing.T) {
	qv := qqVideo{}
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{"PC端 page 链接", "https://v.qq.com/x/page/l3502vppd13.html", "l3502vppd13", false},
		{"PC端 page 带参数", "https://v.qq.com/x/page/l3502vppd13.html?ptag=v_qq_com", "l3502vppd13", false},
		{"PC端 cover 链接", "https://v.qq.com/x/cover/mzc00200mp9v9pw/l3502vppd13.html", "l3502vppd13", false},
		{"PC端 cover 带参数", "https://v.qq.com/x/cover/mzc00200mp9v9pw/l3502vppd13.html?ptag=v_qq_com", "l3502vppd13", false},
		{"移动端播放页", "https://m.v.qq.com/x/m/play?cid=&vid=l3502vppd13&ptag=v_qq_com", "l3502vppd13", false},
		{"无效路径", "https://v.qq.com/x/channel/home", "", true},
		{"移动端缺少vid", "https://m.v.qq.com/x/m/play?cid=abc", "", true},
		{"空字符串", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := qv.extractVid(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractVid() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractVid() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_qqVideo_extractVidFromPath(t *testing.T) {
	qv := qqVideo{}
	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{"/x/page/ 短视频路径", "/x/page/l3502vppd13.html", "l3502vppd13", false},
		{"/x/cover/ 长视频路径", "/x/cover/mzc00200mp9v9pw/l3502vppd13.html", "l3502vppd13", false},
		{"路径末尾带斜杠", "/x/page/l3502vppd13.html/", "l3502vppd13", false},
		{"无效路径", "/x/channel/home", "", true},
		{"空路径", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := qv.extractVidFromPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractVidFromPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractVidFromPath() got = %v, want %v", got, tt.want)
			}
		})
	}
}

package parser

import (
	"testing"
)

// 移动端 UA 返回的页面结构：笔记数据位于 noteData.data.noteData
const redBookMobileHTML = `<html><script>window.__INITIAL_STATE__={"noteData":{"data":{"noteData":{
	"title":"移动端笔记标题","type":"normal",
	"user":{"userId":"uid123","nickName":"移动端作者","avatar":"https://avatar.example.com/a.jpg"},
	"video":{"media":{"stream":{}}},
	"imageList":[
		{"url":"https://sns-webpic-qc.xhscdn.com/202609281327/abc/spectrum/file1!h5_1080jpg","livePhoto":false,"stream":{}},
		{"url":"https://sns-webpic-qc.xhscdn.com/202609281327/abc/spectrum/file2!h5_1080jpg","livePhoto":true,
		 "stream":{"h264":[{"masterUrl":"https://sns-video.example.com/livephoto1.mp4"}]}}
	]
}}}}</script></html>`

// 桌面端页面结构（旧）：笔记数据位于 note.noteDetailMap.<noteId>.note
const redBookDesktopHTML = `<html><script>window.__INITIAL_STATE__={"note":{"currentNoteId":"note1",
	"noteDetailMap":{"note1":{"note":{
	"title":"桌面端笔记标题",
	"user":{"userId":"uid456","nickname":"桌面端作者","avatar":"https://avatar.example.com/b.jpg"},
	"video":{"media":{"stream":{"h264":[{"masterUrl":"https://sns-video.example.com/desktop.mp4"}]}}},
	"imageList":[{"urlDefault":"https://sns-webpic-qc.xhscdn.com/notes_pre_post/file3!h5_1080jpg"}]
}}}}}</script></html>`

// 登录页空壳：__INITIAL_STATE__ 存在但无笔记数据
const redBookLoginHTML = `<html><script>window.__INITIAL_STATE__={"global":{"appSettings":{"fakeModalPercent":0.02}}}</script></html>`

// 双流视频笔记：小红书服务端为 h264 分享流烧录了水印（作者昵称+小红书 logo），h265 流无水印
const redBookDualStreamHTML = `<html><script>window.__INITIAL_STATE__={"noteData":{"data":{"noteData":{
	"title":"双流视频笔记",
	"user":{"userId":"u1","nickName":"a"},
	"video":{"media":{"stream":{
		"h264":[{"masterUrl":"https://sns-video.example.com/watermarked.mp4"}],
		"h265":[{"masterUrl":"https://sns-video.example.com/clean.mp4"}]
	}}},
	"imageList":[]
}}}}</script></html>`

// 仅 h264 流的老页面
const redBookH264OnlyHTML = `<html><script>window.__INITIAL_STATE__={"noteData":{"data":{"noteData":{
	"title":"仅h264流",
	"user":{"userId":"u1","nickName":"a"},
	"video":{"media":{"stream":{
		"h264":[{"masterUrl":"https://sns-video.example.com/legacy.mp4"}]
	}}},
	"imageList":[]
}}}}</script></html>`

// 视频流选择：优先 h265 无水印流，仅 h264 时回退
func Test_redBook_parseNoteInfoFromHtml_videoStreamSelect(t *testing.T) {
	tests := []struct {
		name         string
		html         string
		wantVideoUrl string
	}{
		{
			// 回归：h264 分享流带服务端烧录水印，必须优先取 h265 无水印流
			name:         "双流时优先h265无水印流",
			html:         redBookDualStreamHTML,
			wantVideoUrl: "https://sns-video.example.com/clean.mp4",
		},
		{
			name:         "仅h264流时回退h264",
			html:         redBookH264OnlyHTML,
			wantVideoUrl: "https://sns-video.example.com/legacy.mp4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parseNoteInfoFromHtml([]byte(tt.html))
			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}
			if info.VideoUrl != tt.wantVideoUrl {
				t.Errorf("VideoUrl = %q, want %q", info.VideoUrl, tt.wantVideoUrl)
			}
		})
	}
}

func Test_redBook_parseNoteInfoFromHtml(t *testing.T) {
	tests := []struct {
		name         string
		html         string
		wantErr      bool
		wantTitle    string
		wantAuthor   string
		wantVideoUrl string
		wantImages   int
	}{
		{
			name:         "移动端结构-图集笔记",
			html:         redBookMobileHTML,
			wantTitle:    "移动端笔记标题",
			wantAuthor:   "移动端作者",
			wantVideoUrl: "",
			wantImages:   2,
		},
		{
			name:         "桌面端结构-视频笔记",
			html:         redBookDesktopHTML,
			wantTitle:    "桌面端笔记标题",
			wantAuthor:   "桌面端作者",
			wantVideoUrl: "https://sns-video.example.com/desktop.mp4",
			wantImages:   0,
		},
		{
			name:    "登录页空壳-返回解析失败",
			html:    redBookLoginHTML,
			wantErr: true,
		},
		{
			name:    "无INITIAL_STATE-返回解析失败",
			html:    `<html><body>empty</body></html>`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parseNoteInfoFromHtml([]byte(tt.html))
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNoteInfoFromHtml() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if info.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", info.Title, tt.wantTitle)
			}
			if info.Author.Name != tt.wantAuthor {
				t.Errorf("Author.Name = %q, want %q", info.Author.Name, tt.wantAuthor)
			}
			if info.VideoUrl != tt.wantVideoUrl {
				t.Errorf("VideoUrl = %q, want %q", info.VideoUrl, tt.wantVideoUrl)
			}
			if len(info.Images) != tt.wantImages {
				t.Errorf("Images 数量 = %d, want %d", len(info.Images), tt.wantImages)
			}
		})
	}
}

func Test_redBook_parseNoteInfoFromHtml_images(t *testing.T) {
	t.Run("移动端图片URL不含notes_pre_post时使用原URL", func(t *testing.T) {
		info, err := parseNoteInfoFromHtml([]byte(redBookMobileHTML))
		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}
		if len(info.Images) != 2 {
			t.Fatalf("Images 数量 = %d, want 2", len(info.Images))
		}
		wantUrl := "https://sns-webpic-qc.xhscdn.com/202609281327/abc/spectrum/file1!h5_1080jpg"
		if info.Images[0].Url != wantUrl {
			t.Errorf("Images[0].Url = %q, want %q", info.Images[0].Url, wantUrl)
		}
		if info.Images[1].LivePhotoUrl != "https://sns-video.example.com/livephoto1.mp4" {
			t.Errorf("Images[1].LivePhotoUrl = %q, want %q", info.Images[1].LivePhotoUrl, "https://sns-video.example.com/livephoto1.mp4")
		}
	})

	t.Run("桌面端图片URL含notes_pre_post时替换为高清域名", func(t *testing.T) {
		// 桌面端图集笔记：无视频流，imageList 使用 urlDefault
		html := `<script>window.__INITIAL_STATE__={"noteData":{"data":{"noteData":{"title":"桌面图集",
			"user":{"userId":"u","nickName":"a"},
			"imageList":[{"urlDefault":"https://sns-webpic-qc.xhscdn.com/notes_pre_post/file3!h5_1080jpg"}]
			}}}}</script>`
		info, err := parseNoteInfoFromHtml([]byte(html))
		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}
		if len(info.Images) != 1 {
			t.Fatalf("Images 数量 = %d, want 1", len(info.Images))
		}
		wantUrl := "https://ci.xiaohongshu.com/notes_pre_post/file3?imageView2/format/jpg"
		if info.Images[0].Url != wantUrl {
			t.Errorf("Images[0].Url = %q, want %q", info.Images[0].Url, wantUrl)
		}
	})

	t.Run("结构存在但无任何媒体数据时返回解析失败", func(t *testing.T) {
		html := `<script>window.__INITIAL_STATE__={"noteData":{"data":{"noteData":{"title":"空笔记","user":{}}}}}</script>`
		if _, err := parseNoteInfoFromHtml([]byte(html)); err == nil {
			t.Error("无视频且无图片时应返回错误, got nil")
		}
	})
}

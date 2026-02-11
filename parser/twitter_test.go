package parser

import (
	"testing"
)

func Test_twitter_getToken(t *testing.T) {
	tw := twitter{}
	tests := []struct {
		name    string
		tweetId string
	}{
		{"普通推文ID", "1849000000000000000"},
		{"短ID", "123456789"},
		{"长ID", "1879553847283155177"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := tw.getToken(tt.tweetId)
			if len(token) == 0 {
				t.Errorf("getToken() returned empty token for id: %s", tt.tweetId)
			}
			t.Logf("tweetId=%s, token=%s", tt.tweetId, token)
		})
	}
}

func Test_twitter_extractTweetId(t *testing.T) {
	tw := twitter{}
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{"x.com 标准链接", "https://x.com/elonmusk/status/1849000000000000000", "1849000000000000000", false},
		{"twitter.com 标准链接", "https://twitter.com/elonmusk/status/1849000000000000000", "1849000000000000000", false},
		{"带查询参数", "https://x.com/user/status/1849000000000000000?s=20", "1849000000000000000", false},
		{"mobile.twitter.com", "https://mobile.twitter.com/user/status/1849000000000000000", "1849000000000000000", false},
		{"无效链接", "https://x.com/user/likes", "", true},
		{"空字符串", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tw.extractTweetId(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractTweetId() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractTweetId() got = %v, want %v", got, tt.want)
			}
		})
	}
}

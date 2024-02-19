package parser

import (
	"testing"
)

func Test_douYin_parseIdFromPath(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"抖音视频", args{"/share/video/7329354490828623130/"}, "7329354490828623130", false},
		{"西瓜视频", args{"/douyin/share/video/7144194760184594977"}, "7144194760184594977", false},
		{"异常视频", args{""}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := douYin{}
			got, err := d.parseVideoIdFromPath(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVideoIdFromPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseVideoIdFromPath() got = %v, want %v", got, tt.want)
			}
		})
	}
}

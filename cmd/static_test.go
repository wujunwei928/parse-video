package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

// TestRegisterStaticRoutes 验证 staticFS 非空时挂载 /static，
// 并对常见扩展名返回正确 MIME。
func TestRegisterStaticRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// 保存并恢复全局 staticFS，避免污染其它测试
	orig := staticFS
	t.Cleanup(func() { staticFS = orig })
	staticFS = fstest.MapFS{
		"css/base.css": {Data: []byte("body{color:#000}")},
		"js/app.js":    {Data: []byte("console.log(1)")},
		"favicon.png":  {Data: []byte("\x89PNG fake")},
	}
	registerStaticRoutes(r)

	cases := []struct {
		path     string
		wantCode int
		wantCT   string // Content-Type 应包含此子串（wantCode=200 时校验）
	}{
		{"/static/css/base.css", 200, "text/css"},
		{"/static/js/app.js", 200, "javascript"},
		{"/static/favicon.png", 200, "image/png"},
		{"/static/missing.css", 404, ""},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		r.ServeHTTP(w, req)
		if w.Code != c.wantCode {
			t.Errorf("%s 状态码=%d, 期望 %d", c.path, w.Code, c.wantCode)
			continue
		}
		if c.wantCode == 200 {
			ct := w.Header().Get("Content-Type")
			if !strings.Contains(ct, c.wantCT) {
				t.Errorf("%s Content-Type=%q, 期望包含 %q", c.path, ct, c.wantCT)
			}
		}
	}
}

// TestRegisterStaticRoutesNil 验证 staticFS 为空时安全跳过，不 panic。
func TestRegisterStaticRoutesNil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	orig := staticFS
	t.Cleanup(func() { staticFS = orig })
	staticFS = nil
	registerStaticRoutes(r) // 不应 panic
}

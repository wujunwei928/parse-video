package cmd

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/health", nil)

	healthHandler(c)

	if w.Code != 200 {
		t.Errorf("health 应返回 200，实际: %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"status":"ok"`) {
		t.Errorf("health 应返回 status:ok，实际: %s", body)
	}
	if !strings.Contains(body, `"platforms"`) {
		t.Errorf("health 应包含 platforms 字段，实际: %s", body)
	}
}

func TestPlatformsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/platforms", nil)

	platformsHandler(c)

	if w.Code != 200 {
		t.Errorf("platforms 应返回 200，实际: %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	if resp["status"] != "success" {
		t.Errorf("status 应为 success，实际: %v", resp["status"])
	}
	data := resp["data"].([]any)
	if len(data) == 0 {
		t.Error("平台列表不应为空")
	}
	first := data[0].(map[string]any)
	if first["source"] != "acfun" {
		t.Errorf("第一个平台应为 acfun（字母序），实际: %v", first["source"])
	}
	for _, p := range data {
		pm := p.(map[string]any)
		if pm["source"] == "douyin" {
			if pm["id_parse"] != true {
				t.Errorf("douyin 应支持 ID 解析")
			}
		}
		if pm["source"] == "kuaishou" {
			if pm["id_parse"] != false {
				t.Errorf("kuaishou 不应支持 ID 解析")
			}
		}
	}
}

func TestV1ParseURLHandlerMissingURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/parse", nil)

	v1ParseURLHandler(c)

	if w.Code != 400 {
		t.Errorf("缺少 url 应返回 400，实际: %d", w.Code)
	}
	if !containsJSONField(w.Body.String(), "code", ErrMissingParameter) {
		t.Errorf("应返回 MISSING_PARAMETER，实际: %s", w.Body.String())
	}
}

func TestV1ParseURLHandlerInvalidURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/parse?url=not-a-url", nil)

	v1ParseURLHandler(c)

	if w.Code != 400 {
		t.Errorf("无效 URL 应返回 400，实际: %d", w.Code)
	}
	if !containsJSONField(w.Body.String(), "code", ErrUnsupportedURL) {
		t.Errorf("应返回 UNSUPPORTED_URL，实际: %s", w.Body.String())
	}
}

func TestV1ParseURLHandlerUnknownDomain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/parse?url=https://unknown-platform.com/video/123", nil)

	v1ParseURLHandler(c)

	if w.Code != 400 {
		t.Errorf("未知平台应返回 400，实际: %d", w.Code)
	}
	if !containsJSONField(w.Body.String(), "code", ErrUnsupportedURL) {
		t.Errorf("应返回 UNSUPPORTED_URL，实际: %s", w.Body.String())
	}
}

func TestV1ParseURLHandlerDomainMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// 域名包含但不等于平台域名（如 notbilibili.com ≠ bilibili.com）
	c.Request = httptest.NewRequest("GET", "/api/v1/parse?url=https://notbilibili.com/video/123", nil)

	v1ParseURLHandler(c)

	if w.Code != 400 {
		t.Errorf("域名不精确匹配应返回 400，实际: %d", w.Code)
	}
	if !containsJSONField(w.Body.String(), "code", ErrUnsupportedURL) {
		t.Errorf("应返回 UNSUPPORTED_URL，实际: %s", w.Body.String())
	}
}

func TestMatchPlatformSubdomainHost(t *testing.T) {
	if !matchPlatform("https://www.bilibili.com/video/BV1xx411c7mD") {
		t.Error("www.bilibili.com 应匹配 bilibili.com 域名配置")
	}
	if matchPlatform("https://notbilibili.com/video/123") {
		t.Error("notbilibili.com 不应被识别为 bilibili 平台")
	}
}

func TestV1ParseIDHandlerUnknownSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/parse/unknown/123", nil)
	c.Params = gin.Params{{Key: "source", Value: "unknown"}, {Key: "video_id", Value: "123"}}

	v1ParseIDHandler(c)

	if w.Code != 400 {
		t.Errorf("未知平台应返回 400，实际: %d", w.Code)
	}
	if !containsJSONField(w.Body.String(), "code", ErrUnsupportedSource) {
		t.Errorf("应返回 UNSUPPORTED_SOURCE，实际: %s", w.Body.String())
	}
}

func TestV1ParseIDHandlerNoIDParse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/parse/kuaishou/123", nil)
	c.Params = gin.Params{{Key: "source", Value: "kuaishou"}, {Key: "video_id", Value: "123"}}

	v1ParseIDHandler(c)

	if w.Code != 400 {
		t.Errorf("不支持 ID 解析应返回 400，实际: %d", w.Code)
	}
	if !containsJSONField(w.Body.String(), "code", ErrIDParseNotSupported) {
		t.Errorf("应返回 ID_PARSE_NOT_SUPPORTED，实际: %s", w.Body.String())
	}
}

func TestLegacyParseURLHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/video/share/url/parse?url=not-a-url", nil)

	legacyParseURLHandler(c)

	// 旧路由始终 200
	if w.Code != 200 {
		t.Errorf("旧路由应始终 200，实际: %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != float64(201) {
		t.Errorf("解析失败 code 应为 201，实际: %v", resp["code"])
	}
}

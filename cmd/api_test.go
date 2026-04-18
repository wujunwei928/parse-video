package cmd

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/wujunwei928/parse-video/parser"
)

// stubParserFuncs 替换 parser 函数变量，测试结束后自动恢复
func stubParserFuncs(t *testing.T,
	share func(string) (*parser.VideoParseInfo, error),
	id func(string, string) (*parser.VideoParseInfo, error),
) {
	oldShare := parseVideoShareURL
	oldID := parseVideoID
	t.Cleanup(func() {
		parseVideoShareURL = oldShare
		parseVideoID = oldID
	})
	if share != nil {
		parseVideoShareURL = share
	}
	if id != nil {
		parseVideoID = id
	}
}

// setupTestRouter 创建完整配置的测试路由（含完整中间件栈）
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(recoveryMiddleware())
	r.Use(corsMiddleware("*"))
	r.Use(requestLogMiddleware())
	r.Use(rateLimitMiddleware(newIPRateLimiterWithBurst(600, 100), "/api/v1/health"))

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", healthHandler)
		v1.GET("/platforms", platformsHandler)
		v1.GET("/parse", v1ParseURLHandler)
		v1.GET("/parse/:source/:video_id", v1ParseIDHandler)
	}

	r.GET("/video/share/url/parse", legacyParseURLHandler)
	r.GET("/video/id/parse", legacyParseIDHandler)
	r.GET("/", func(c *gin.Context) { c.String(200, "web ui") })
	return r
}

// setupTestRouterWithAuth 创建带认证的测试路由
func setupTestRouterWithAuth() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(recoveryMiddleware())
	r.Use(corsMiddleware("*"))
	r.Use(requestLogMiddleware())
	r.Use(rateLimitMiddleware(newIPRateLimiterWithBurst(600, 100), "/api/v1/health"))
	r.Use(basicAuthMiddleware("testuser", "testpass", map[string]bool{
		"/api/v1/health":    true,
		"/api/v1/platforms": true,
		"/":                 true,
	}))

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", healthHandler)
		v1.GET("/platforms", platformsHandler)
		v1.GET("/parse", v1ParseURLHandler)
		v1.GET("/parse/:source/:video_id", v1ParseIDHandler)
	}

	r.GET("/video/share/url/parse", legacyParseURLHandler)
	r.GET("/video/id/parse", legacyParseIDHandler)
	r.GET("/", func(c *gin.Context) { c.String(200, "web ui") })
	return r
}

func TestIntegrationHealthEndpoint(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("health 应返回 200，实际: %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("health status 应为 ok，实际: %v", resp["status"])
	}
}

func TestIntegrationPlatformsEndpoint(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/platforms", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("platforms 应返回 200，实际: %d", w.Code)
	}
}

func TestIntegrationV1ParseMissingURL(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/parse", nil)
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("缺少 url 应返回 400，实际: %d", w.Code)
	}
}

func TestIntegrationV1ParseInvalidURL(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/parse?url=just-text-no-url", nil)
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("无有效 URL 应返回 400，实际: %d", w.Code)
	}
}

func TestIntegrationV1ParseUnknownPlatform(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/parse?url=https://example.com/video/123", nil)
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("未知平台应返回 400，实际: %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	errObj := resp["error"].(map[string]any)
	if errObj["code"] != "UNSUPPORTED_URL" {
		t.Errorf("错误码应为 UNSUPPORTED_URL，实际: %v", errObj["code"])
	}
}

func TestIntegrationV1ParseIDUnknownSource(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/parse/unknown_platform/123", nil)
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("未知 source 应返回 400，实际: %d", w.Code)
	}
}

func TestIntegrationV1ParseIDNoIDParse(t *testing.T) {
	r := setupTestRouter()

	// redbook 没有 ID 解析器
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/parse/redbook/123", nil)
	r.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("不支持 ID 解析应返回 400，实际: %d", w.Code)
	}
}

func TestIntegrationV1ParseURLSuccess(t *testing.T) {
	stubParserFuncs(t, func(string) (*parser.VideoParseInfo, error) {
		info := &parser.VideoParseInfo{}
		info.Title = "测试视频"
		info.VideoUrl = "https://example.com/video.mp4"
		return info, nil
	}, nil)
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/parse?url=https://v.douyin.com/test/", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("v1 URL 解析成功应返回 200，实际: %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "success" {
		t.Errorf("status 应为 success，实际: %v", resp["status"])
	}
	data := resp["data"].(map[string]any)
	if data["title"] != "测试视频" {
		t.Errorf("data.title 应为 '测试视频'，实际: %v", data["title"])
	}
	// 验证 VideoParseInfo 所有字段始终存在
	for _, key := range []string{"author", "title", "video_url", "music_url", "cover_url", "images"} {
		if _, ok := data[key]; !ok {
			t.Errorf("data 缺少必需字段: %s", key)
		}
	}
	// 验证 author 嵌套字段
	author := data["author"].(map[string]any)
	for _, key := range []string{"uid", "name", "avatar"} {
		if _, ok := author[key]; !ok {
			t.Errorf("author 缺少必需字段: %s", key)
		}
	}
}

func TestIntegrationV1ParseIDSuccess(t *testing.T) {
	stubParserFuncs(t, nil, func(string, string) (*parser.VideoParseInfo, error) {
		info := &parser.VideoParseInfo{}
		info.Title = "ID解析视频"
		return info, nil
	})
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/parse/douyin/123", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("v1 ID 解析成功应返回 200，实际: %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "success" {
		t.Errorf("status 应为 success，实际: %v", resp["status"])
	}
	data := resp["data"].(map[string]any)
	if data["title"] != "ID解析视频" {
		t.Errorf("data.title 应为 'ID解析视频'，实际: %v", data["title"])
	}
	for _, key := range []string{"author", "title", "video_url", "music_url", "cover_url", "images"} {
		if _, ok := data[key]; !ok {
			t.Errorf("data 缺少必需字段: %s", key)
		}
	}
	// 验证 author 嵌套字段
	author := data["author"].(map[string]any)
	for _, key := range []string{"uid", "name", "avatar"} {
		if _, ok := author[key]; !ok {
			t.Errorf("author 缺少必需字段: %s", key)
		}
	}
}

func TestIntegrationV1ParseURL422OnParseFailure(t *testing.T) {
	stubParserFuncs(t, func(string) (*parser.VideoParseInfo, error) {
		return nil, fmt.Errorf("upstream parse failed")
	}, nil)
	r := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/parse?url=https://v.douyin.com/test/", nil)
	r.ServeHTTP(w, req)

	if w.Code != 422 {
		t.Errorf("解析失败应返回 422，实际: %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	errObj := resp["error"].(map[string]any)
	if errObj["code"] != "PARSE_FAILED" {
		t.Errorf("422 错误码应为 PARSE_FAILED，实际: %v", errObj["code"])
	}
}

func TestIntegrationV1ParseID422OnParseFailure(t *testing.T) {
	stubParserFuncs(t, nil, func(string, string) (*parser.VideoParseInfo, error) {
		return nil, fmt.Errorf("upstream parse failed")
	})
	r := setupTestRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/parse/douyin/123", nil)
	r.ServeHTTP(w, req)

	if w.Code != 422 {
		t.Errorf("解析失败应返回 422，实际: %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	errObj := resp["error"].(map[string]any)
	if errObj["code"] != "PARSE_FAILED" {
		t.Errorf("422 错误码应为 PARSE_FAILED，实际: %v", errObj["code"])
	}
}

func TestIntegrationLegacyRouteFormat(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/video/share/url/parse?url=invalid", nil)
	r.ServeHTTP(w, req)

	// 旧路由始终 200
	if w.Code != 200 {
		t.Errorf("旧路由应始终 200，实际: %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["code"]; !ok {
		t.Error("旧路由应包含 code 字段")
	}
	if _, ok := resp["msg"]; !ok {
		t.Error("旧路由应包含 msg 字段")
	}
}

func TestIntegrationCORSHeaders(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/api/v1/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != 204 {
		t.Errorf("OPTIONS 应返回 204，实际: %d", w.Code)
	}
}

func TestIntegrationNotFound(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/nonexistent", nil)
	r.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("不存在的路由应返回 404，实际: %d", w.Code)
	}
}

func TestIntegrationV1ParseIDMissingPathParam(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/parse/douyin", nil)
	r.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("缺少 video_id 的路由应返回 404，实际: %d", w.Code)
	}
}

func TestIntegrationLegacyIDParse(t *testing.T) {
	r := setupTestRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/video/id/parse?source=unknown&video_id=123", nil)
	r.ServeHTTP(w, req)

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

// === 认证覆盖测试 ===

func TestIntegrationAuthProtectsV1Parse(t *testing.T) {
	r := setupTestRouterWithAuth()

	// 无凭证访问受保护路由 → 401
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/parse?url=test", nil)
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Errorf("无凭证访问 /api/v1/parse 应返回 401，实际: %d", w.Code)
	}

	// 正确凭证 → 通过认证（可能 400 因为 URL 无效，但不应该是 401）
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/v1/parse?url=test", nil)
	req2.SetBasicAuth("testuser", "testpass")
	r.ServeHTTP(w2, req2)
	if w2.Code == 401 {
		t.Error("正确凭证不应返回 401")
	}
}

func TestIntegrationAuthExemptPlatforms(t *testing.T) {
	r := setupTestRouterWithAuth()

	// /api/v1/platforms 无需认证
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/platforms", nil)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("/platforms 应无需认证返回 200，实际: %d", w.Code)
	}
}

func TestIntegrationAuthExemptHealth(t *testing.T) {
	r := setupTestRouterWithAuth()

	// /api/v1/health 无需认证
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("/health 应无需认证返回 200，实际: %d", w.Code)
	}
}

func TestIntegrationAuthExemptWebUI(t *testing.T) {
	r := setupTestRouterWithAuth()

	// GET /（Web UI）无需认证
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("/ 应无需认证返回 200，实际: %d", w.Code)
	}
}

func TestIntegrationAuthProtectsLegacyRoutes(t *testing.T) {
	r := setupTestRouterWithAuth()

	// 旧路由 /video/share/url/parse 也受认证保护
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/video/share/url/parse?url=test", nil)
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Errorf("无凭证访问旧路由应返回 401，实际: %d", w.Code)
	}

	// 旧路由 /video/id/parse 也受认证保护
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/video/id/parse?source=test&video_id=123", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != 401 {
		t.Errorf("无凭证访问旧路由 /video/id/parse 应返回 401，实际: %d", w2.Code)
	}
}

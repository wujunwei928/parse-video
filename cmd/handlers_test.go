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

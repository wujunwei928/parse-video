package cmd

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSendSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"key": "value"}
	sendSuccess(c, data)

	if w.Code != 200 {
		t.Errorf("状态码应为 200，实际: %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"status":"success"`) {
		t.Errorf("响应应包含 status:success，实际: %s", body)
	}
	if !strings.Contains(body, `"key":"value"`) {
		t.Errorf("响应应包含 data 内容，实际: %s", body)
	}
}

func TestSendError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sendError(c, 400, ErrMissingParameter, "url 参数缺失")

	if w.Code != 400 {
		t.Errorf("状态码应为 400，实际: %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是有效 JSON: %v", err)
	}
	if resp["status"] != "error" {
		t.Errorf("status 应为 error，实际: %v", resp["status"])
	}
	errObj := resp["error"].(map[string]any)
	if errObj["code"] != "MISSING_PARAMETER" {
		t.Errorf("error.code 应为 MISSING_PARAMETER，实际: %v", errObj["code"])
	}
}

func TestErrorCodeConstants(t *testing.T) {
	codes := []string{
		ErrMissingParameter, ErrUnsupportedURL, ErrUnsupportedSource,
		ErrIDParseNotSupported, ErrParseFailed, ErrUnauthorized,
		ErrRateLimited, ErrInternal,
	}
	for _, code := range codes {
		if code == "" {
			t.Error("错误码不应为空")
		}
		if strings.Contains(code, " ") {
			t.Errorf("错误码不应包含空格: %s", code)
		}
	}
}

func TestSendErrorContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sendError(c, 422, ErrParseFailed, "解析失败")

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type 应为 application/json，实际: %s", ct)
	}
}

func TestClassifyParseError(t *testing.T) {
	// parser 返回的 error 统一归类为 PARSE_FAILED
	status, code := classifyParseError(fmt.Errorf("任意 parser 错误"))
	if status != 422 {
		t.Errorf("状态码应为 422，实际: %d", status)
	}
	if code != ErrParseFailed {
		t.Errorf("错误码应为 PARSE_FAILED，实际: %s", code)
	}
}

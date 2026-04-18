package cmd

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// v1 API 错误码
const (
	ErrMissingParameter    = "MISSING_PARAMETER"
	ErrUnsupportedURL      = "UNSUPPORTED_URL"
	ErrUnsupportedSource   = "UNSUPPORTED_SOURCE"
	ErrIDParseNotSupported = "ID_PARSE_NOT_SUPPORTED"
	ErrParseFailed         = "PARSE_FAILED"
	ErrUnauthorized        = "UNAUTHORIZED"
	ErrRateLimited         = "RATE_LIMITED"
	ErrInternal            = "INTERNAL_ERROR"
)

// apiResponse v1 成功响应
type apiResponse struct {
	Status string `json:"status"`
	Data   any    `json:"data"`
}

// apiErrorResponse v1 错误响应
type apiErrorResponse struct {
	Status string   `json:"status"`
	Error  apiError `json:"error"`
}

// apiError 错误详情
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// sendSuccess 发送 v1 成功响应
func sendSuccess(c *gin.Context, data any) {
	c.JSON(http.StatusOK, apiResponse{Status: "success", Data: data})
}

// sendError 发送 v1 错误响应
func sendError(c *gin.Context, httpStatus int, code string, message string) {
	c.JSON(httpStatus, apiErrorResponse{
		Status: "error",
		Error:  apiError{Code: code, Message: message},
	})
}

// classifyParseError 将 parser 返回的 error 统一分类为 PARSE_FAILED（422）
// parser 包返回 error 接口无类型化错误，一律归类为解析失败
func classifyParseError(err error) (int, string) {
	return 422, ErrParseFailed
}

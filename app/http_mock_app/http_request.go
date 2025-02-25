package http_mock_app

import (
	"encoding/json"
	"fmt"
	model "go_mock_server/internal/domain/model/mock_rule"
	"io"
	"net/http"
	"strings"
)

// HTTPRequestInfo 封装 HTTP 请求信息
type HTTPRequestInfo struct {
	Request *http.Request
}

var _ model.RequestInfo = (*HTTPRequestInfo)(nil) // 确保实现了 RequestInfo 接口

func NewHTTPRequestInfo(r *http.Request) *HTTPRequestInfo {
	return &HTTPRequestInfo{Request: r}
}

func (h *HTTPRequestInfo) GetProtocol() string {
	return "http"
}

func (h *HTTPRequestInfo) GetMethod() string {
	return h.Request.Method
}

func (h *HTTPRequestInfo) GetPath() string {
	return h.Request.URL.Path
}

func (h *HTTPRequestInfo) GetHeaders() map[string]string {
	headers := make(map[string]string)
	for key, values := range h.Request.Header {
		headers[key] = values[0] // 这里只取第一个值，根据实际需求可能需要调整
	}
	return headers
}

func (h *HTTPRequestInfo) GetBody() []byte {
	//  这里为了简化示例，直接读取 Body 为 string，实际应用中可能需要根据 Content-Type 进行更智能的处理
	buf := new(strings.Builder)
	_, _ = io.Copy(buf, h.Request.Body) // 忽略错误
	return []byte(buf.String())
}

func (h *HTTPRequestInfo) GetBodyJSON() (map[string]any, error) {
	bodyMap := make(map[string]any)
	decoder := json.NewDecoder(h.Request.Body)
	err := decoder.Decode(&bodyMap)
	if err != nil {
		return nil, fmt.Errorf("failed to decode HTTP body as JSON: %w", err)
	}
	return bodyMap, nil
}

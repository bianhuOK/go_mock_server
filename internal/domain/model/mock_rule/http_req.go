package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type HTTPRequestInfo struct {
	req       *http.Request
	bodyCache []byte
}

// 创建 HTTP RequestInfo 的工厂方法
func NewHTTPRequest(r *http.Request) RequestInfo {
	// 预读请求体并缓存
	body, _ := io.ReadAll(r.Body)
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	return &HTTPRequestInfo{
		req:       r,
		bodyCache: body,
	}
}

// 实现接口方法
func (h *HTTPRequestInfo) GetProtocol() string {
	if h.req.TLS != nil {
		return "https"
	}
	return "http"
}

func (h *HTTPRequestInfo) GetMethod() string {
	return h.req.Method
}

func (h *HTTPRequestInfo) GetPath() string {
	return h.req.URL.Path
}

func (h *HTTPRequestInfo) GetHeaders() map[string]string {
	headers := make(map[string]string)
	for k, v := range h.req.Header {
		headers[strings.ToLower(k)] = strings.Join(v, ",")
	}
	return headers
}

func (h *HTTPRequestInfo) GetBody() []byte {
	return h.bodyCache
}

func (h *HTTPRequestInfo) GetBodyJSON() (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal(h.bodyCache, &result); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}
	return result, nil
}

func (h *HTTPRequestInfo) GetMatchIndex() string {
	return BuildL1MatchIndexKeyFromReq(h)
}

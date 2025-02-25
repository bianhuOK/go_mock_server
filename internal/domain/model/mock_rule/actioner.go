package model

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"go_mock_server/utils"
	"html/template"
	"net/http"
	"time"
)

type ActionType string

const (
	ActionTypeResponse ActionType = "response" // 返回响应
	ActionTypeForward  ActionType = "forward"  // 转发请求 (未来扩展)
	ActionTypeError    ActionType = "error"    // 返回错误 (未来扩展)
)

type Protocol string

const (
	ProtocolHTTP      Protocol = "http"
	ProtocolHTTPS     Protocol = "https"
	ProtocolGRPC      Protocol = "grpc"
	ProtocolWebSocket Protocol = "websocket"
)

// Action 接口，定义所有 ActionConfig 需要实现的方法
type Action interface {
	Validate() error
	Execute(ctx context.Context, req RequestInfo) (ResponseInfo, error)
}

type ActionConfigWrapper struct {
	AType  ActionType `json:"type" gorm:"-"`
	Config Action     `json:"config" gorm:"-"`
	Raw    []byte     `gorm:"type:json" json:"-"` // 实际存储字段
}

// 实现 GORM 的 Scanner/Valuer 接口
func (w *ActionConfigWrapper) Scan(value interface{}) error {
	// 处理数据库读取
	bytes, _ := value.([]byte)
	return w.UnmarshalJSON(bytes)
}

func (w ActionConfigWrapper) Value() (driver.Value, error) {
	return w.MarshalJSON()
}

// 自定义 JSON 序列化
func (w *ActionConfigWrapper) MarshalJSON() ([]byte, error) {
	type Alias struct {
		Type   ActionType `json:"type"`
		Config any        `json:"config"`
	}
	return json.Marshal(&Alias{
		Type:   w.AType,
		Config: w.Config,
	})
}

// 自定义 JSON 反序列化
func (w *ActionConfigWrapper) UnmarshalJSON(data []byte) error {
	type Alias struct {
		Type   ActionType      `json:"type"`
		Config json.RawMessage `json:"config"`
	}

	var temp Alias
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	w.AType = temp.Type

	switch temp.Type {
	case ActionTypeResponse:
		var cfg ResponseAction
		if err := json.Unmarshal(temp.Config, &cfg); err != nil {
			return err
		}
		w.Config = &cfg
	// case ActionProxy:
	// 	var cfg ProxyConfig
	// 	if err := json.Unmarshal(temp.Config, &cfg); err != nil {
	// 		return err
	// 	}
	// 	w.Config = cfg
	default:
		return errors.New("unknown action type")
	}

	return nil
}

// GenericResponse represents a generic HTTP response structure
type GenericResponse struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte
}

type ResponseAction struct {
	StatusCode   int                    `json:"statusCode,omitempty"`
	Headers      map[string]string      `json:"headers,omitempty"`
	Body         string                 `json:"body,omitempty"`       // 统一使用字符串类型
	BodyBytes    []byte                 `json:"-" gorm:"-"`           // 运行时二进制数据
	BodyBase64   string                 `json:"bodyBase64,omitempty"` // 显式Base64编码字段
	Template     bool                   `json:"template,omitempty"`
	TemplateData map[string]interface{} `json:"templateData,omitempty"`
	Delay        time.Duration          `json:"delay,omitempty"`
}

// 自定义序列化/反序列化逻辑
func (r *ResponseAction) MarshalJSON() ([]byte, error) {
	type Alias ResponseAction
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	// 自动处理二进制数据
	if len(r.BodyBytes) > 0 {
		aux.BodyBase64 = base64.StdEncoding.EncodeToString(r.BodyBytes)
		aux.Body = "" // 清空文本内容
	}

	return json.Marshal(aux)
}

func (r *ResponseAction) UnmarshalJSON(data []byte) error {
	type Alias ResponseAction
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// 自动解码Base64
	if aux.BodyBase64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(aux.BodyBase64)
		if err != nil {
			return err
		}
		r.BodyBytes = decoded
	}

	return nil
}

// 模板渲染方法
func (r *ResponseAction) RenderTemplate(data map[string]interface{}) ([]byte, error) {
	tpl := template.Must(template.New("response").Parse(r.Body))

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, mergeData(r.TemplateData, data)); err != nil {
		return nil, err
	}

	// 优先返回二进制数据
	if len(r.BodyBytes) > 0 {
		return r.BodyBytes, nil
	}
	return buf.Bytes(), nil
}

// 合并模板数据
func mergeData(base, extra map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range extra {
		merged[k] = v
	}
	return merged
}

func (r *ResponseAction) Validate() error {
	if r.StatusCode < 100 || r.StatusCode > 599 {
		return errors.New("invalid status code")
	}
	return nil
}

func (r *ResponseAction) Execute(ctx context.Context, req RequestInfo) (ResponseInfo, error) {
	// 创建基础响应对象
	resp := &BaseResponse{
		status:  r.StatusCode,
		headers: r.Headers,
		delay:   r.Delay,
	}

	// 优先级1：二进制数据直接返回
	if len(r.BodyBytes) > 0 {
		resp.body = r.BodyBytes
		return resp, nil
	}

	// 优先级2：处理模板渲染
	if r.Template {
		// 合并请求数据和模板数据
		reqData, err := req.GetBodyJSON()
		if err != nil {
			return nil, fmt.Errorf("获取请求数据失败: %w", err)
		}
		mergedData := mergeMaps(r.TemplateData, reqData)
		utils.GetLogger().Infof("merge data %v", mergedData)

		// 渲染模板
		rendered, err := r.RenderTemplate(mergedData)
		if err != nil {
			return &BaseResponse{
				err: fmt.Errorf("模板渲染失败: %w", err),
			}, nil
		}
		resp.body = rendered
		return resp, nil
	}

	// 默认返回文本内容
	resp.body = []byte(r.Body)
	return resp, nil
}

// 实现 ResponseInfo 接口的具体类型
type BaseResponse struct {
	status  int
	headers map[string]string
	body    []byte
	delay   time.Duration
	err     error
}

// 实现 ResponseInfo 接口方法
func (r *BaseResponse) GetStatus() int {
	if r.status == 0 { // 默认状态码处理
		return http.StatusOK
	}
	return r.status
}

func (r *BaseResponse) GetHeaders() map[string]string {
	return r.headers
}

func (r *BaseResponse) GetBody() []byte {
	return r.body
}

func (r *BaseResponse) GetBodyJSON() (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal(r.body, &result); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}
	return result, nil
}

func (r *BaseResponse) GetDelay() time.Duration {
	return r.delay
}

func (r *BaseResponse) GetError() error {
	return r.err
}

func (r *BaseResponse) String() string {
	return fmt.Sprintf("Status: %d, Headers: %v, Body: %s, Delay: %v, Error: %v",
		r.GetStatus(),
		r.headers,
		string(r.body),
		r.delay,
		r.err)
}

// 合并map数据（支持嵌套map）
func mergeMaps(base, extra map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range extra {
		if existing, ok := merged[k]; ok {
			if existingMap, ok1 := existing.(map[string]interface{}); ok1 {
				if vMap, ok2 := v.(map[string]interface{}); ok2 {
					merged[k] = mergeMaps(existingMap, vMap)
					continue
				}
			}
		}
		merged[k] = v
	}
	return merged
}

// // todo RuleActionResult 定义规则动作执行结果
// type RuleActionResult struct {
// 	StatusCode int               // HTTP 状态码 (例如 200, 404, 500)
// 	Headers    map[string]string // HTTP 响应头
// 	Body       any               // HTTP 响应体 (可以使用 interface{} 以支持不同类型的数据，例如 string, []byte, struct 等)
// 	BodyType   string            // 响应体类型 (例如 "plain", "json", "bytes")， 用于 Handler 层正确设置 Content-Type
// 	Error      error             // 执行过程中发生的错误 (如果存在)
// }

type ForwardAction struct {
	ActionType ActionType
	ForwardURL string `json:"forwardURL"` // 转发的目标 URL
	// 可以添加更多转发相关的配置，例如 Header 修改，Path 修改等
}

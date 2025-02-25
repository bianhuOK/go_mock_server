package model

import "time"

type RequestInfo interface {
	GetProtocol() string           // 获取协议类型 (例如 "http", "grpc", "tcp")
	GetMethod() string             // 获取请求方法 (例如 HTTP Method, gRPC Method Name)
	GetPath() string               // 获取请求路径/资源标识符 (例如 HTTP Path, gRPC Service/Method)
	GetHeaders() map[string]string // 获取请求头 (例如 HTTP Headers, gRPC Metadata)
	GetBody() []byte
	GetBodyJSON() (map[string]any, error)
	GetMatchIndex() string // 获取匹配索引
	//  可以根据需要添加更多通用方法，例如获取查询参数、客户端地址等
}

type ResponseInfo interface {
	GetStatus() int                       // Get response status code (e.g., HTTP status code, gRPC status code)
	GetHeaders() map[string]string        // Get response headers
	GetBody() []byte                      // Get response body as raw bytes
	GetBodyJSON() (map[string]any, error) // Get response body as JSON
	GetDelay() time.Duration              // Get configured response delay
	GetError() error                      // Get any error associated with the response
}

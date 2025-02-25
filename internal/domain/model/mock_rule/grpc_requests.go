package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type GRPCRequestInfo struct {
	method   string        // 完整方法路径 如 /package.Service/Method
	metadata metadata.MD   // 元数据
	body     []byte        // 原始二进制数据
	message  proto.Message // 反序列化后的消息
}

// 创建 gRPC RequestInfo 的工厂方法
func NewGRPCRequest(
	ctx context.Context,
	fullMethod string,
	req proto.Message,
) RequestInfo {
	// 获取元数据
	md, _ := metadata.FromIncomingContext(ctx)

	// 序列化消息体
	var body []byte
	if req != nil {
		body, _ = proto.Marshal(req)
	}

	return &GRPCRequestInfo{
		method:   fullMethod,
		metadata: md,
		body:     body,
		message:  req,
	}
}

// 实现接口方法
func (g *GRPCRequestInfo) GetProtocol() string {
	return "grpc"
}

func (g *GRPCRequestInfo) GetMethod() string {
	parts := strings.Split(g.method, "/")
	if len(parts) >= 3 {
		return parts[2] // 返回方法名如 Method
	}
	return g.method
}

func (g *GRPCRequestInfo) GetPath() string {
	parts := strings.Split(g.method, "/")
	if len(parts) >= 2 {
		return parts[1] // 返回服务路径如 package.Service
	}
	return g.method
}

func (g *GRPCRequestInfo) GetHeaders() map[string]string {
	headers := make(map[string]string)
	for k, v := range g.metadata {
		headers[k] = strings.Join(v, ",")
	}
	return headers
}

func (g *GRPCRequestInfo) GetBody() []byte {
	return g.body
}

func (g *GRPCRequestInfo) GetBodyJSON() (map[string]any, error) {
	if g.message == nil {
		return nil, errors.New("空消息体")
	}

	// 使用 protojson 转换
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: true,
	}

	jsonBytes, err := marshaler.Marshal(g.message)
	if err != nil {
		return nil, fmt.Errorf("proto转JSON失败: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}
	return result, nil
}

func (g *GRPCRequestInfo) GetMatchIndex() string {
	return fmt.Sprintf("%s:%s", g.GetPath(), g.GetMethod())
}

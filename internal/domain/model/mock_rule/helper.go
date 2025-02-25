package model

import (
	"regexp"
	"strings"
)

// NormalizePath 将路径中的动态部分替换为 *
// 示例：
//
//	/api/user/spx123           => /api/user/*
//	^/api/user/\d+$            => /api/user/*
//	/api/order/{order_id}      => /api/order/*
//	/api/product/[a-zA-Z0-9]+  => /api/product/*
func NormalizePath(path string) string {
	// 处理正则表达式开头的 ^ 和结尾的 $
	path = strings.TrimPrefix(path, "^")
	path = strings.TrimSuffix(path, "$")

	// 替换 {xxx} 或 :xxx 为 *
	path = regexp.MustCompile(`\{[^}]+\}|:\w+`).ReplaceAllString(path, "*")

	// 替换连续的非斜杠字符段为 *
	// 匹配类似 /spx123 或 /\d+ 或 /[a-z]+ 等
	path = regexp.MustCompile(`/([^/]*(\\\d+|\\[a-zA-Z]+|\[.*\]|\+|\*|\?)[^/]*|)`).ReplaceAllString(path, "/*")

	// 合并连续的 *
	path = regexp.MustCompile(`/\*(\*)+`).ReplaceAllString(path, "/*")

	return path
}

func BuildL1MatchIndexKeyFromRule(rule *MockRule) string {
	return BuildL1MatchIndexKey(rule.Protocol, rule.Method, rule.PathPattern)
}

func BuildL1MatchIndexKeyFromReq(req RequestInfo) string {
	return BuildL1MatchIndexKey(req.GetProtocol(), req.GetMethod(), req.GetPath())
}

func BuildL1MatchIndexKey(schema string, method string, path string) string {
	if method == "" {
		method = "*"
	}
	normalizedPath := NormalizePath(path)
	return strings.Join([]string{schema, method, normalizedPath}, "_")
}

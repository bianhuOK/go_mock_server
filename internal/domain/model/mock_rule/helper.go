package model

import (
	"regexp"
	"strings"
)

// NormalizePath 将路径中的动态部分替换为 *
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

	// 替换路径中，正则表达式格式 为 *
	path = regexp.MustCompile(`(/[^/]*[+*?[\]{}\\][^/]*)`).ReplaceAllString(path, "/*")

	// 替换纯数字路径段为 *
	path = regexp.MustCompile(`(/[\d]+)`).ReplaceAllString(path, "/*")

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
	methodLower := strings.ToLower(method)
	return strings.Join([]string{schema, methodLower, normalizedPath}, "_")
}

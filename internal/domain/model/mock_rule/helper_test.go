package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple path",
			path:     "/api/users",
			expected: "/api/users",
		},
		{
			name:     "simple path",
			path:     "/api/users/123",
			expected: "/api/users/123",
		},
		{
			name:     "simple path2",
			path:     "*/api/users",
			expected: "*/api/users",
		},
		{
			name:     "path with numeric ID",
			path:     "/api/users/123",
			expected: "/api/users/*",
		},
		{
			name:     "path with regex pattern",
			path:     "^/api/users/\\d+$",
			expected: "/api/users/*",
		},
		{
			name:     "path with curly brace parameter",
			path:     "/api/users/{user_id}",
			expected: "/api/users/*",
		},
		{
			name:     "path with colon parameter",
			path:     "/api/users/:userId",
			expected: "/api/users/*",
		},
		{
			name:     "path with complex regex",
			path:     "/api/products/[a-zA-Z0-9]+",
			expected: "/api/products/*",
		},
		{
			name:     "path with multiple parameters",
			path:     "/api/users/{userId}/orders/{orderId}",
			expected: "/api/users/*/orders/*",
		},
		{
			name:     "path with mixed parameter styles",
			path:     "/api/users/:userId/orders/{orderId}",
			expected: "/api/users/*/orders/*",
		},
		{
			name:     "path with regex quantifiers",
			path:     "/api/items/\\w+/details",
			expected: "/api/items/*/details",
		},
		{
			name:     "path with consecutive dynamic parts",
			path:     "/api/v1/**/*",
			expected: "/api/v1/*/*",
		},
		{
			name:     "path with v1 prefix",
			path:     "^/api/v1/users",
			expected: "/api/v1/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePath(tt.path)
			assert.Equal(t, tt.expected, result, "paths should match")
		})
	}
}

func TestNormalizePathEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "empty path",
			path:     "",
			expected: "",
		},
		{
			name:     "root path",
			path:     "/",
			expected: "/",
		},
		{
			name:     "single asterisk",
			path:     "*",
			expected: "*",
		},
		{
			name:     "multiple consecutive asterisks",
			path:     "/**/**/***",
			expected: "/*",
		},
		{
			name:     "invalid regex pattern",
			path:     "/api/[invalid",
			expected: "/api/*",
		},
		{
			name:     "path with query parameters",
			path:     "/api/users?id=123",
			expected: "/api/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePath(tt.path)
			assert.Equal(t, tt.expected, result, "paths should match")
		})
	}
}

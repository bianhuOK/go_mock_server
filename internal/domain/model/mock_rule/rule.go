package model

import (
	"context"
	"fmt"
	"log"

	"gorm.io/gorm"
)

type MockRuleIface interface {
	IsMatch(ctx context.Context, requestInfo RequestInfo) bool
	ExecuteAction(ctx context.Context, requestInfo RequestInfo) (ResponseInfo, error)
}

var _ MockRuleIface = (*MockRule)(nil)

// Mock规则聚合根（核心领域对象）
type MockRule struct {
	ID           string              `gorm:"primaryKey;type:varchar(36)" json:"id" redis:"id"`
	Name         string              `gorm:"type:varchar(50)" json:"name" redis:"name" validate:"required"`                            // 规则名称
	Protocol     string              `gorm:"type:varchar(20);index:idx_protocol" json:"protocol" redis:"protocol" validate:"required"` // 协议类型标识
	MatchConfig  MatchConfig         `gorm:"type:json" json:"match" redis:"match" validate:"required"`                                 // 复合匹配条件
	ActionConfig ActionConfigWrapper `gorm:"type:json" json:"action" redis:"action" validate:"required"`                               // 响应配置
	Priority     int                 `gorm:"default:0" json:"priority" redis:"priority"`                                               // 匹配优先级
	Status       RuleStatus          `gorm:"type:varchar(20);index" json:"status" redis:"status"`                                      // 规则状态
	Version      int                 `gorm:"default:1" json:"version" redis:"version"`                                                 // 版本控制
	CreatedAt    int                 `gorm:"createdAt" json:"createdAt" redis:"created_at"`
	UpdatedAt    int                 `gorm:"updatedAt" json:"updatedAt" redis:"updated_at"`
	Method       string              `gorm:"type:varchar(20)" json:"method"`                       // 请求方法
	OriginalPath string              `gorm:"type:varchar(255)" json:"original_path"`               // 原始路径（用于展示）
	PathPattern  string              `gorm:"type:varchar(255)" json:"path_pattern"`                // 路径匹配模式（如 /api/user/:id）
	L1MatchIndex string              `gorm:"type:varchar(255);index:idx_l1" json:"l1_match_index"` // 标准化后的路径（如 /api/user/*）
	L2MatchIndex string              `gorm:"type:varchar(255);index:idx_l2" json:"l2_match_index"` // 更宽泛的匹配索引，其他高频条件（如 method+protocol）
}

// RuleFilter 定义规则查询的过滤器
type RuleFilter struct {
	RuleID          *string // 规则 ID 精确匹配
	Protocol        *string // 协议类型精确匹配
	CreatedByUserID *int    // 创建者用户 ID 精确匹配
	TagIDs          []int   // 标签 ID 列表 (规则需要包含任一标签 ID)
	IsEnabled       *bool   // 是否启用状态精确匹配
	PathContains    *string // Path 包含指定字符串 (模糊匹配)
	L1MatchIndex    *string // L1MatchIndex 精确匹配
	// ... 可以根据需求添加更多 Filter 字段 ...
}

func (m *MockRule) IsMatch(ctx context.Context, requestInfo RequestInfo) bool {
	// Check if the rule is enabled
	if m.Status != RuleStatusActive {
		return false
	}

	// Validate protocol match
	if m.Protocol != requestInfo.GetProtocol() {
		return false
	}

	// Delegate to MatchConfig for detailed matching logic
	return m.MatchConfig.Match(ctx, requestInfo)
}

func (m *MockRule) ExecuteAction(ctx context.Context, req RequestInfo) (ResponseInfo, error) {
	// 前置校验
	if m.Status != RuleStatusActive {
		return nil, fmt.Errorf("规则 [%s] 当前状态不可用 (状态: %s)", m.ID, m.Status)
	}

	// 参数校验
	if m.ActionConfig.Config == nil {
		return nil, fmt.Errorf("规则 [%s] 未配置 Action", m.ID)
	}

	// 记录执行日志
	log.Printf("执行规则: %s, 协议: %s, 优先级: %d", m.Name, m.Protocol, m.Priority)

	// 执行具体 Action
	// start := time.Now()
	resp, err := m.ActionConfig.Config.Execute(ctx, req)
	// duration := time.Since(start)

	// // 埋点监控
	// metrics.RecordExecution(m.Protocol, duration, err == nil)

	return resp, err
}

func (m *MockRule) BeforeSave(tx *gorm.DB) (err error) {
	m.L1MatchIndex = NormalizePath(m.OriginalPath)

	return nil
}

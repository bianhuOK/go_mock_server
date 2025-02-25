package iface

import (
	"context"
	model "go_mock_server/internal/domain/model/mock_rule"
)

// RuleService 规则服务接口
type RuleService interface {
	// CreateRule 创建规则
	CreateRule(context context.Context, rule *model.MockRule) error
}

type RuleMatchService interface {
	// MatchRule 匹配规则
	MatchRule(ctx context.Context, reqInfo model.RequestInfo) (*model.MockRule, error)
	// ExecuteRuleAction 执行规则动作
	ExecuteRuleAction(ctx context.Context, rule *model.MockRule) (model.ResponseInfo, error)
}

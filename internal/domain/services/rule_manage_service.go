package services

import (
	"context"
	"fmt"
	model "go_mock_server/internal/domain/model/mock_rule"
	"go_mock_server/internal/infra/repo"
)

type RuleManageService struct {
	ruleRepo repo.RuleRepositoryIface
}

func NewRuleManageService(ruleRepo repo.RuleRepositoryIface) *RuleManageService {
	return &RuleManageService{
		ruleRepo: ruleRepo,
	}
}

// CreateRule 创建规则
func (s *RuleManageService) CreateRule(ctx context.Context, rule *model.MockRule) error {
	if err := s.validateRule(rule); err != nil {
		return fmt.Errorf("rule validation failed: %w", err)
	}

	//  rule.ID = generateUniqueID() //  例如使用 UUID 生成

	if err := s.ruleRepo.SaveRule(ctx, rule); err != nil {
		return fmt.Errorf("failed to save rule to repository: %w", err)
	}

	return nil
}

func (s *RuleManageService) validateRule(rule *model.MockRule) error {
	if rule.Protocol == "" {
		return fmt.Errorf("missing 'protocol' field")
	}
	if len(rule.MatchConfig.Conditions) == 0 {
		return fmt.Errorf("matcher must have at least one condition")
	}

	//  更复杂的条件组合验证，例如 Operator 和 Value 是否匹配，Key 是否在特定 Type 下必填等等
	for _, condition := range rule.MatchConfig.Conditions {
		if condition.Type == "" {
			return fmt.Errorf("condition 'type' cannot be empty")
		}
		if condition.Operator == "" {
			return fmt.Errorf("condition 'operator' cannot be empty")
		}
		// ...  根据 condition.Type 和 condition.Operator  进行更细致的验证 ...
	}

	// Action 配置验证
	if rule.ActionConfig == nil {
		return fmt.Errorf("action configuration is missing")
	}

	return nil // 验证通过
}

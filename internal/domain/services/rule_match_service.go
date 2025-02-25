package services

import (
	"context"
	"fmt"
	model "go_mock_server/internal/domain/model/mock_rule"
	"go_mock_server/internal/infra/repo"
)

type RuleMatchService struct {
	ruleRepo repo.RuleRepositoryIface
}

func (s *RuleMatchService) MatchRule(ctx context.Context, reqInfo model.RequestInfo) (*model.MockRule, error) {
	bestMatchRule, err := s.ruleRepo.FindBestMatchRule(ctx, reqInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to find best match rule from repo: %w", err)
	}

	if bestMatchRule != nil {
		isMatch := bestMatchRule.MatchConfig.Match(ctx, reqInfo)
		if isMatch {
			return bestMatchRule, nil // 规则匹配成功
		}
	}

	return nil, nil //  未找到匹配规则
}

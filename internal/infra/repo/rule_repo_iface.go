package repo

import (
	"context"
	model "go_mock_server/internal/domain/model/mock_rule"
)

// RuleRepository 接口 - 定义数据仓库操作
type RuleRepositoryIface interface {
	SaveRule(ctx context.Context, rule *model.MockRule) error
	DeleteRule(ctx context.Context, ruleID string) error
	FindByID(ctx context.Context, ruleID string) (*model.MockRule, error)
	FindBestMatchRule(ctx context.Context, req model.RequestInfo) (*model.MockRule, error)
	ListRulesWithPage(ctx context.Context, filter *model.RuleFilter, page, pageSize int) ([]*model.MockRule, int64, error)
	// ListAll(ctx context.Context) ([]*MockRule, error)

	GetIndexRule(ctx context.Context, indexKey string) ([]*model.MockRule, error)
}

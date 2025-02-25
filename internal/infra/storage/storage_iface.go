package storage

import (
	"context"
	model "go_mock_server/internal/domain/model/mock_rule"
)

type MySQLRuleStorageIface interface {
	SaveRuleToDB(ctx context.Context, rule *model.MockRule) error
	GetRuleFromDB(ctx context.Context, ruleID string) (*model.MockRule, error)
	DeleteRuleFromDB(ctx context.Context, ruleID string) error
	BatchGetRules(ctx context.Context, ruleIDs []string) ([]*model.MockRule, error)

	// ListRules 增加了通用的规则列表查询方法，支持 Filter
	ListRules(ctx context.Context, filter *model.RuleFilter) ([]*model.MockRule, error)
	ListRulesWithPage(ctx context.Context, filter *model.RuleFilter, page, pageSize int) ([]*model.MockRule, int64, error)
	// ... 可以根据需求继续添加 ListRulesByXxx 方法 ...
}

// RedisRuleCacheInterface 定义 Redis 缓存操作接口
type RedisRuleCacheIface interface {
	GetRuleFromCache(ctx context.Context, ruleID string) (*model.MockRule, error)
	SetRuleToCache(ctx context.Context, rule *model.MockRule) error
	DeleteRuleFromCache(ctx context.Context, ruleID string) error
	// index

	RemoveFromIndex(ctx context.Context, rule *model.MockRule) error
	GetIndexCache(ctx context.Context, indexKey string) ([]string, error)
	SetIndexCache(ctx context.Context, indexKey string, ruleID string) error
	UpdateIndexCache(ctx context.Context, rule *model.MockRule) error
}

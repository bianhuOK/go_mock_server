package repo

import (
	"context"
	"fmt"
	model "go_mock_server/internal/domain/model/mock_rule"
	configs "go_mock_server/internal/infra/config"
	"go_mock_server/internal/infra/storage"
	"go_mock_server/utils"
	"strings"

	"github.com/avast/retry-go/v4"
	"github.com/go-redis/redis/v8"
	"github.com/panjf2000/ants/v2"
	"golang.org/x/sync/singleflight"
)

// ruleRepoImpl 实现了 RuleRepository 接口 (增加 singleflight 并发控制, retry-go, ants pool, config)
type ruleRepoImpl struct {
	mysqlStorage storage.MySQLRuleStorageIface
	redisCache   storage.RedisRuleCacheIface
	redisClient  *redis.Client
	config       *configs.RuleRepoConfig
	taskPool     *ants.Pool
	sfGroup      singleflight.Group
}

// 确保 ruleRepoImpl 实现了 RuleRepository 接口 (编译时检查)
var _ RuleRepositoryIface = (*ruleRepoImpl)(nil)

// indexUpdateRequest 定义异步索引更新请求的结构体
type indexUpdateRequest struct {
	ctx           context.Context
	rule          *model.MockRule
	operationType indexOperationType
}

type indexOperationType string

const (
	indexOperationTypeUpdate = "update"
	indexOperationTypeRemove = "remove"
)

func NewRuleRepoConfig(c *configs.RuleConfig) *configs.RuleRepoConfig {
	return &c.RuleRepoConfig
}

func NewRuleRepoImpl(mysqlStorage storage.MySQLRuleStorageIface, redisCache storage.RedisRuleCacheIface, redisClient *redis.Client, config *configs.RuleRepoConfig) RuleRepositoryIface {
	taskPool, err := ants.NewPool(config.IndexUpdatePoolSize)
	if err != nil {
		panic(fmt.Errorf("failed to create ants pool: %w", err)) //  ants pool 初始化失败，直接 panic
	}

	repo := &ruleRepoImpl{
		mysqlStorage: mysqlStorage,
		redisCache:   redisCache,
		redisClient:  redisClient,
		config:       config,
		taskPool:     taskPool,             //  使用 ants pool
		sfGroup:      singleflight.Group{}, // 初始化 singleflight Group
	}
	return repo
}

// ListRulesWithPage 列出规则并支持分页
func (r *ruleRepoImpl) ListRulesWithPage(ctx context.Context, filter *model.RuleFilter, page, pageSize int) ([]*model.MockRule, int64, error) {
	// 使用 singleflight 防止并发查询
	key := fmt.Sprintf("list_rules_%s", r.getListRulesFilterKey(filter, page, pageSize))
	data, err, _ := r.sfGroup.Do(key, func() (interface{}, error) {
		// 从数据库查询规则列表
		rules, total, err := r.mysqlStorage.ListRulesWithPage(ctx, filter, page, pageSize)
		if err != nil {
			return nil, fmt.Errorf("failed to list rules from db: %w", err)
		}

		return struct {
			rules []*model.MockRule
			total int64
		}{rules, total}, nil
	})

	if err != nil {
		return nil, 0, err
	}

	result := data.(struct {
		rules []*model.MockRule
		total int64
	})
	return result.rules, result.total, nil
}

// FindByID 根据ID查询规则
func (r *ruleRepoImpl) FindByID(ctx context.Context, id string) (*model.MockRule, error) {
	// 先从缓存查询
	rule, err := r.redisCache.GetRuleFromCache(ctx, id)
	if err == nil {
		utils.GetLogger().Debugf("rule found in cache: %s", id)
		return rule, nil
	}

	// 使用 singleflight 防止缓存击穿
	data, err, _ := r.sfGroup.Do(fmt.Sprintf("find_rule_by_id_%s", id), func() (interface{}, error) {
		// 从数据库查询
		rule, err := r.mysqlStorage.GetRuleFromDB(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get rule from db: %w", err)
		}

		// 设置缓存，使用重试机制
		err = retry.Do(
			func() error {
				return r.redisCache.SetRuleToCache(ctx, rule)
			},
			retry.Attempts(uint(r.config.RedisCacheRetryCount)),
			retry.Delay(r.config.RedisCacheRetryDelay),
		)
		if err != nil {
			return rule, fmt.Errorf("failed to set rule cache: %w", err)
		}

		return rule, nil
	})

	if err != nil {
		return nil, err
	}

	return data.(*model.MockRule), nil
}

func (r *ruleRepoImpl) GetIndexRule(ctx context.Context, indexKey string) ([]*model.MockRule, error) {
	// 2. 优先从 Redis sorted set 获取可能匹配的规则ID集合
	ruleIDs, err := r.redisCache.GetIndexCache(ctx, indexKey)
	if err != nil || len(ruleIDs) == 0 {
		// Redis 未命中，从数据库中查找 path like 的规则集合
		utils.GetLogger().Debugf("index cache miss for key: %s, now get data from db", indexKey)
		rules, err := r.mysqlStorage.ListRules(ctx, &model.RuleFilter{L1MatchIndex: &indexKey})
		if err != nil {
			return nil, fmt.Errorf("failed to get rules from db: %w", err)
		}

		// 将查询结果更新到 Redis
		if len(rules) > 0 {
			// Async update rule scores using task pool
			if err := r.taskPool.Submit(func() {
				err := retry.Do(
					func() error {
						for _, rule := range rules {
							if err := r.redisCache.UpdateIndexCache(ctx, rule); err != nil {
								return err
							}
							// 同时设置规则缓存
							if err := r.redisCache.SetRuleToCache(ctx, rule); err != nil {
								return err
							}
						}
						return nil
					},
					retry.Attempts(uint(r.config.RedisCacheRetryCount)),
					retry.Delay(r.config.RedisCacheRetryDelay),
				)
				if err != nil {
					fmt.Printf("failed to update rule scores and cache: %v\n", err)
				}
			}); err != nil {
				fmt.Printf("failed to submit update task: %v\n", err)
			}
		}
		return rules, nil
	}

	// Redis 中有索引数据,批量获取规则缓存
	rules := make([]*model.MockRule, 0, len(ruleIDs))
	missRuleIDs := make([]string, 0)

	// 尝试从缓存获取规则
	for _, ruleID := range ruleIDs {
		rule, err := r.redisCache.GetRuleFromCache(ctx, ruleID)
		if err != nil {
			missRuleIDs = append(missRuleIDs, ruleID)
			continue
		}
		rules = append(rules, rule)
	}

	// 对缓存未命中的规则批量查询DB
	if len(missRuleIDs) > 0 {
		dbRules, err := r.mysqlStorage.BatchGetRules(ctx, missRuleIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to batch get rules from db: %w", err)
		}
		rules = append(rules, dbRules...)

		// 异步更新缓存
		if err := r.taskPool.Submit(func() {
			err := retry.Do(
				func() error {
					for _, rule := range dbRules {
						if err := r.redisCache.SetRuleToCache(ctx, rule); err != nil {
							return err
						}
					}
					return nil
				},
				retry.Attempts(uint(r.config.RedisCacheRetryCount)),
				retry.Delay(r.config.RedisCacheRetryDelay),
			)
			if err != nil {
				fmt.Printf("failed to update rule cache: %v\n", err)
			}
		}); err != nil {
			fmt.Printf("failed to submit cache update task: %v\n", err)
		}
	}

	return rules, nil
}

// FindBestMatchRule 根据请求匹配最佳规则
func (r *ruleRepoImpl) FindBestMatchRule(ctx context.Context, req model.RequestInfo) (*model.MockRule, error) {
	// 1. 获取匹配索引
	matchIndex := req.GetMatchIndex()

	// 使用 singleflight 防止并发查询
	data, err, _ := r.sfGroup.Do(fmt.Sprintf("find_rule_%s", matchIndex), func() (interface{}, error) {
		utils.GetLogger().Debugf("finding best match rule for index: %s", matchIndex)
		var rules []*model.MockRule
		var err error

		// 2. 优先从 Redis sorted set 获取可能匹配的规则ID集合
		rules, err = r.GetIndexRule(ctx, matchIndex)
		if err != nil {
			utils.GetLogger().Warnf("failed to get index rule: %v", err)
			return nil, fmt.Errorf("failed to get index rule: %w", err)
		}

		// 3. 遍历规则集合进行精确匹配
		var bestMatch *model.MockRule
		for _, rule := range rules {
			if rule.IsMatch(ctx, req) {
				bestMatch = rule
				break
			}
		}

		if bestMatch == nil {
			return nil, fmt.Errorf("no matching rule found")
		}

		return bestMatch, nil
	})

	if err != nil {
		return nil, err
	}

	return data.(*model.MockRule), nil
}

// SaveRule 保存规则，同时更新缓存和索引
func (r *ruleRepoImpl) SaveRule(ctx context.Context, rule *model.MockRule) error {
	log := utils.GetLogger()
	_, err, _ := r.sfGroup.Do(fmt.Sprintf("save_rule_%s", rule.ID), func() (interface{}, error) {
		// 1. Save to DB (primary source of truth)
		err := retry.Do(
			func() error {
				return r.mysqlStorage.SaveRuleToDB(ctx, rule)
			},
			retry.Attempts(uint(r.config.SaveRuleDBRetryCount)),
			retry.Delay(r.config.SaveRuleDBRetryDelay),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to save rule to db: %w", err)
		}

		// 2. Async update cache and index
		r.taskPool.Submit(func() {
			// Update cache
			err := retry.Do(
				func() error {
					log.Infof("updating cache for rule %s", rule.ID)
					return r.redisCache.SetRuleToCache(ctx, rule)
				},
				retry.Attempts(uint(r.config.RedisCacheRetryCount)),
				retry.Delay(r.config.RedisCacheRetryDelay),
			)
			if err != nil {
				log.Printf("async cache update failed: %v", err)
			}

			// Update index
			r.handleIndexUpdate(&indexUpdateRequest{
				ctx:           ctx,
				rule:          rule,
				operationType: indexOperationTypeUpdate,
			})
		})

		return rule, nil
	})

	return err
}

// GetRule 获取规则，先查缓存，缓存未命中则查数据库
func (r *ruleRepoImpl) GetRule(ctx context.Context, ruleID string) (*model.MockRule, error) {
	// 先查缓存
	rule, err := r.redisCache.GetRuleFromCache(ctx, ruleID)
	if err == nil {
		return rule, nil
	}

	// 使用 singleflight 防止缓存击穿
	data, err, _ := r.sfGroup.Do(fmt.Sprintf("get_rule_%s", ruleID), func() (interface{}, error) {
		// 查询数据库
		rule, err := r.mysqlStorage.GetRuleFromDB(ctx, ruleID)
		if err != nil {
			return nil, err
		}

		// 设置缓存，使用重试机制
		err = retry.Do(
			func() error {
				return r.redisCache.SetRuleToCache(ctx, rule)
			},
			retry.Attempts(uint(r.config.RedisCacheRetryCount)),
			retry.Delay(r.config.RedisCacheRetryDelay),
		)
		if err != nil {
			return rule, fmt.Errorf("failed to set rule cache: %w", err)
		}

		return rule, nil
	})

	if err != nil {
		return nil, err
	}

	return data.(*model.MockRule), nil
}

// DeleteRule 删除规则，同时删除缓存和索引
func (r *ruleRepoImpl) DeleteRule(ctx context.Context, ruleID string) error {
	// 使用 singleflight 防止并发删除
	_, err, _ := r.sfGroup.Do(fmt.Sprintf("delete_rule_%s", ruleID), func() (interface{}, error) {
		// 获取规则信息（用于更新索引）
		rule, err := r.mysqlStorage.GetRuleFromDB(ctx, ruleID)
		if err != nil {
			return nil, fmt.Errorf("failed to get rule before delete: %w", err)
		}

		// 从数据库删除
		if err := r.mysqlStorage.DeleteRuleFromDB(ctx, ruleID); err != nil {
			return nil, fmt.Errorf("failed to delete rule from db: %w", err)
		}

		// 异步更新索引
		updateReq := &indexUpdateRequest{
			ctx:           ctx,
			rule:          rule,
			operationType: indexOperationTypeRemove,
		}
		if err := r.taskPool.Submit(func() {
			r.handleIndexUpdate(updateReq)
		}); err != nil {
			return nil, fmt.Errorf("failed to submit index update task: %w", err)
		}

		// 删除缓存，使用重试机制
		err = retry.Do(
			func() error {
				return r.redisCache.DeleteRuleFromCache(ctx, ruleID)
			},
			retry.Attempts(uint(r.config.RedisCacheRetryCount)),
			retry.Delay(r.config.RedisCacheRetryDelay),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to delete rule cache: %w", err)
		}

		return nil, nil
	})

	return err
}

// handleIndexUpdate 处理索引更新请求
func (r *ruleRepoImpl) handleIndexUpdate(req *indexUpdateRequest) {
	err := retry.Do(
		func() error {
			switch req.operationType {
			case indexOperationTypeUpdate:
				return r.redisCache.UpdateIndexCache(req.ctx, req.rule)
			case indexOperationTypeRemove:
				return r.redisCache.RemoveFromIndex(req.ctx, req.rule)
			default:
				return fmt.Errorf("unknown index operation type: %s", req.operationType)
			}
		},
		retry.Attempts(uint(r.config.IndexUpdateRetryCount)),
		retry.Delay(r.config.IndexUpdateRetryDelay),
	)
	if err != nil {
		utils.GetLogger().Errorf("failed to update index: %v\n", err)
	}
}

// getListRulesFilterKey generates a cache key for ListRules based on filter and pagination
func (r *ruleRepoImpl) getListRulesFilterKey(filter *model.RuleFilter, page, pageSize int) string {
	var parts []string

	if filter != nil {
		if filter.RuleID != nil {
			parts = append(parts, fmt.Sprintf("rid:%s", *filter.RuleID))
		}
		if filter.Protocol != nil {
			parts = append(parts, fmt.Sprintf("proto:%s", *filter.Protocol))
		}
		if filter.CreatedByUserID != nil {
			parts = append(parts, fmt.Sprintf("uid:%d", *filter.CreatedByUserID))
		}
		if len(filter.TagIDs) > 0 {
			parts = append(parts, fmt.Sprintf("tags:%v", filter.TagIDs))
		}
		if filter.IsEnabled != nil {
			parts = append(parts, fmt.Sprintf("enabled:%v", *filter.IsEnabled))
		}
		if filter.PathContains != nil {
			parts = append(parts, fmt.Sprintf("path:%s", *filter.PathContains))
		}
		if filter.L1MatchIndex != nil {
			parts = append(parts, fmt.Sprintf("l1:%s", *filter.L1MatchIndex))
		}
	}

	parts = append(parts, fmt.Sprintf("page:%d", page))
	parts = append(parts, fmt.Sprintf("size:%d", pageSize))

	if len(parts) == 0 {
		return "all"
	}
	return fmt.Sprintf("list_rules_%s", strings.Join(parts, "_"))
}

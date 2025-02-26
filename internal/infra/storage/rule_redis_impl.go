package storage

import (
	"context"
	"encoding/json"
	"fmt"
	model "go_mock_server/internal/domain/model/mock_rule"
	configs "go_mock_server/internal/infra/config"
	"go_mock_server/utils"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

const ruleKeyPrefix = "mock_rule:" // Redis Key 前缀

type redisRuleStorageImpl struct {
	redisClient *redis.Client
}

func NewRedisClient(c *configs.RuleConfig) *redis.Client {
	// 配置 Redis 连接参数
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", c.RedisConfig.Host, c.RedisConfig.Port), // Redis 服务器地址
		Password: c.RedisConfig.Password,                                       // Redis 密码，默认为空
		DB:       0,                                                            // Redis 数据库编号，默认为 0
	})

	// 测试连接是否成功
	ctx := context.Background()
	err := client.Ping(ctx).Err()
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis: %v", err))
	}

	fmt.Println("Successfully connected to Redis")
	return client
}

func NewredisRuleStorageImpl(redisClient *redis.Client) RedisRuleCacheIface {
	return &redisRuleStorageImpl{
		redisClient: redisClient,
	}
}

var _ RedisRuleCacheIface = (*redisRuleStorageImpl)(nil)

// GetIndexCache retrieves all rules from the index
func (r *redisRuleStorageImpl) GetIndexCache(ctx context.Context, indexKey string) ([]string, error) {
	utils.GetLogger().Debugf("Getting index members for key: %s", indexKey)
	// Use ZRange to get all members from the sorted set, ordered by priority
	members, err := r.redisClient.ZRevRange(ctx, indexKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get index members: %w", err)
	}

	var rules []string
	for _, ruleID := range members {
		rule, err := r.GetRuleFromCache(ctx, ruleID)
		if err != nil {
			continue
		}
		if rule != nil {
			rules = append(rules, rule.ID)
		}
	}
	return rules, nil
}

// SetIndexCache adds a rule ID to the index with priority as score
func (r *redisRuleStorageImpl) SetIndexCache(ctx context.Context, indexKey string, ruleID string) error {
	// Get the rule to access its priority
	rule, err := r.GetRuleFromCache(ctx, ruleID)
	if err != nil {
		return err
	}

	err = r.redisClient.ZAdd(ctx, indexKey, &redis.Z{
		Score:  float64(rule.Priority),
		Member: ruleID,
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to add rule to index: %w", err)
	}
	return nil
}

func (r *redisRuleStorageImpl) UpdateIndexCache(ctx context.Context, rule *model.MockRule) error {
	var err error
	// First ensure the rule exists in cache
	// err := r.SetRuleToCache(ctx, rule)
	// if err != nil {
	// 	return err
	// }

	// Update method-path index
	var indexKey string
	if rule.L1MatchIndex == "" {
		indexKey = model.BuildL1MatchIndexKeyFromRule(rule)
	} else {
		indexKey = rule.L1MatchIndex
	}
	err = r.SetIndexCache(ctx, indexKey, rule.ID)
	if err != nil {
		return err
	}
	return nil
}

// RemoveFromIndex removes a rule from the index
func (r *redisRuleStorageImpl) RemoveFromIndex(ctx context.Context, rule *model.MockRule) error {
	indexKey := model.BuildL1MatchIndexKeyFromRule(rule)
	err := r.redisClient.ZRem(ctx, indexKey, rule.ID).Err()
	if err != nil {
		return fmt.Errorf("failed to remove rule from index: %w", err)
	}
	return nil
}

// RuleRedisRepository Rule Redis 仓库
func (r *redisRuleStorageImpl) SetRuleToCache(ctx context.Context, rule *model.MockRule) error {
	if rule.ID == "" {
		rule.ID = generateUniqueID(rule)
	}

	ruleJSON, err := json.Marshal(rule) // 将 MockRule 序列化为 JSON
	if err != nil {
		return fmt.Errorf("failed to marshal rule to JSON: %w", err)
	}

	key := ruleKeyPrefix + rule.ID
	err = r.redisClient.Set(ctx, key, ruleJSON, 0).Err() //  存储到 Redis，Key 为 ruleKeyPrefix + RuleID
	if err != nil {
		return fmt.Errorf("failed to set rule to redis: %w", err)
	}
	return nil
}

// DeleteRuleFromCache deletes a rule from Redis cache by its ID
func (r *redisRuleStorageImpl) DeleteRuleFromCache(ctx context.Context, ruleID string) error {
	key := ruleKeyPrefix + ruleID
	err := r.redisClient.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete rule from redis: %w", err)
	}
	return nil
}

// FindByID  根据 RuleID 从 Redis 中查找 MockRule
func (r *redisRuleStorageImpl) GetRuleFromCache(ctx context.Context, ruleID string) (*model.MockRule, error) {
	key := ruleKeyPrefix + ruleID
	ruleJSON, err := r.redisClient.Get(ctx, key).Result()
	if err == redis.Nil { // Redis 中 Key 不存在
		utils.GetLogger().Info("Rule not found in cache, ", "ruleID", ruleID)
		return nil, fmt.Errorf("key %s not exist", key) //  Rule 不存在，返回 nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get rule from redis: %w", err)
	}

	rule := &model.MockRule{}
	err = json.Unmarshal([]byte(ruleJSON), rule) // 将 JSON 反序列化为 MockRule
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal rule from JSON: %w", err)
	}
	return rule, nil
}

func generateUniqueID(rule *model.MockRule) string {
	// Replace spaces with underscores and convert to lowercase for consistency
	sanitizedName := strings.ReplaceAll(strings.ToLower(rule.Name), " ", "_")
	return fmt.Sprintf("mock_%s_%d", sanitizedName, time.Now().UnixNano())
}

package storage

import (
	"context"
	"fmt"
	model "go_mock_server/internal/domain/model/mock_rule"
	configs "go_mock_server/internal/infra/config"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type MysqlRuleStorage struct {
	mysqlClient *gorm.DB
}

// new mysql client
func NewMySQLClient(c *configs.RuleConfig) *gorm.DB {
	dsn := c.DatabaseConfig.GetDSN()
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		panic("failed to connect database")
	}
	return db
}

func NewMysqlRuleStorage(mysqlClient *gorm.DB) MySQLRuleStorageIface {
	return &MysqlRuleStorage{mysqlClient: mysqlClient}
}

var _ MySQLRuleStorageIface = (*MysqlRuleStorage)(nil)

func (s *MysqlRuleStorage) SaveRuleToDB(ctx context.Context, rule *model.MockRule) error {
	tx := s.mysqlClient.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	if err := tx.Create(rule).Error; err != nil {
		tx.Rollback()
		if strings.Contains(err.Error(), "Error 1062: Duplicate entry") {
			return fmt.Errorf("rule with name '%s' and protocol '%s' already exists: %w", rule.Name, rule.Protocol, err)
		}
		return fmt.Errorf("failed to save rule to mysql: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *MysqlRuleStorage) GetRuleFromDB(ctx context.Context, ruleID string) (*model.MockRule, error) {
	rule := &model.MockRule{}
	if err := s.mysqlClient.WithContext(ctx).First(rule, "id = ?", ruleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // MySQL 中 Rule 不存在
		}
		return nil, fmt.Errorf("failed to get rule from mysql: %w", err)
	}
	return rule, nil
}

func (s *MysqlRuleStorage) DeleteRuleFromDB(ctx context.Context, ruleID string) error {
	if err := s.mysqlClient.WithContext(ctx).Delete(&model.MockRule{}, "id = ?", ruleID).Error; err != nil {
		return fmt.Errorf("failed to delete rule from mysql: %w", err)
	}
	return nil
}

// ListRules  通用的规则列表查询方法，支持 RuleFilter (不变)
func (s *MysqlRuleStorage) ListRules(ctx context.Context, filter *model.RuleFilter) ([]*model.MockRule, error) {
	var rules []*model.MockRule
	db := s.mysqlClient.WithContext(ctx).Model(&model.MockRule{}) //  使用 Model 创建 DB 查询构建器

	// 构建 WHERE 条件
	if filter != nil {
		if filter.RuleID != nil {
			db = db.Where("id = ?", *filter.RuleID)
		}
		if filter.Protocol != nil {
			db = db.Where("protocol = ?", *filter.Protocol)
		}
		if filter.CreatedByUserID != nil {
			db = db.Where("created_by_user_id = ?", *filter.CreatedByUserID)
		}
		if filter.IsEnabled != nil {
			db = db.Where("is_enabled = ?", *filter.IsEnabled)
		}
		if filter.PathContains != nil {
			db = db.Where("matcher_config LIKE ?", fmt.Sprintf("%%%s%%", *filter.PathContains)) // 模糊匹配 Path (JSON 字符串中)
		}
		if filter.L1MatchIndex != nil {
			db = db.Where("l1_match_index = ?", *filter.L1MatchIndex)
		}
		// ... 可以根据 RuleFilter 中的字段继续添加 WHERE 条件 ...
	}

	if err := db.Find(&rules).Error; err != nil { // 执行查询
		return nil, fmt.Errorf("failed to list rules from mysql with filter: %w", err)
	}
	return rules, nil
}

func (s *MysqlRuleStorage) ListRulesWithPage(ctx context.Context, filter *model.RuleFilter, page, pageSize int) ([]*model.MockRule, int64, error) {
	var rules []*model.MockRule
	var total int64
	db := s.mysqlClient.WithContext(ctx).Model(&model.MockRule{})

	// Apply filters
	if filter != nil {
		if filter.RuleID != nil {
			db = db.Where("id = ?", *filter.RuleID)
		}
		if filter.Protocol != nil {
			db = db.Where("protocol = ?", *filter.Protocol)
		}
		if filter.CreatedByUserID != nil {
			db = db.Where("created_by_user_id = ?", *filter.CreatedByUserID)
		}
		if filter.IsEnabled != nil {
			db = db.Where("is_enabled = ?", *filter.IsEnabled)
		}
		if filter.PathContains != nil {
			db = db.Where("matcher_config LIKE ?", fmt.Sprintf("%%%s%%", *filter.PathContains))
		}
		if filter.L1MatchIndex != nil {
			db = db.Where("l1_match_index = ?", *filter.L1MatchIndex)
		}
	}

	// Get total count
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count rules from mysql: %w", err)
	}

	// Apply pagination and get results
	if err := db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&rules).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list rules with pagination from mysql: %w", err)
	}

	return rules, total, nil
}

func (s *MysqlRuleStorage) BatchGetRules(ctx context.Context, ruleIDs []string) ([]*model.MockRule, error) {
	var rules []*model.MockRule
	if err := s.mysqlClient.WithContext(ctx).Where("id IN ?", ruleIDs).Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("failed to batch get rules from mysql: %w", err)
	}
	return rules, nil
}

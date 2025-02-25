// config/config.go
package configs

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 序列服务配置
type RuleConfig struct {
	DatabaseConfig       DatabaseConfig       `yaml:"database"`
	DatabaseOptionConfig DatabaseOptionConfig `yaml:"databaseConfig"`
	RedisConfig          RedisConfig          `yaml:"redis"`
	RuleRepoConfig       RuleRepoConfig       `yaml:"ruleRepo"`
}

// RuleRepoConfig 封装 ruleRepoImpl 的配置参数 (不变)
type RuleRepoConfig struct {
	RedisCacheRetryCount  int           `json:"redisCacheRetryCount" yaml:"redisCacheRetryCount"`
	RedisCacheRetryDelay  time.Duration `json:"redisCacheRetryDelay" yaml:"redisCacheRetryDelay"`
	SaveRuleDBRetryCount  int           `json:"saveRuleDBRetryCount" yaml:"saveRuleDBRetryCount"`
	SaveRuleDBRetryDelay  time.Duration `json:"saveRuleDBRetryDelay" yaml:"saveRuleDBRetryDelay"`
	IndexUpdateRetryCount int           `json:"indexUpdateRetryCount" yaml:"indexUpdateRetryCount"`
	IndexUpdateRetryDelay time.Duration `json:"indexUpdateRetryDelay" yaml:"indexUpdateRetryDelay"`
	IndexUpdatePoolSize   int           `json:"indexUpdatePoolSize" yaml:"indexUpdatePoolSize"`
}

// LoadConfig 加载配置
func LoadRuleConfig() (*RuleConfig, error) {
	// 1. 确定配置文件路径
	configPath := getConfigPath()

	// 2. 读取配置文件
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 3. 解析配置
	config := &RuleConfig{}
	if err := yaml.Unmarshal(configFile, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 4. 验证配置
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return config, nil
}

func LoadDbOptionConfig() (*DatabaseOptionConfig, error) {
	config, err := LoadRuleConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read seq config: %w", err)
	}
	return &config.DatabaseOptionConfig, nil
}

// getConfigPath 获取配置文件路径
func getConfigPath() string {
	// 优先使用环境变量
	os.Setenv("RULE_CONFIG_PATH", "/Users/bianniu/GolandProjects/go_mock_server/internal/infra/config/rule.local.yaml")
	if path := os.Getenv("RULE_CONFIG_PATH"); path != "" {
		return path
	}

	// 默认配置文件路径
	env := os.Getenv("RULE_ENV")
	if env == "" {
		env = "local"
	}

	return fmt.Sprintf("rule.%s.yaml", env)
}

// validate 验证配置
func (c *RuleConfig) validate() error {
	// 验证数据库配置
	db := c.DatabaseConfig
	if db.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if db.Port == 0 {
		return fmt.Errorf("database port is required")
	}
	if db.Username == "" {
		return fmt.Errorf("database username is required")
	}
	if db.Database == "" {
		return fmt.Errorf("database name is required")
	}

	// 验证数据库连接池配置
	dbConfig := c.DatabaseOptionConfig
	if dbConfig.MaxIdleConns <= 0 {
		return fmt.Errorf("maxIdleConns must be positive")
	}
	if dbConfig.MaxOpenConns <= 0 {
		return fmt.Errorf("maxOpenConns must be positive")
	}
	if dbConfig.MaxOpenConns < dbConfig.MaxIdleConns {
		return fmt.Errorf("maxOpenConns must be greater than or equal to maxIdleConns")
	}

	return nil
}

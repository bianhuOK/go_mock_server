package configs

import (
	"fmt"
	"time"
)

// DatabaseConfig 数据库基础配置
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

// DatabaseOptionConfig 数据库连接池配置
type DatabaseOptionConfig struct {
	MaxIdleConns    int           `yaml:"maxIdleConns"`
	MaxOpenConns    int           `yaml:"maxOpenConns"`
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime"`
	ConnMaxIdleTime time.Duration `yaml:"connMaxIdleTime"`
	LogLevel        string        `yaml:"logLevel"`
	SlowThreshold   time.Duration `yaml:"slowThreshold"`
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)
}

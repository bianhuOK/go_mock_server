package configs

import "time"

type RedisConfig struct {
	Host         string        `json:"host" yaml:"host"`
	Port         int           `json:"port" yaml:"port"`
	Password     string        `json:"password" yaml:"password"`
	Database     int           `json:"database" yaml:"db"`
	PoolSize     int           `json:"poolSize" yaml:"poolSize"`
	MinIdleConns int           `json:"minIdleConns" yaml:"minIdleConns"`
	MaxRetries   int           `json:"maxRetries" yaml:"maxRetries"`
	DialTimeout  time.Duration `json:"dialTimeout" yaml:"dialTimeout"`
	ReadTimeout  time.Duration `json:"readTimeout" yaml:"readTimeout"`
	WriteTimeout time.Duration `json:"writeTimeout" yaml:"writeTimeout"`
	PoolTimeout  time.Duration `json:"poolTimeout" yaml:"poolTimeout"`
	IdleTimeout  time.Duration `json:"idleTimeout" yaml:"idleTimeout"`
}

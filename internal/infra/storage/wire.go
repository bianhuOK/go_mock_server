package storage

import (
	configs "go_mock_server/internal/infra/config"

	"github.com/google/wire"
)

// StorageSet is a Wire provider set that includes all storage-related providers
var StorageSet = wire.NewSet(
	configs.LoadRuleConfig,
	NewMySQLClient, // Add MySQL client provider
	NewMysqlRuleStorage,
	NewRedisClient,
	NewredisRuleStorageImpl,
)

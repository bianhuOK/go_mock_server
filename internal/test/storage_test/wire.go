//go:build wireinject
// +build wireinject

package storagetest

import (
	"go_mock_server/internal/infra/storage"

	"github.com/google/wire"
)

type RuleStorageTestSuite struct {
	storage      storage.MySQLRuleStorageIface
	redisStorage storage.RedisRuleCacheIface
}

func NewRuleStorageTestSuite(st storage.MySQLRuleStorageIface, rd storage.RedisRuleCacheIface) *RuleStorageTestSuite {
	return &RuleStorageTestSuite{storage: st, redisStorage: rd}
}

func InitializeStorageTest() (*RuleStorageTestSuite, error) {
	wire.Build(storage.StorageSet, NewRuleStorageTestSuite)
	return &RuleStorageTestSuite{}, nil
}

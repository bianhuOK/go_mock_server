package repo

import (
	"go_mock_server/internal/infra/storage"

	"github.com/google/wire"
)

var Reposet = wire.NewSet(
	NewRuleRepoConfig,
	storage.StorageSet,
	NewRuleRepoImpl,
)

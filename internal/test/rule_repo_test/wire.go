//go:build wireinject
// +build wireinject

package rulerepotest

import (
	"go_mock_server/internal/infra/repo"

	"github.com/google/wire"
)

type RepoRuleTestSuite struct {
	Repo repo.RuleRepositoryIface
}

func NewRepoRuleTestSuite(r repo.RuleRepositoryIface) *RepoRuleTestSuite {
	return &RepoRuleTestSuite{Repo: r}
}

func InitializeRepoTest() (*RepoRuleTestSuite, error) {
	wire.Build(repo.Reposet, NewRepoRuleTestSuite)
	return &RepoRuleTestSuite{}, nil
}

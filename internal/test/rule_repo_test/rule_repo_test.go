package rulerepotest

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	model "go_mock_server/internal/domain/model/mock_rule"

	"github.com/stretchr/testify/assert"
)

func TestSaveRule(t *testing.T) {

	matcher := model.MatchConfig{
		Logical: "AND",
		Conditions: []model.MatchCondition{
			{
				Type:     "method",
				Operator: "eq",
				Value:    "POST",
			},
			{
				Type:     "path",
				Operator: "regex",
				Value:    "^/api/v1/users",
			},
		},
	}

	actionconfig := model.ActionConfigWrapper{
		AType: "response",
		Config: &model.ResponseAction{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Template: true,
			Body:     `{"message": "success", "order": "{{.order}}", "path": "{{.Request.URL.Path}}", "headers": {{.Request.Headers}}}`,
		},
	}

	timestamp := time.Now().Unix()
	rule := &model.MockRule{
		ID:           fmt.Sprintf("test-rule-%d", timestamp),
		Name:         fmt.Sprintf("Test Rule %d", timestamp),
		Protocol:     "http",
		MatchConfig:  matcher,
		ActionConfig: actionconfig,
		Status:       model.RuleStatusActive,
		Priority:     1,
		CreatedAt:    int(timestamp),
		UpdatedAt:    int(timestamp),
	}

	tests := []struct {
		name    string
		rule    *model.MockRule
		wantErr bool
	}{
		{
			name:    "success case",
			rule:    rule,
			wantErr: false,
		},
	}

	ts, err := InitializeRepoTest()
	assert.Nil(t, err)
	repo := ts.Repo

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = repo.SaveRule(ctx, tt.rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveRule() error = %v, wantErr %v", err, tt.wantErr)
			}
			// Read the saved rule and verify key fields
			savedRule, err := repo.FindByID(ctx, tt.rule.ID)
			if err != nil {
				t.Errorf("FindByID() error = %v", err)
				return
			}
			t.Logf("Saved Rule: %+v", savedRule)

			// Verify key fields
			assert.Equal(t, tt.rule.ID, savedRule.ID)
			assert.Equal(t, tt.rule.Name, savedRule.Name)
			assert.Equal(t, tt.rule.Protocol, savedRule.Protocol)
			assert.True(t, reflect.DeepEqual(tt.rule.MatchConfig, savedRule.MatchConfig))
			assert.True(t, reflect.DeepEqual(tt.rule.ActionConfig, savedRule.ActionConfig))
			assert.Equal(t, tt.rule.Status, savedRule.Status)
			assert.Equal(t, tt.rule.Priority, savedRule.Priority)
		})
	}

}

package rulerepotest

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	model "go_mock_server/internal/domain/model/mock_rule"

	"github.com/stretchr/testify/assert"
)

func TestSaveRule(t *testing.T) {
	testTimestamp := fmt.Sprintf("%d", time.Now().Unix())
	lastP := testTimestamp
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
				Value:    fmt.Sprintf("^/api/v1/users/%s", lastP),
			},
			{
				Type:     "header",
				Operator: "eq",
				Key:      "timestamp",
				Value:    testTimestamp,
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

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = tt.rule.Validate()
			assert.Nil(t, err)
			err = repo.SaveRule(ctx, tt.rule)
			assert.Nil(t, err)
			// Wait for Redis to complete the write operation
			time.Sleep(100 * time.Millisecond)
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

			// Test FindBestRule with a matching request
			requestBody := strings.NewReader(`{"name": "test user", "age": 25, "email": "test@example.com"}`)
			httpReq, _ := http.NewRequest("POST",
				fmt.Sprintf("http://localhost/api/v1/users/%s", lastP), requestBody)
			httpReq.Header.Set("timestamp", testTimestamp)
			requestInfo := model.NewHTTPRequest(httpReq)

			bestRule, err := repo.FindBestMatchRule(ctx, requestInfo)
			assert.Nil(t, err)
			assert.NotNil(t, bestRule)
			assert.Equal(t, tt.rule.ID, bestRule.ID)

			resp, err := bestRule.ExecuteAction(ctx, requestInfo)
			assert.Nil(t, err)
			assert.NotNil(t, resp)
			t.Logf("Response: %s", resp.GetBody())
		})
	}

}

package storagetest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	model "go_mock_server/internal/domain/model/mock_rule"

	"github.com/stretchr/testify/assert"
)

func TestSaveRule(t *testing.T) {
	ctx := context.Background()
	store, err := InitializeStorageTest()
	assert.NoError(t, err)

	// Create rule from http_mock.json
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

	// Save rule
	err = store.storage.SaveRuleToDB(ctx, rule)
	assert.NoError(t, err)

	// Get saved rule
	savedRule, err := store.storage.GetRuleFromDB(ctx, rule.ID)
	assert.NoError(t, err)
	assert.Equal(t, rule.ID, savedRule.ID)
	assert.Equal(t, rule.Protocol, savedRule.Protocol)
	assert.Equal(t, rule.Priority, savedRule.Priority)
	assert.Equal(t, rule.Status, savedRule.Status)

	// Write to http_mock_back.json
	output, err := json.MarshalIndent(savedRule, "", "  ")
	assert.NoError(t, err)
	err = os.WriteFile("http_mock_back.json", output, 0644)
	assert.NoError(t, err)
}

func TestRedisSaveRule(t *testing.T) {
	ctx := context.Background()
	store, err := InitializeStorageTest()
	assert.NoError(t, err)

	// Create rule from http_mock.json
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

	// Save rule
	err = store.redisStorage.SetRuleToCache(ctx, rule)
	assert.NoError(t, err)

	// Get saved rule
	savedRule, err := store.redisStorage.GetRuleFromCache(ctx, rule.ID)
	assert.NoError(t, err)
	assert.Equal(t, rule.ID, savedRule.ID)
	assert.Equal(t, rule.Protocol, savedRule.Protocol)
	assert.Equal(t, rule.Priority, savedRule.Priority)
	assert.Equal(t, rule.Status, savedRule.Status)

	// Write to http_mock_back.json
	output, err := json.MarshalIndent(savedRule, "", "  ")
	assert.NoError(t, err)
	err = os.WriteFile("http_mock_back2.json", output, 0644)
	assert.NoError(t, err)
}

func TestBatchGetRules(t *testing.T) {
	ctx := context.Background()
	store, err := InitializeStorageTest()
	assert.NoError(t, err)

	// Create multiple test rules
	timestamp := time.Now().Unix()
	rules := make([]*model.MockRule, 3)
	for i := 0; i < 3; i++ {
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
					Value:    fmt.Sprintf("^/api/v1/test%d", i),
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
				Body:     `{"message": "success"}`,
			},
		}

		rules[i] = &model.MockRule{
			ID:           fmt.Sprintf("test-rule-%d-%d", timestamp, i),
			Name:         fmt.Sprintf("Test Rule %d-%d", timestamp, i),
			Protocol:     "http",
			MatchConfig:  matcher,
			ActionConfig: actionconfig,
			Status:       model.RuleStatusActive,
			Priority:     i + 1,
			CreatedAt:    int(timestamp),
			UpdatedAt:    int(timestamp),
		}

		// Save rules to both storage and cache
		err = store.storage.SaveRuleToDB(ctx, rules[i])
		assert.NoError(t, err)
		err = store.redisStorage.SetRuleToCache(ctx, rules[i])
		assert.NoError(t, err)
	}

	// Test batch get from storage
	ruleIDs := make([]string, len(rules))
	for i, rule := range rules {
		ruleIDs[i] = rule.ID
	}

	// Get rules from storage
	storedRules, err := store.storage.BatchGetRules(ctx, ruleIDs)
	assert.NoError(t, err)
	assert.Equal(t, len(rules), len(storedRules))

	// Verify stored rules
	for i, rule := range rules {
		assert.Equal(t, rule.ID, storedRules[i].ID)
		assert.Equal(t, rule.Protocol, storedRules[i].Protocol)
		assert.Equal(t, rule.Priority, storedRules[i].Priority)
		assert.Equal(t, rule.Status, storedRules[i].Status)
	}

	// Test batch get from Redis
	cachedRules := make([]*model.MockRule, len(ruleIDs))
	for i, id := range ruleIDs {
		rule, err := store.redisStorage.GetRuleFromCache(ctx, id)
		assert.NoError(t, err)
		cachedRules[i] = rule
	}

	// Verify cached rules
	for i, rule := range rules {
		assert.Equal(t, rule.ID, cachedRules[i].ID)
		assert.Equal(t, rule.Protocol, cachedRules[i].Protocol)
		assert.Equal(t, rule.Priority, cachedRules[i].Priority)
		assert.Equal(t, rule.Status, cachedRules[i].Status)
	}
}

func TestGetLatestRuleAndExecute(t *testing.T) {
	ctx := context.Background()
	store, err := InitializeStorageTest()
	assert.NoError(t, err)

	// Get latest rule
	rules, err := store.storage.ListRules(ctx, &model.RuleFilter{})
	assert.NoError(t, err)
	assert.NotEmpty(t, rules)

	// Find the most recent rule
	var latestRule *model.MockRule
	latestTime := 0
	for _, rule := range rules {
		if rule.CreatedAt > latestTime {
			latestTime = rule.CreatedAt
			latestRule = rule
		}
	}
	assert.NotNil(t, latestRule)

	// Create request info for testing
	req := http.Request{
		Method: "POST",
		URL: &url.URL{
			Path: "/api/v1/users",
		},
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(`{"test": "data", "order": "123312332"}`)),
	}
	reqInfo := model.NewHTTPRequest(&req)
	// Execute rule action
	resp, err := latestRule.ExecuteAction(ctx, reqInfo)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	fmt.Print(resp)
}

func TestRedisGetLatestRuleAndExecute(t *testing.T) {
	ctx := context.Background()
	store, err := InitializeStorageTest()
	assert.NoError(t, err)

	// Get latest rule
	latestRule, err := store.redisStorage.GetRuleFromCache(ctx, "test-rule-1740318464")
	assert.NoError(t, err)
	assert.NotNil(t, latestRule)

	// Create request info for testing
	req := http.Request{
		Method: "POST",
		URL: &url.URL{
			Path: "/api/v1/users",
		},
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(`{"test": "data", "order": "123312332"}`)),
	}
	reqInfo := model.NewHTTPRequest(&req)
	// Execute rule action
	resp, err := latestRule.ExecuteAction(ctx, reqInfo)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	fmt.Print(resp)
}

package http_mock_app

import (
	"encoding/json"
	"fmt"
	model "go_mock_server/internal/domain/model/mock_rule"

	"github.com/go-playground/validator/v10"
)

type CreateMockRuleRequest struct {
	Name     string         `json:"name" validate:"required,min=1,max=50"`
	Protocol string         `json:"protocol" validate:"required,oneof=http grpc"`
	Match    MatchConfigDTO `json:"match" validate:"required"`
	Action   ActionDTO      `json:"action" validate:"required"`
	Priority int            `json:"priority" validate:"min=0"`
}

// Validate performs validation on CreateMockRuleRequest
func (req *CreateMockRuleRequest) Validate() error {
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	// Validate action config based on action type
	switch req.Action.Type {
	case "response":
		var responseAction ResponseActionDTO
		if err := json.Unmarshal(req.Action.Config, &responseAction); err != nil {
			return fmt.Errorf("invalid response action config: %w", err)
		}
		if err := validate.Struct(responseAction); err != nil {
			return fmt.Errorf("invalid response action config: %w", err)
		}
	case "forward":
		var forwardAction ForwardActionDTO
		if err := json.Unmarshal(req.Action.Config, &forwardAction); err != nil {
			return fmt.Errorf("invalid forward action config: %w", err)
		}
		if err := validate.Struct(forwardAction); err != nil {
			return fmt.Errorf("invalid forward action config: %w", err)
		}
	}

	return nil
}

type MatchConfigDTO struct {
	Logical    string              `json:"logical" validate:"required,oneof=AND OR"`
	Conditions []MatchConditionDTO `json:"conditions" validate:"required,dive"`
}

type MatchConditionDTO struct {
	Type     string         `json:"type" validate:"required,oneof=method path header body_json"`
	Operator string         `json:"operator" validate:"required,oneof=eq regex exists json_path"`
	Key      any            `json:"key,omitempty"`
	Value    any            `json:"value"`
	Config   map[string]any `json:"config,omitempty"`
}

type ActionDTO struct {
	Type   string          `json:"type" validate:"required,oneof=response forward error"`
	Config json.RawMessage `json:"config" validate:"required"`
}

type ResponseActionDTO struct {
	StatusCode   int               `json:"statusCode" validate:"required,min=100,max=599"`
	Headers      map[string]string `json:"headers,omitempty"`
	Body         string            `json:"body,omitempty"`
	BodyBase64   string            `json:"bodyBase64,omitempty"`
	Template     bool              `json:"template,omitempty"`
	TemplateData map[string]any    `json:"templateData,omitempty"`
}

type ForwardActionDTO struct {
	ForwardURL string `json:"forwardURL" validate:"required,url"`
}

// ConvertToMockRule converts CreateMockRuleRequest DTO to MockRule model
func (dto *CreateMockRuleRequest) ConvertToMockRule() (*model.MockRule, error) {
	// Convert MatchConfig
	matchConfig := model.MatchConfig{
		Logical:    dto.Match.Logical,
		Conditions: make([]model.MatchCondition, len(dto.Match.Conditions)),
	}

	for i, c := range dto.Match.Conditions {
		matchConfig.Conditions[i] = model.MatchCondition{
			Type:     c.Type,
			Operator: c.Operator,
			Key:      c.Key,
			Value:    c.Value,
			Config:   c.Config,
		}
	}

	// Convert ActionConfig
	actionConfig, err := json.Marshal(dto.Action)
	if dto.Action.Type == "response" {
		var responseAction ResponseActionDTO
		if err := json.Unmarshal(dto.Action.Config, &responseAction); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response action config: %w", err)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to marshal action config: %w", err)
	}

	return &model.MockRule{
		Name:         dto.Name,
		Protocol:     dto.Protocol,
		MatchConfig:  matchConfig,
		ActionConfig: actionConfig,
		Priority:     dto.Priority,
		Status:       model.RuleStatusActive, // Set default status
		Version:      1,                      // Set initial version
	}, nil
}

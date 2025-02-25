package http_mock_app

import (
	"runtime/debug"

	"go_mock_server/internal/domain/iface"
	"go_mock_server/utils"

	"github.com/go-chassis/go-chassis/v2/pkg/metrics"
	rf "github.com/go-chassis/go-chassis/v2/server/restful"
)

type MockController struct {
	MockService       iface.RuleMatchService
	RuleManageService iface.RuleService
}

func NewMockController(mockService iface.RuleMatchService, ruleManageService iface.RuleService) *MockController {
	return &MockController{
		MockService:       mockService,
		RuleManageService: ruleManageService,
	}
}

func (c *MockController) CreateMockRule(b *rf.Context) {
	logger := utils.GetLogger()
	logger.Info("CreateMockRule Begin")

	// Record request metrics
	metrics.CounterAdd("request_counter", 1, map[string]string{
		"method":   b.ReadRequest().Method,
		"endpoint": b.ReadRequest().URL.Path,
	})

	defer func() {
		if err := recover(); err != nil {
			logger.WithFields(map[string]interface{}{
				"panic": err,
				"stack": string(debug.Stack()),
			}).Error("handle request panic")
			b.WriteJSON(struct {
				Error string `json:"error"`
			}{Error: "Internal server error"}, "application/json")
		}
	}()

	var req CreateMockRuleRequest
	if err := b.ReadEntity(&req); err != nil {
		logger.Errorf("read request body err: %v", err)
		b.WriteJSON(struct {
			Error string `json:"error"`
		}{Error: err.Error()}, "application/json")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		logger.Errorf("validate request err: %v", err)
		b.WriteJSON(struct {
			Error string `json:"error"`
		}{Error: err.Error()}, "application/json")
		return
	}

	// Convert request to model
	mockRule, err := req.ConvertToMockRule()
	if err != nil {
		logger.Errorf("convert request to model err: %v", err)
		b.WriteJSON(struct {
			Error string `json:"error"`
		}{Error: err.Error()}, "application/json")
		return
	}

	// Create rule using RuleManageService
	err = c.RuleManageService.CreateRule(b.Ctx, mockRule)
	if err != nil {
		logger.Errorf("create mock rule err: %v", err)
		b.WriteJSON(struct {
			Error string `json:"error"`
		}{Error: err.Error()}, "application/json")
		return
	}

	b.WriteJSON(struct {
		Message string `json:"message"`
	}{Message: "success"}, "application/json")
}

func (c *MockController) URLPatterns() []rf.Route {
	return []rf.Route{
		{Method: "POST", Path: "/mock/create_rule", ResourceFunc: c.CreateMockRule,
			Returns: []*rf.Returns{{Code: 200}}},
	}
}

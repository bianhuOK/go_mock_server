package model

// Mock规则聚合根（核心领域对象）
type MockRule struct {
	ID          string          `gorm:"primaryKey;type:varchar(36)" json:"id" redis:"id"`
	Protocol    string          `gorm:"type:varchar(20);index:idx_protocol" json:"protocol" redis:"protocol"` // 协议类型标识
	MatchConfig MatchConfig     `gorm:"embedded;embeddedPrefix:match_" json:"match" redis:"match"`          // 复合匹配条件
	Response    ResponseConfig  `gorm:"embedded;embeddedPrefix:response_" json:"response" redis:"response"` // 响应配置
	Status      RuleStatus      `gorm:"type:varchar(20);index" json:"status" redis:"status"`                // 规则状态
	Version     int             `gorm:"default:1" json:"version" redis:"version"`                           // 版本控制
	CreatedAt   time.Time       `gorm:"autoCreateTime" json:"createdAt" redis:"created_at"`
}

// 匹配条件配置（值对象）
type MatchConfig struct {
	Path        string          `gorm:"type:varchar(255);not null" json:"path" redis:"path"`                 // 路径模板
	Method      string          `gorm:"type:varchar(10)" json:"method" redis:"method"`                       // HTTP方法
	Priority    int             `gorm:"default:0" json:"priority" redis:"priority"`                          // 匹配优先级
	Conditions  []MatchCondition `gorm:"serializer:json" json:"conditions" redis:"-"`                         // 动态匹配条件
	RateLimit   int             `gorm:"default:0" json:"rateLimit" redis:"rate_limit"`                       // 限流阈值
}

// 匹配条件接口（支持扩展）
type MatchCondition interface {
	Type() string
	Match(*http.Request) bool
}

// 示例实现：Header匹配条件
type HeaderCondition struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Regex  bool   `json:"regex"`
}

func (h HeaderCondition) Type() string { return "header" }
func (h HeaderCondition) Match(r *http.Request) bool {
    // 具体匹配逻辑实现
}

// 响应配置（值对象）
type ResponseConfig struct {
	StatusCode  int               `gorm:"default:200" json:"statusCode" redis:"status_code"`
	ContentType string            `gorm:"type:varchar(50)" json:"contentType" redis:"content_type"`
	Body        string            `gorm:"type:text" json:"body" redis:"body"`                // 模板内容
	Variables   map[string]string `gorm:"serializer:json" json:"variables" redis:"variables"` // 模板变量
	Headers     map[string]string `gorm:"serializer:json" json:"headers" redis:"headers"`     // 响应头
}
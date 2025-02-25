package model

// RuleStatus represents the current state of a mock rule
type RuleStatus string

const (
	RuleStatusActive   RuleStatus = "active"
	RuleStatusInactive RuleStatus = "inactive"
	RuleStatusDraft    RuleStatus = "draft"
	RuleStatusArchived RuleStatus = "archived"
)

func (s RuleStatus) IsValid() bool {
	switch s {
	case RuleStatusActive, RuleStatusInactive, RuleStatusDraft, RuleStatusArchived:
		return true
	default:
		return false
	}
}

func (s RuleStatus) String() string {
	return string(s)
}

// 匹配类型枚举
const (
	MatchPath       = "path"
	MatchMethod     = "method"
	MatchHeader     = "header"
	MatchQueryParam = "query_param"
	MatchBodyJSON   = "json"
	MatchBodyRaw    = "body_raw"
)

// 操作符枚举
const (
	OpEqual    = "eq"
	OpRegex    = "regex"
	OpExists   = "exists"
	OpNotEmpty = "not_empty"
	OpJsonPath = "json_path"
	OpContains = "contains"
)

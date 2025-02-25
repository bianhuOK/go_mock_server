package model

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// 匹配条件配置（值对象）
type MatchConfig struct {
	Logical    string           `json:"logical" redis:"logical"` // AND/OR逻辑
	Conditions []MatchCondition `json:"conditions" redis:"conditions"`
}

// MatchCondition 定义单个匹配条件
type MatchCondition struct {
	Type     string         `json:"type" redis:"type"`               // 匹配类型 (method, path, header, body_json)
	Operator string         `json:"operator" redis:"operator"`       // 操作符 (eq, regex, exists, json_path)
	Key      any            `json:"key,omitempty" redis:"key"`       // 键 (header 或 body_json 时使用)
	Value    any            `json:"value" redis:"value"`             // 值
	Config   map[string]any `json:"config,omitempty" redis:"config"` // 扩展配置 (未来扩展使用)
}

// 2. 为 MatchConfig 实现 GORM 的 Scanner/Valuer 接口
func (mc *MatchConfig) Scan(value interface{}) error {
	// 处理数据库读取时的反序列化
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("类型转换失败")
	}
	return json.Unmarshal(bytes, mc)
}

func (mc MatchConfig) Value() (driver.Value, error) {
	// 处理数据库写入时的序列化
	return json.Marshal(mc)
}

// 3. 添加自定义类型校验（可选）
func (mc *MatchConfig) Validate() error {
	if mc.Logical == "" {
		return errors.New("path 不能为空")
	}
	return nil
}

func (m *MatchConfig) Match(ctx context.Context, reqInfo RequestInfo) bool { //  参数类型改为 RequestInfo
	if len(m.Conditions) == 0 {
		return false
	}

	isAnd := strings.ToUpper(m.Logical) == "AND"
	for _, cond := range m.Conditions {
		matched := m.matchCondition(ctx, reqInfo, cond) // 传递 RequestInfo
		if isAnd && !matched {
			return false
		}
		if !isAnd && matched {
			return true
		}
	}
	return isAnd
}

// matchCondition  匹配单个条件 (修改 - 接受 RequestInfo 接口)
func (m *MatchConfig) matchCondition(ctx context.Context, reqInfo RequestInfo, cond MatchCondition) bool { //  参数类型改为 RequestInfo
	switch strings.ToLower(cond.Type) {
	case "method":
		return m.matchMethod(reqInfo, cond)
	case "path":
		return m.matchPath(reqInfo, cond)
	case "header":
		return m.matchHeader(reqInfo, cond)
	case "body_json":
		return m.matchBodyJSON(ctx, reqInfo, cond)
	default:
		fmt.Printf("Warning: Unknown match type: %s\n", cond.Type)
		return false // Unknown match type, default to not match
	}
}

func (m *MatchConfig) matchMethod(reqInfo RequestInfo, cond MatchCondition) bool {
	reqMethod := reqInfo.GetMethod()
	ruleMethod, ok := cond.Value.(string)
	if !ok {
		fmt.Printf("Warning: Invalid rule method value type, expect string, got: %T\n", cond.Value)
		return false
	}
	return strings.ToUpper(reqMethod) == strings.ToUpper(ruleMethod)
}

func (m *MatchConfig) matchPath(reqInfo RequestInfo, cond MatchCondition) bool {
	reqPath := reqInfo.GetPath()
	rulePath, ok := cond.Value.(string)
	if !ok {
		fmt.Printf("Warning: Invalid rule path value type, expect string, got: %T\n", cond.Value)
		return false
	}

	operator := strings.ToLower(cond.Operator)
	switch operator {
	case "eq":
		return reqPath == rulePath
	case "regex":
		matched, _ := regexp.MatchString(rulePath, reqPath) // 忽略错误，正则表达式错误在规则加载时已验证
		return matched
	default: // 默认 Exact 匹配
		return reqPath == rulePath
	}
}

func (m *MatchConfig) matchHeader(reqInfo RequestInfo, cond MatchCondition) bool {
	reqHeaders := reqInfo.GetHeaders()
	ruleHeaderKey, ok := cond.Key.(string)
	if !ok {
		fmt.Printf("Warning: Invalid rule header key type, expect string, got: %T\n", cond.Key)
		return false
	}
	ruleHeaderValue, ok := cond.Value.(string)
	if !ok {
		fmt.Printf("Warning: Invalid rule header value type, expect string, got: %T\n", cond.Value)
		return false
	}

	reqHeaderValue, ok := reqHeaders[ruleHeaderKey]
	if !ok {
		return false // 请求头不存在
	}

	operator := strings.ToLower(cond.Operator)
	switch operator {
	case "eq":
		return reqHeaderValue == ruleHeaderValue
	case "regex":
		matched, _ := regexp.MatchString(ruleHeaderValue, ruleHeaderValue) // 忽略错误，正则表达式错误在规则加载时已验证
		return matched
	case "exists":
		return ok // Header 存在即匹配 (忽略 Value)
	default: // 默认 Exact 匹配
		return reqHeaderValue == ruleHeaderValue
	}
}

func (m *MatchConfig) matchBodyJSON(ctx context.Context, reqInfo RequestInfo, cond MatchCondition) bool {
	reqBodyJSON, err := reqInfo.GetBodyJSON()
	if err != nil {
		fmt.Printf("Warning: Failed to get request body as JSON: %v\n", err)
		return false // 无法解析 JSON，不匹配
	}
	if reqBodyJSON == nil {
		return false //  Body JSON 为空，不匹配
	}

	jsonPath, ok := cond.Key.(string) //  JSONPath 表达式作为 Key
	if !ok {
		fmt.Printf("Warning: Invalid rule body_json key type, expect string (JSONPath), got: %T\n", cond.Key)
		return false
	}
	ruleValue := cond.Value //  规则值 (any 类型)

	operator := strings.ToLower(cond.Operator)
	switch operator {
	case "json_path": // 使用 jsonpath 库进行 JSONPath 匹配
		res, err := JsonPathLookup(reqBodyJSON, jsonPath) //  假设 JsonPathLookup 函数已实现 (需要引入 jsonpath 库)
		if err != nil {
			fmt.Printf("Warning: JSONPath lookup error: %v, JSONPath: %s\n", err, jsonPath)
			return false // JSONPath 查询错误，不匹配
		}

		//  **  类型转换和比较 **: res 的类型可能是 interface{}, 需要根据 ruleValue 的类型进行转换和比较
		//  **  这里仅简单示例，假设 ruleValue 是 string 类型，进行字符串比较 **
		ruleValueStr, ruleValueOK := ruleValue.(string)
		resStr, resOK := res.(string)

		if ruleValueOK && resOK { //  Both are string, compare as string
			return resStr == ruleValueStr
		} else { // 类型不匹配，或者无法转换为字符串，不匹配
			fmt.Printf("Warning: JSONPath result type mismatch, expect string, got: %T, rule value type: %T\n", res, ruleValue)
			return false
		}

	default: // 默认 JSONPath 匹配
		res, err := JsonPathLookup(reqBodyJSON, jsonPath) //  默认也使用 jsonpath
		if err != nil {
			fmt.Printf("Warning: JSONPath lookup error: %v, JSONPath: %s\n", err, jsonPath)
			return false
		}

		//  ** 类型转换和比较 (同上) **
		ruleValueStr, ruleValueOK := ruleValue.(string)
		resStr, resOK := res.(string)

		if ruleValueOK && resOK { //  Both are string, compare as string
			return resStr == ruleValueStr
		} else { // 类型不匹配，或者无法转换为字符串，不匹配
			fmt.Printf("Warning: JSONPath result type mismatch, expect string, got: %T, rule value type: %T\n", res, ruleValue)
			return false
		}
	}
}

// ** 假设的 JsonPathLookup 函数 (需要引入第三方 jsonpath 库并实现) **
func JsonPathLookup(jsonData map[string]any, jsonPath string) (interface{}, error) {
	//  **  这里需要使用第三方 jsonpath 库，例如 "github.com/jsonpath-community/jsonpath" **
	//  **  示例代码仅为演示，需要根据实际使用的 jsonpath 库 API 进行调整 **
	//  **  以下代码仅为伪代码，需要替换为真实的 jsonpath 查询实现 **
	/*
	   res, err := jsonpath.JsonPathLookup(jsonData, jsonPath) //  假设的 jsonpath 查询函数
	   if err != nil {
	       return nil, fmt.Errorf("jsonpath lookup failed: %w", err)
	   }
	   return res, nil
	*/
	return nil, fmt.Errorf("JsonPathLookup not implemented, please install jsonpath library and implement it") //  TODO: Implement JsonPathLookup
}

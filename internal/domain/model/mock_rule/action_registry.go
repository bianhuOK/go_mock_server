package model

var configRegistry = make(map[ActionType]func() Action)

func RegisterConfig(t ActionType, factory func() Action) {
	configRegistry[t] = factory
}

// 初始化时注册
func init() {
	RegisterConfig(ActionTypeResponse, func() Action { return &ResponseAction{} })
}

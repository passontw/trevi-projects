package types

// IDGenerator 定義了生成唯一 ID 的接口
type IDGenerator interface {
	// NextID 生成並返回一個新的唯一 ID
	NextID() (int64, error)
}

// IDParser 定義了解析 ID 的接口
type IDParser interface {
	// ParseID 解析 ID 並返回其組成部分
	ParseID(id int64) map[string]interface{}
}

// IDValidator 定義了驗證 ID 的接口
type IDValidator interface {
	// ValidateID 驗證 ID 是否有效
	ValidateID(id int64) bool
}

// IDManager 組合 ID 生成、解析和驗證的接口
type IDManager interface {
	IDGenerator
	IDParser
	IDValidator
}

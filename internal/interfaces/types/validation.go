package types

// ValidationRule 定義了驗證規則
type ValidationRule struct {
	// Tag 是驗證標籤，例如 "required", "email", "min=1", "max=10" 等
	Tag string

	// Message 是自定義錯誤訊息
	Message string
}

// ValidationError 表示驗證錯誤
type ValidationError struct {
	// Field 是發生錯誤的欄位名稱
	Field string

	// Tag 是違反的驗證標籤
	Tag string

	// Value 是欄位的值
	Value interface{}

	// Message 是錯誤訊息
	Message string
}

// Validator 定義了請求驗證器的接口
type Validator interface {
	// Validate 驗證給定的結構體
	// 返回錯誤列表，若無錯誤則返回 nil
	Validate(obj interface{}) []ValidationError

	// ValidateField 驗證單個欄位
	// 返回錯誤列表，若無錯誤則返回 nil
	ValidateField(val interface{}, tag string) []ValidationError

	// RegisterValidation 註冊自定義驗證函數
	RegisterValidation(tag string, fn interface{}) error

	// RegisterCustomTypeFunc 註冊自定義類型處理函數
	RegisterCustomTypeFunc(fn interface{}, types ...interface{}) error

	// RegisterTagNameFunc 註冊標籤名稱函數
	RegisterTagNameFunc(fn func(field string) string)

	// RegisterCustomRule 註冊自定義驗證規則
	RegisterCustomRule(field string, rule ValidationRule) error
}

// ValidationResult 表示驗證結果
type ValidationResult struct {
	// Valid 表示驗證是否通過
	Valid bool

	// Errors 包含所有驗證錯誤
	Errors []ValidationError
}

// RequestValidator 定義了請求驗證器的接口
type RequestValidator interface {
	// ValidateRequest 驗證請求
	ValidateRequest(request interface{}) ValidationResult
}

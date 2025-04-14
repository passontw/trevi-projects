package utils

import (
	"reflect"
	"strings"
	"sync"

	"g38_lottery_servic/internal/interfaces/types"

	"github.com/go-playground/validator/v10"
)

// CustomValidator 是一個基於 go-playground/validator 的客製化驗證器
type CustomValidator struct {
	validator   *validator.Validate
	lock        sync.RWMutex
	customRules map[string]map[string]types.ValidationRule // fieldName -> tagName -> rule
}

var (
	validatorInstance *CustomValidator
	validatorOnce     sync.Once
)

// GetValidator 返回全局驗證器實例
func GetValidator() *CustomValidator {
	validatorOnce.Do(func() {
		v := validator.New()

		// 使用 JSON 標籤作為欄位名稱
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return fld.Name
			}
			return name
		})

		validatorInstance = &CustomValidator{
			validator:   v,
			customRules: make(map[string]map[string]types.ValidationRule),
		}
	})
	return validatorInstance
}

// Validate 驗證給定的結構體
func (v *CustomValidator) Validate(obj interface{}) []types.ValidationError {
	if err := v.validator.Struct(obj); err != nil {
		return v.translateErrors(err)
	}
	return nil
}

// ValidateField 驗證單個欄位
func (v *CustomValidator) ValidateField(val interface{}, tag string) []types.ValidationError {
	if err := v.validator.Var(val, tag); err != nil {
		return v.translateErrors(err)
	}
	return nil
}

// RegisterValidation 註冊自定義驗證函數
func (v *CustomValidator) RegisterValidation(tag string, fn interface{}) error {
	return v.validator.RegisterValidation(tag, fn.(validator.Func))
}

// RegisterCustomTypeFunc 註冊自定義類型處理函數
func (v *CustomValidator) RegisterCustomTypeFunc(fn interface{}, types ...interface{}) error {
	// 轉換為 CustomTypeFunc 類型
	customFunc := fn.(validator.CustomTypeFunc)
	v.validator.RegisterCustomTypeFunc(customFunc, types...)
	return nil
}

// RegisterTagNameFunc 註冊標籤名稱函數
func (v *CustomValidator) RegisterTagNameFunc(fn func(field string) string) {
	v.validator.RegisterTagNameFunc(func(fld reflect.StructField) string {
		return fn(fld.Name)
	})
}

// RegisterCustomRule 註冊自定義驗證規則
func (v *CustomValidator) RegisterCustomRule(field string, rule types.ValidationRule) error {
	v.lock.Lock()
	defer v.lock.Unlock()

	if _, exists := v.customRules[field]; !exists {
		v.customRules[field] = make(map[string]types.ValidationRule)
	}
	v.customRules[field][rule.Tag] = rule
	return nil
}

// translateErrors 將驗證錯誤轉換為客製化格式
func (v *CustomValidator) translateErrors(err error) []types.ValidationError {
	if err == nil {
		return nil
	}

	var result []types.ValidationError

	validationErrors := err.(validator.ValidationErrors)
	for _, e := range validationErrors {
		field := e.Field()
		tag := e.Tag()

		// 檢查是否有自定義錯誤訊息
		var message string
		v.lock.RLock()
		if fieldRules, exists := v.customRules[field]; exists {
			if rule, exists := fieldRules[tag]; exists {
				message = rule.Message
			}
		}
		v.lock.RUnlock()

		// 如果沒有自定義訊息，使用默認錯誤訊息
		if message == "" {
			message = getDefaultErrorMessage(field, tag, e.Param())
		}

		result = append(result, types.ValidationError{
			Field:   field,
			Tag:     tag,
			Value:   e.Value(),
			Message: message,
		})
	}

	return result
}

// getDefaultErrorMessage 獲取默認錯誤訊息
func getDefaultErrorMessage(field, tag, param string) string {
	switch tag {
	case "required":
		return field + "為必填欄位"
	case "email":
		return field + "必須是有效的電子郵件地址"
	case "min":
		return field + "長度必須至少為" + param + "個字符"
	case "max":
		return field + "長度不能超過" + param + "個字符"
	case "strong_password":
		return "密碼必須至少包含8個字符，包括大寫字母、小寫字母和數字"
	case "tw_phone":
		return "手機號碼必須是有效的台灣手機號碼(09開頭的10位數字)"
	case "valid_amount":
		return "金額必須大於0且不超過999,999,999.99"
	case "valid_username":
		return "用戶名必須是3-20個字符，只能包含字母、數字、下劃線和連字符"
	default:
		return field + "格式不正確"
	}
}

// ValidateRequest 實現 RequestValidator 接口
func (v *CustomValidator) ValidateRequest(request interface{}) types.ValidationResult {
	errors := v.Validate(request)
	return types.ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// NewCustomValidator 創建一個新的客製化驗證器實例
func NewCustomValidator() *CustomValidator {
	return GetValidator()
}

// 確保 CustomValidator 實現了 types.Validator 和 types.RequestValidator 接口
var (
	_ types.Validator        = (*CustomValidator)(nil)
	_ types.RequestValidator = (*CustomValidator)(nil)
)

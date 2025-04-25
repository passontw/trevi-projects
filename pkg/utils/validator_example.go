package utils

import (
	"fmt"

	"g38_lottery_service/internal/interfaces/types"
)

// 示例請求結構體
type ExampleRequest struct {
	Username string  `json:"username" validate:"required,valid_username"`
	Email    string  `json:"email" validate:"required,email"`
	Password string  `json:"password" validate:"required,strong_password"`
	Phone    string  `json:"phone" validate:"required,tw_phone"`
	Amount   float64 `json:"amount" validate:"required,valid_amount"`
	Age      int     `json:"age" validate:"required,min=18,max=120"`
	Nickname string  `json:"nickname" validate:"required,min=2,max=30"`
}

// ValidateExampleRequest 驗證示例請求
func ValidateExampleRequest(req *ExampleRequest) types.ValidationResult {
	validator := GetValidator()

	// 註冊自定義錯誤訊息
	_ = validator.RegisterCustomRule("Username", types.ValidationRule{
		Tag:     "valid_username",
		Message: "用戶名必須是3-20個字符，只能包含字母、數字、下劃線和連字符",
	})

	_ = validator.RegisterCustomRule("Password", types.ValidationRule{
		Tag:     "strong_password",
		Message: "密碼必須至少包含8個字符，包括大寫字母、小寫字母和數字",
	})

	_ = validator.RegisterCustomRule("Phone", types.ValidationRule{
		Tag:     "tw_phone",
		Message: "手機號碼必須是有效的台灣手機號碼(09開頭的10位數字)",
	})

	_ = validator.RegisterCustomRule("Amount", types.ValidationRule{
		Tag:     "valid_amount",
		Message: "金額必須大於0且不超過999,999,999.99",
	})

	// 驗證請求
	return validator.ValidateRequest(req)
}

// ExampleUsage 展示如何使用驗證器
func ExampleUsage() {
	// 確保自定義驗證函數已註冊
	RegisterCustomValidations()

	// 創建一個有錯誤的請求
	req := &ExampleRequest{
		Username: "u$er",          // 包含無效字符
		Email:    "invalid-email", // 無效的電子郵件
		Password: "weak",          // 弱密碼
		Phone:    "12345678",      // 無效的手機號碼
		Amount:   -100,            // 負數金額
		Age:      15,              // 未滿18歲
		Nickname: "A",             // 太短
	}

	// 驗證請求
	result := ValidateExampleRequest(req)

	// 處理驗證結果
	if !result.Valid {
		fmt.Println("驗證失敗，錯誤如下：")
		for _, err := range result.Errors {
			fmt.Printf("欄位: %s, 錯誤: %s\n", err.Field, err.Message)
		}
	} else {
		fmt.Println("驗證成功！")
	}
}

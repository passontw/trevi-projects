package utils

import (
	"regexp"
	"unicode"

	"github.com/go-playground/validator/v10"
)

// RegisterCustomValidations 註冊所有自定義驗證函數到驗證器
func RegisterCustomValidations() {
	v := GetValidator()

	// 註冊密碼驗證
	_ = v.RegisterValidation("strong_password", ValidateStrongPassword)

	// 註冊電話號碼驗證
	_ = v.RegisterValidation("tw_phone", ValidateTWPhone)

	// 註冊金額驗證
	_ = v.RegisterValidation("valid_amount", ValidateValidAmount)

	// 註冊用戶名驗證
	_ = v.RegisterValidation("valid_username", ValidateValidUsername)
}

// ValidateStrongPassword 驗證強密碼規則
func ValidateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	var (
		hasMinLen = len(password) >= 8
		hasUpper  = false
		hasLower  = false
		hasNumber = false
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		}
	}

	return hasMinLen && hasUpper && hasLower && hasNumber
}

// ValidateTWPhone 驗證台灣手機號碼
func ValidateTWPhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	pattern := `^09[0-9]{8}$`
	matched, _ := regexp.MatchString(pattern, phone)
	return matched
}

// ValidateValidEmail 驗證電子郵件
func ValidateValidEmail(fl validator.FieldLevel) bool {
	email := fl.Field().String()
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// ValidateValidUsername 驗證用戶名
func ValidateValidUsername(fl validator.FieldLevel) bool {
	username := fl.Field().String()

	if len(username) < 3 || len(username) > 20 {
		return false
	}

	pattern := `^[a-zA-Z0-9_-]+$`
	matched, _ := regexp.MatchString(pattern, username)
	return matched
}

// ValidateValidAmount 驗證金額
func ValidateValidAmount(fl validator.FieldLevel) bool {
	amount := fl.Field().Float()
	return amount > 0 && amount <= 999999999.99
}

// 以下是為了向後兼容的全局函數

// ValidatePassword 驗證密碼是否符合強密碼要求
func ValidatePassword(password string) bool {
	var (
		hasMinLen = len(password) >= 8
		hasUpper  = false
		hasLower  = false
		hasNumber = false
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		}
	}

	return hasMinLen && hasUpper && hasLower && hasNumber
}

// ValidatePhone 驗證手機號碼
func ValidatePhone(phone string) bool {
	pattern := `^09[0-9]{8}$`
	matched, _ := regexp.MatchString(pattern, phone)
	return matched
}

// ValidateEmail 驗證電子郵件
func ValidateEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// ValidateUsername 驗證用戶名
func ValidateUsername(username string) bool {
	if len(username) < 3 || len(username) > 20 {
		return false
	}

	pattern := `^[a-zA-Z0-9_-]+$`
	matched, _ := regexp.MatchString(pattern, username)
	return matched
}

// ValidateAmount 驗證金額
func ValidateAmount(amount float64) bool {
	return amount > 0 && amount <= 999999999.99
}

package utils

import (
	"regexp"
)

// RemoveJSONComments 移除 JSON 字符串中的 JavaScript 風格註解
func RemoveJSONComments(jsonStr string) string {
	// 移除單行註解 (// 註解內容)
	singleLineCommentRegex := regexp.MustCompile(`//.*`)
	noSingleLineComments := singleLineCommentRegex.ReplaceAllString(jsonStr, "")

	// 移除多行註解 (/* 註解內容 */)
	multiLineCommentRegex := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	noComments := multiLineCommentRegex.ReplaceAllString(noSingleLineComments, "")

	// 去除可能留下的多餘逗號
	trailingCommaRegex := regexp.MustCompile(`,\s*}`)
	noTrailingCommas := trailingCommaRegex.ReplaceAllString(noComments, "}")

	trailingCommaArrayRegex := regexp.MustCompile(`,\s*\]`)
	result := trailingCommaArrayRegex.ReplaceAllString(noTrailingCommas, "]")

	return result
}

// CleanJSONString 清理 JSON 字符串，移除註解並修正常見格式問題
func CleanJSONString(jsonStr string) string {
	// 1. 移除註解
	cleaned := RemoveJSONComments(jsonStr)

	// 2. 移除多餘的空白字符
	whitespaceRegex := regexp.MustCompile(`\s+`)
	cleaned = whitespaceRegex.ReplaceAllString(cleaned, " ")

	// 3. 修復缺少引號的鍵名
	// 例如 {key: "value"} -> {"key": "value"}
	unquotedKeyRegex := regexp.MustCompile(`(\{|\,)\s*([a-zA-Z0-9_]+)\s*:`)
	cleaned = unquotedKeyRegex.ReplaceAllString(cleaned, `$1"$2":`)

	return cleaned
}

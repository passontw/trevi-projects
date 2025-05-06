package gameflow

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// SidePicker 處理額外球選邊邏輯
type SidePicker struct{}

// NewSidePicker 創建新的選邊器
func NewSidePicker() *SidePicker {
	return &SidePicker{}
}

// PickSide 選擇額外球一側
func (p *SidePicker) PickSide() (ExtraBallSide, error) {
	// 使用密碼學安全的隨機數生成器
	n, err := rand.Int(rand.Reader, big.NewInt(2))
	if err != nil {
		return "", fmt.Errorf("隨機數生成失敗: %w", err)
	}

	if n.Int64() == 0 {
		return ExtraBallSideLeft, nil
	}
	return ExtraBallSideRight, nil
}

// GetSideName 獲取選邊的顯示名稱
func GetSideName(side ExtraBallSide) string {
	switch side {
	case ExtraBallSideLeft:
		return "左側"
	case ExtraBallSideRight:
		return "右側"
	default:
		return "未知"
	}
}

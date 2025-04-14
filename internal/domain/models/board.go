package models

import (
	"fmt"

	"g38_lottery_servic/internal/domain"
)

type Board [3][3]domain.Symbol

type WinningLine struct {
	Type     string
	Position int
	Symbol   domain.Symbol
	Payout   float64
}

func (b Board) PrintBoard() string {
	var result string

	result += "┌───┬───┬───┐\n"

	// 打印每一行
	for i := 0; i < 3; i++ {
		result += "│"
		for j := 0; j < 3; j++ {
			result += fmt.Sprintf(" %s │", b[i][j])
		}
		result += "\n"

		// 打印分隔線或下邊框
		if i < 2 {
			result += "├───┼───┼───┤\n"
		} else {
			result += "└───┴───┴───┘\n"
		}
	}

	return result
}

func (b Board) GetSymbolCount() map[domain.Symbol]int {
	counts := make(map[domain.Symbol]int)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			counts[b[i][j]]++
		}
	}
	return counts
}

func (b Board) GetAllPositions(symbol domain.Symbol) [][2]int {
	var positions [][2]int
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if b[i][j] == symbol {
				positions = append(positions, [2]int{i, j})
			}
		}
	}
	return positions
}

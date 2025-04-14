package domain

type Symbol int

const (
	Cherry Symbol = iota
	Bell
	Lemon
	Orange
	Star
	Skull
	Crown
	Diamond
	Seven
	BAR
)

type SymbolInfo struct {
	Symbol Symbol
	Weight int
	Payout float64
}

func (s Symbol) String() string {
	symbols := []string{
		"🍒",   // Cherry
		"🔔",   // Bell
		"🍋",   // Lemon
		"🍊",   // Orange
		"⭐",   // Star
		"💀",   // Skull
		"👑",   // Crown
		"💎",   // Diamond
		"7️⃣", // Seven
		"📊",   // BAR
	}
	return symbols[s]
}

func GetSymbolList() []SymbolInfo {
	return []SymbolInfo{
		{Cherry, 10, 2.0},
		{Bell, 8, 2.5},
		{Lemon, 10, 2.0},
		{Orange, 10, 2.0},
		{Star, 6, 3.0},
		{Skull, 4, 5.0},
		{Crown, 4, 5.0},
		{Diamond, 2, 10.0},
		{Seven, 3, 7.0},
		{BAR, 5, 4.0},
	}
}

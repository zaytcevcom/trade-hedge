package valueobjects

import "strings"

// TradingPair представляет торговую пару
type TradingPair struct {
	value string
}

// NewTradingPair создает новую торговую пару
func NewTradingPair(pair string) *TradingPair {
	return &TradingPair{value: pair}
}

// String возвращает строковое представление пары
func (tp *TradingPair) String() string {
	return tp.value
}

// ToBybitFormat конвертирует пару в формат Bybit (убирает слэш)
func (tp *TradingPair) ToBybitFormat() string {
	return strings.ReplaceAll(tp.value, "/", "")
}

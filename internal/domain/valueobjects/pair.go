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

// BaseCurrency возвращает базовую валюту торговой пары (например, XRP для XRP/USDT)
func (tp *TradingPair) BaseCurrency() string {
	parts := strings.Split(tp.value, "/")
	if len(parts) >= 1 {
		return parts[0]
	}
	return tp.value
}

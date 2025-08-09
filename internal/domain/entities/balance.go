package entities

import "fmt"

// Balance представляет баланс аккаунта
type Balance struct {
	Asset     string  // Валюта (например, USDT, BTC)
	Available float64 // Доступный баланс
	Total     float64 // Общий баланс
}

// HasSufficientBalance проверяет, достаточно ли средств для покупки
func (b *Balance) HasSufficientBalance(requiredAmount float64) bool {
	return b.Available >= requiredAmount
}

// String возвращает строковое представление баланса
func (b *Balance) String() string {
	return fmt.Sprintf("%s: доступно %.4f, всего %.4f", b.Asset, b.Available, b.Total)
}

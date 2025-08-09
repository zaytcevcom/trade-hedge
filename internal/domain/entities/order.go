package entities

// OrderSide представляет направление ордера
type OrderSide string

const (
	OrderSideBuy  OrderSide = "Buy"
	OrderSideSell OrderSide = "Sell"
)

// OrderType представляет тип ордера
type OrderType string

const (
	OrderTypeMarket OrderType = "MARKET"
	OrderTypeLimit  OrderType = "LIMIT"
)

// Order представляет торговый ордер
type Order struct {
	Symbol   string
	Side     OrderSide
	Type     OrderType
	Quantity float64
	Price    float64 // Для лимитных ордеров
}

// OrderResult представляет результат размещения ордера
type OrderResult struct {
	OrderID string
	Success bool
	Error   string
}

// NewMarketOrder создает рыночный ордер
func NewMarketOrder(symbol string, side OrderSide, quantity float64) *Order {
	return &Order{
		Symbol:   symbol,
		Side:     side,
		Type:     OrderTypeMarket,
		Quantity: quantity,
		Price:    0, // Цена не нужна для рыночного ордера
	}
}

// NewLimitOrder создает лимитный ордер
func NewLimitOrder(symbol string, side OrderSide, quantity, price float64) *Order {
	return &Order{
		Symbol:   symbol,
		Side:     side,
		Type:     OrderTypeLimit,
		Quantity: quantity,
		Price:    price,
	}
}

// CalculateQuantityFromAmount рассчитывает количество валюты для покупки на определенную сумму
func CalculateQuantityFromAmount(amount, currentPrice float64) float64 {
	return amount / currentPrice
}

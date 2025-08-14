package services

import (
	"context"
	"time"
	"trade-hedge/internal/domain/entities"
)

// OrderStatusInfo информация о статусе ордера
type OrderStatusInfo struct {
	OrderID      string
	Status       entities.OrderStatus
	FilledPrice  *float64   // Цена исполнения (если исполнен)
	FilledTime   *time.Time // Время исполнения (если исполнен)
	FilledQty    float64    // Исполненное количество
	RemainingQty float64    // Остаток количества
}

// InstrumentInfo информация об инструменте (минимальные лимиты, размеры шагов и т.д.)
type InstrumentInfo struct {
	Symbol      string  // Символ инструмента (например, SOLUSDT)
	BaseCoin    string  // Базовая валюта (например, SOL)
	QuoteCoin   string  // Котируемая валюта (например, USDT)
	MinOrderQty float64 // Минимальное количество для ордера
	MinOrderAmt float64 // Минимальная сумма ордера в котируемой валюте
	MaxOrderQty float64 // Максимальное количество для ордера
	MaxOrderAmt float64 // Максимальная сумма ордера в котируемой валюте
	TickSize    float64 // Минимальный шаг цены
	StepSize    float64 // Минимальный шаг количества
	Status      string  // Статус инструмента (Trading, Break, etc.)
}

// ExchangeService определяет интерфейс для работы с биржей
type ExchangeService interface {
	// PlaceOrder размещает ордер на бирже
	PlaceOrder(ctx context.Context, order *entities.Order) (*entities.OrderResult, error)

	// GetBalance получает баланс по определенной валюте
	GetBalance(ctx context.Context, asset string) (*entities.Balance, error)

	// GetOrderStatus получает статус ордера по ID
	GetOrderStatus(ctx context.Context, orderID, symbol string) (*OrderStatusInfo, error)

	// GetInstrumentInfo получает информацию об инструменте (минимальные лимиты, размеры шагов)
	GetInstrumentInfo(ctx context.Context, symbol string) (*InstrumentInfo, error)
}

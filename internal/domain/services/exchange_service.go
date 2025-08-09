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

// ExchangeService определяет интерфейс для работы с биржей
type ExchangeService interface {
	// PlaceOrder размещает ордер на бирже
	PlaceOrder(ctx context.Context, order *entities.Order) (*entities.OrderResult, error)

	// GetBalance получает баланс по определенной валюте
	GetBalance(ctx context.Context, asset string) (*entities.Balance, error)

	// GetOrderStatus получает статус ордера по ID
	GetOrderStatus(ctx context.Context, orderID, symbol string) (*OrderStatusInfo, error)
}

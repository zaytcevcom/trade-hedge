package services

import (
	"context"
	"trade-hedge/internal/domain/entities"
	"trade-hedge/internal/domain/services"
	"trade-hedge/internal/infrastructure/clients"
)

// ExchangeServiceAdapter адаптер для сервиса биржи
type ExchangeServiceAdapter struct {
	bybitClient *clients.BybitClient
}

// NewExchangeServiceAdapter создает новый адаптер сервиса биржи
func NewExchangeServiceAdapter(bybitClient *clients.BybitClient) *ExchangeServiceAdapter {
	return &ExchangeServiceAdapter{
		bybitClient: bybitClient,
	}
}

// PlaceOrder размещает ордер на бирже
func (e *ExchangeServiceAdapter) PlaceOrder(ctx context.Context, order *entities.Order) (*entities.OrderResult, error) {
	return e.bybitClient.PlaceOrder(ctx, order)
}

// GetBalance получает баланс по определенной валюте
func (e *ExchangeServiceAdapter) GetBalance(ctx context.Context, asset string) (*entities.Balance, error) {
	return e.bybitClient.GetBalance(ctx, asset)
}

// GetOrderStatus получает статус ордера по ID
func (e *ExchangeServiceAdapter) GetOrderStatus(ctx context.Context, orderID, symbol string) (*services.OrderStatusInfo, error) {
	return e.bybitClient.GetOrderStatus(ctx, orderID, symbol)
}

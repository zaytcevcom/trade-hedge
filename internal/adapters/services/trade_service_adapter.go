package services

import (
	"context"
	"trade-hedge/internal/domain/entities"
	"trade-hedge/internal/infrastructure/clients"
)

// TradeServiceAdapter адаптер для сервиса получения сделок
type TradeServiceAdapter struct {
	freqtradeClient *clients.FreqtradeClient
}

// NewTradeServiceAdapter создает новый адаптер сервиса сделок
func NewTradeServiceAdapter(freqtradeClient *clients.FreqtradeClient) *TradeServiceAdapter {
	return &TradeServiceAdapter{
		freqtradeClient: freqtradeClient,
	}
}

// GetActiveTrades получает активные сделки из Freqtrade
func (t *TradeServiceAdapter) GetActiveTrades(ctx context.Context) ([]*entities.Trade, error) {
	return t.freqtradeClient.GetActiveTrades(ctx)
}

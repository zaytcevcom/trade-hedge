package services

import (
	"context"
	"trade-hedge/internal/domain/entities"
)

// TradeService отвечает за получение данных о сделках из внешних источников
type TradeService interface {
	// GetActiveTrades получает активные сделки из торговой платформы
	GetActiveTrades(ctx context.Context) ([]*entities.Trade, error)
}

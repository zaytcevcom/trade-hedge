package repositories

import (
	"context"
	"time"
	"trade-hedge/internal/domain/entities"
)

// HedgeRepository отвечает только за сохранение данных о хеджировании
type HedgeRepository interface {
	// IsTradeHedged проверяет, была ли сделка хеджирована
	IsTradeHedged(ctx context.Context, tradeID int) (bool, error)

	// SaveHedgedTrade сохраняет информацию о хеджированной сделке
	SaveHedgedTrade(ctx context.Context, hedgedTrade *entities.HedgedTrade) error

	// GetActiveHedgedTrades получает все активные хеджированные сделки
	GetActiveHedgedTrades(ctx context.Context) ([]*entities.HedgedTrade, error)

	// UpdateHedgedTradeStatus обновляет статус хеджированной сделки
	UpdateHedgedTradeStatus(ctx context.Context, orderID string, status entities.OrderStatus, closePrice *float64, closeTime *time.Time) error
}

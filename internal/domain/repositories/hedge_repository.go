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

	// GetHedgedTrades получает хеджированные сделки по статусу
	// Если status = nil, возвращает все сделки
	// Если status указан, возвращает сделки только с этим статусом
	GetHedgedTrades(ctx context.Context, status *string) ([]*entities.HedgedTrade, error)

	// UpdateHedgedTradeStatus обновляет статус хеджированной сделки
	UpdateHedgedTradeStatus(ctx context.Context, orderID string, status entities.OrderStatus, closePrice *float64, closeTime *time.Time) error

	// GetHedgeHistory получает историю хедж-ордеров по конкретной сделке
	GetHedgeHistory(ctx context.Context, tradeID int) ([]*entities.HedgedTrade, error)
}

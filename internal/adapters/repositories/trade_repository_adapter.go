package repositories

import (
	"context"
	"time"
	"trade-hedge/internal/domain/entities"
	"trade-hedge/internal/infrastructure/database"
)

// HedgeRepositoryAdapter адаптер для репозитория хеджирования
type HedgeRepositoryAdapter struct {
	dbRepo *database.PostgreSQLTradeRepository
}

// NewHedgeRepositoryAdapter создает новый адаптер репозитория
func NewHedgeRepositoryAdapter(
	dbRepo *database.PostgreSQLTradeRepository,
) *HedgeRepositoryAdapter {
	return &HedgeRepositoryAdapter{
		dbRepo: dbRepo,
	}
}

// IsTradeHedged проверяет, была ли сделка хеджирована
func (r *HedgeRepositoryAdapter) IsTradeHedged(ctx context.Context, tradeID int) (bool, error) {
	return r.dbRepo.IsTradeHedged(ctx, tradeID)
}

// SaveHedgedTrade сохраняет информацию о хеджированной сделке
func (r *HedgeRepositoryAdapter) SaveHedgedTrade(ctx context.Context, hedgedTrade *entities.HedgedTrade) error {
	return r.dbRepo.SaveHedgedTrade(ctx, hedgedTrade)
}

// GetActiveHedgedTrades получает все активные хеджированные сделки
func (r *HedgeRepositoryAdapter) GetActiveHedgedTrades(ctx context.Context) ([]*entities.HedgedTrade, error) {
	return r.dbRepo.GetActiveHedgedTrades(ctx)
}

// UpdateHedgedTradeStatus обновляет статус хеджированной сделки
func (r *HedgeRepositoryAdapter) UpdateHedgedTradeStatus(ctx context.Context, orderID string, status entities.OrderStatus, closePrice *float64, closeTime *time.Time) error {
	return r.dbRepo.UpdateHedgedTradeStatus(ctx, orderID, status, closePrice, closeTime)
}

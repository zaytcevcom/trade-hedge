package usecases

import (
	"context"
	"fmt"
	"time"

	"trade-hedge/internal/domain/entities"
	"trade-hedge/internal/domain/repositories"
	"trade-hedge/internal/domain/services"
	"trade-hedge/internal/pkg/logger"
)

// StatusCheckerUseCase отвечает за проверку статусов всех активных хеджированных ордеров
type StatusCheckerUseCase struct {
	hedgeRepo       repositories.HedgeRepository
	exchangeService services.ExchangeService
}

// NewStatusCheckerUseCase создает новый use case для проверки статусов
func NewStatusCheckerUseCase(
	hedgeRepo repositories.HedgeRepository,
	exchangeService services.ExchangeService,
) *StatusCheckerUseCase {
	return &StatusCheckerUseCase{
		hedgeRepo:       hedgeRepo,
		exchangeService: exchangeService,
	}
}

// CheckAllActiveOrders проверяет статусы всех активных хеджированных ордеров
func (s *StatusCheckerUseCase) CheckAllActiveOrders(ctx context.Context) error {
	logger.LogWithTime("🔍 Начинаем проверку статусов активных хеджированных ордеров...")

	// 1. Получаем все активные хеджированные сделки
	pendingStatus := "PENDING"
	activeTrades, err := s.hedgeRepo.GetHedgedTrades(ctx, &pendingStatus)
	if err != nil {
		return fmt.Errorf("ошибка получения активных хеджированных сделок: %w", err)
	}

	if len(activeTrades) == 0 {
		logger.LogWithTime("✅ Активных хеджированных ордеров не найдено")
		return nil
	}

	logger.LogWithTime("📊 Найдено %d активных хеджированных ордеров для проверки", len(activeTrades))

	// 2. Проверяем статус каждого ордера
	updatedCount := 0
	for _, trade := range activeTrades {
		updated, err := s.checkSingleOrderStatus(ctx, trade)
		if err != nil {
			logger.LogWithTime("❌ Ошибка проверки ордера %s (пара %s): %v",
				trade.BybitOrderID, trade.Pair, err)
			continue
		}

		if updated {
			updatedCount++
		}
	}

	logger.LogWithTime("✅ Проверка завершена. Обновлено статусов: %d из %d", updatedCount, len(activeTrades))
	return nil
}

// checkSingleOrderStatus проверяет статус одного ордера
func (s *StatusCheckerUseCase) checkSingleOrderStatus(ctx context.Context, trade *entities.HedgedTrade) (bool, error) {
	// Получаем актуальный статус с биржи
	statusInfo, err := s.exchangeService.GetOrderStatus(ctx, trade.BybitOrderID, trade.Pair)
	if err != nil {
		return false, fmt.Errorf("ошибка получения статуса ордера: %w", err)
	}

	// Проверяем, изменился ли статус
	if statusInfo.Status == trade.OrderStatus {
		// Статус не изменился, обновляем только время последней проверки
		err := s.hedgeRepo.UpdateHedgedTradeStatus(ctx, trade.BybitOrderID, trade.OrderStatus, trade.ClosePrice, trade.CloseTime)
		if err != nil {
			return false, fmt.Errorf("ошибка обновления времени проверки: %w", err)
		}
		return false, nil
	}

	// Статус изменился
	logger.LogWithTime("🔄 Ордер %s (пара %s): %s → %s",
		trade.BybitOrderID, trade.Pair, trade.OrderStatus, statusInfo.Status)

	// Подготавливаем данные для обновления
	var closePrice *float64
	var closeTime *time.Time

	// Если ордер исполнен, сохраняем цену и время исполнения
	if statusInfo.Status == entities.OrderStatusFilled {
		closePrice = statusInfo.FilledPrice
		closeTime = statusInfo.FilledTime

		// Рассчитываем и выводим прибыль
		if closePrice != nil {
			profit := (*closePrice - trade.HedgeOpenPrice) * trade.HedgeAmount
			logger.LogWithTime("💰 Хеджирование завершено! Прибыль: %.4f USDT", profit)
			logger.LogWithTime("   📈 Открытие: %.4f, Закрытие: %.4f, Количество: %.4f",
				trade.HedgeOpenPrice, *closePrice, trade.HedgeAmount)
		}
	} else if statusInfo.Status.IsCompleted() {
		// Ордер завершен неуспешно (отменен или отклонен)
		now := time.Now()
		closeTime = &now
		logger.LogWithTime("❌ Ордер %s завершен неуспешно: %s", trade.BybitOrderID, statusInfo.Status)
	}

	// Обновляем статус в базе данных
	err = s.hedgeRepo.UpdateHedgedTradeStatus(ctx, trade.BybitOrderID, statusInfo.Status, closePrice, closeTime)
	if err != nil {
		return false, fmt.Errorf("ошибка обновления статуса в БД: %w", err)
	}

	return true, nil
}

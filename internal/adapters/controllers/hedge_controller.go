package controllers

import (
	"context"
	"errors"
	domainErrors "trade-hedge/internal/domain/errors"
	"trade-hedge/internal/pkg/logger"
	"trade-hedge/internal/usecases"
)

// HedgeController контроллер для выполнения стратегии хеджирования
type HedgeController struct {
	hedgeUseCase *usecases.HedgeStrategyUseCase
}

// NewHedgeController создает новый контроллер
func NewHedgeController(hedgeUseCase *usecases.HedgeStrategyUseCase) *HedgeController {
	return &HedgeController{
		hedgeUseCase: hedgeUseCase,
	}
}

// ExecuteHedgeStrategy выполняет стратегию хеджирования с выводом результатов
func (h *HedgeController) ExecuteHedgeStrategy(ctx context.Context) {
	logger.LogWithTime("🚀 Запуск стратегии хеджирования убытков")

	err := h.hedgeUseCase.ExecuteHedgeStrategy(ctx)
	if err != nil {
		// Проверяем на типизированные ошибки стратегии
		var strategyErr *domainErrors.StrategyError
		if errors.As(err, &strategyErr) && strategyErr.IsExpected() {
			logger.LogWithTime("✅ %s. Действия не требуются", err.Error())
			return
		}
		// Используем log.Printf вместо log.Fatalf чтобы не останавливать приложение
		logger.LogWithTime("❌ Ошибка выполнения стратегии: %v", err)
		return
	}

	logger.LogWithTime("🎉 Хеджирование выполнено успешно!")
	logger.LogWithTime("💾 Полная информация о сделке сохранена в базе данных")
}

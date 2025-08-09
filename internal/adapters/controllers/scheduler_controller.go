package controllers

import (
	"context"
	"time"
	"trade-hedge/internal/pkg/logger"
	"trade-hedge/internal/usecases"
)

// SchedulerController контроллер для периодического выполнения стратегии
type SchedulerController struct {
	hedgeUseCase         *usecases.HedgeStrategyUseCase
	statusCheckerUseCase *usecases.StatusCheckerUseCase
	interval             time.Duration
}

// NewSchedulerController создает новый scheduler контроллер
func NewSchedulerController(hedgeUseCase *usecases.HedgeStrategyUseCase, statusCheckerUseCase *usecases.StatusCheckerUseCase, interval time.Duration) *SchedulerController {
	return &SchedulerController{
		hedgeUseCase:         hedgeUseCase,
		statusCheckerUseCase: statusCheckerUseCase,
		interval:             interval,
	}
}

// Start запускает периодическое выполнение стратегии
func (s *SchedulerController) Start(ctx context.Context) {
	logger.LogWithTime("🕒 Запуск периодической проверки каждые %v", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Выполняем сразу при запуске
	s.executeStrategy(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.LogWithTime("🛑 Получен сигнал остановки")
			return
		case <-ticker.C:
			s.executeStrategy(ctx)
		}
	}
}

// executeStrategy выполняет одну итерацию стратегии
func (s *SchedulerController) executeStrategy(ctx context.Context) {
	// Добавляем отступ для лучшей читаемости логов
	logger.LogPlain("\n")
	logger.LogWithTime("⏰ Проверка позиций...")

	// 1. Сначала проверяем статусы существующих хеджированных ордеров
	if err := s.statusCheckerUseCase.CheckAllActiveOrders(ctx); err != nil {
		logger.LogWithTime("❌ Ошибка проверки статусов ордеров: %v", err)
	}

	// 2. Затем проверяем новые сделки для хеджирования
	hedgeController := NewHedgeController(s.hedgeUseCase)
	hedgeController.ExecuteHedgeStrategy(ctx)
}

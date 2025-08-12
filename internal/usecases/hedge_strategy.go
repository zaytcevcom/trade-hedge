package usecases

import (
	"context"
	"fmt"
	"time"
	"trade-hedge/internal/domain/entities"
	"trade-hedge/internal/domain/errors"
	"trade-hedge/internal/domain/repositories"
	"trade-hedge/internal/domain/services"
	"trade-hedge/internal/domain/valueobjects"
	"trade-hedge/internal/pkg/logger"
)

// HedgeStrategyConfig конфигурация стратегии хеджирования
type HedgeStrategyConfig struct {
	PositionAmount float64 // Фиксированная сумма позиции в базовой валюте
	MaxLossPercent float64
	ProfitRatio    float64
	BaseCurrency   string // Базовая валюта для покупки (например, USDT)
	RetryAttempts  int    // Количество попыток размещения ордера
	RetryDelay     int    // Задержка между попытками в секундах
}

// HedgeStrategyUseCase реализует сценарий хеджирования убытков
type HedgeStrategyUseCase struct {
	tradeService    services.TradeService
	hedgeRepo       repositories.HedgeRepository
	exchangeService services.ExchangeService
	config          *HedgeStrategyConfig
}

// NewHedgeStrategyUseCase создает новый экземпляр use case
func NewHedgeStrategyUseCase(
	tradeService services.TradeService,
	hedgeRepo repositories.HedgeRepository,
	exchangeService services.ExchangeService,
	config *HedgeStrategyConfig,
) *HedgeStrategyUseCase {

	return &HedgeStrategyUseCase{
		tradeService:    tradeService,
		hedgeRepo:       hedgeRepo,
		exchangeService: exchangeService,
		config:          config,
	}
}

// ExecuteHedgeStrategy выполняет стратегию хеджирования
func (h *HedgeStrategyUseCase) ExecuteHedgeStrategy(ctx context.Context) error {
	// 1. Получаем все активные сделки
	trades, err := h.tradeService.GetActiveTrades(ctx)
	if err != nil {
		return fmt.Errorf("ошибка получения активных сделок: %w", err)
	}

	// 2. Фильтруем уже хеджированные сделки
	unhedgedTrades, err := h.filterUnhedgedTrades(ctx, trades)
	if err != nil {
		return fmt.Errorf("ошибка фильтрации хеджированных сделок: %w", err)
	}

	if len(unhedgedTrades) == 0 {
		return errors.NewNoTradesError()
	}

	// 3. Находим сделку для хеджирования
	tradeToHedge := h.findTradeToHedge(unhedgedTrades)
	if tradeToHedge == nil {
		return errors.NewNoLossyTradesError(h.config.MaxLossPercent)
	}

	// 4. Выполняем хеджирование
	return h.hedgeTrade(ctx, tradeToHedge)
}

// filterUnhedgedTrades фильтрует сделки, исключая уже хеджированные
func (h *HedgeStrategyUseCase) filterUnhedgedTrades(ctx context.Context, trades []*entities.Trade) ([]*entities.Trade, error) {
	var unhedged []*entities.Trade

	for _, trade := range trades {
		isHedged, err := h.hedgeRepo.IsTradeHedged(ctx, trade.ID)
		if err != nil {
			return nil, fmt.Errorf("ошибка проверки хеджирования для сделки %d: %w", trade.ID, err)
		}

		if !isHedged {
			unhedged = append(unhedged, trade)
		}
	}

	return unhedged, nil
}

// findTradeToHedge находит первую подходящую сделку для хеджирования
func (h *HedgeStrategyUseCase) findTradeToHedge(trades []*entities.Trade) *entities.Trade {
	for _, trade := range trades {
		if trade.ShouldBeHedged(h.config.MaxLossPercent) {
			return trade
		}
	}
	return nil
}

// hedgeTrade выполняет хеджирование конкретной сделки
func (h *HedgeStrategyUseCase) hedgeTrade(ctx context.Context, trade *entities.Trade) error {
	pair := valueobjects.NewTradingPair(trade.Pair)
	symbol := pair.ToBybitFormat()

	// 1. Проверяем баланс базовой валюты
	balance, err := h.exchangeService.GetBalance(ctx, h.config.BaseCurrency)
	if err != nil {
		return fmt.Errorf("ошибка получения баланса %s: %w", h.config.BaseCurrency, err)
	}

	// Рассчитываем необходимую сумму для покупки с запасом на проскальзывание
	requiredAmount := h.config.PositionAmount * 1.01 // +1% запас на проскальзывание

	if !balance.HasSufficientBalance(requiredAmount) {
		return errors.NewInsufficientBalanceError(requiredAmount, balance.Available, h.config.BaseCurrency)
	}

	// Рассчитываем количество валюты для покупки на фиксированную сумму
	orderQuantity := entities.CalculateQuantityFromAmount(h.config.PositionAmount, trade.CurrentRate)

	// Проверяем минимальный лимит ордера (Bybit требует минимум 5 USDT для большинства пар)
	minOrderValue := 5.0 // Минимальная стоимость ордера в USDT
	orderValue := h.config.PositionAmount

	if orderValue < minOrderValue {
		logger.LogWithTime("⚠️ ВНИМАНИЕ: Стоимость ордера %.2f %s меньше минимального лимита %.2f %s",
			orderValue, h.config.BaseCurrency, minOrderValue, h.config.BaseCurrency)
		logger.LogWithTime("💡 Рекомендуется увеличить position_amount в конфигурации до минимум %.2f %s",
			minOrderValue, h.config.BaseCurrency)
	}

	logger.LogPlain("💰 Баланс %s: доступно %.4f, требуется %.4f\n",
		h.config.BaseCurrency, balance.Available, requiredAmount)
	logger.LogPlain("📊 Исходная сделка Freqtrade: %.6f %s по цене %.4f (убыток %.2f%%)\n",
		trade.Amount, pair.String(), trade.OpenRate, trade.ProfitRatio*100)
	logger.LogPlain("🛒 Хеджирующая покупка: %.6f %s на сумму %.2f %s по цене %.4f\n",
		orderQuantity, pair.ToBybitFormat(), h.config.PositionAmount, h.config.BaseCurrency, trade.CurrentRate)

	// 2. Размещаем рыночный ордер на покупку
	buyOrder := entities.NewMarketOrder(symbol, entities.OrderSideBuy, orderQuantity)
	buyResult, err := h.exchangeService.PlaceOrder(ctx, buyOrder)
	if err != nil {
		return fmt.Errorf("ошибка размещения ордера на покупку: %w", err)
	}

	if !buyResult.Success {
		return fmt.Errorf("неудачное размещение ордера на покупку: %s", buyResult.Error)
	}

	// 3. Проверяем фактическое исполнение ордера на покупку
	buyOrderStatus, err := h.exchangeService.GetOrderStatus(ctx, buyResult.OrderID, symbol)
	if err != nil {
		return fmt.Errorf("ошибка получения статуса ордера на покупку: %w", err)
	}

	// Используем фактически купленное количество для ордера на продажу
	actualQuantity := buyOrderStatus.FilledQty
	if actualQuantity <= 0 {
		return fmt.Errorf("ордер на покупку не был исполнен или исполнен на 0")
	}

	// Проверяем на частичное исполнение
	fillRatio := actualQuantity / orderQuantity
	if fillRatio < 0.95 { // Если исполнено менее 95%
		logger.LogWithTime("⚠️ ЧАСТИЧНОЕ ИСПОЛНЕНИЕ: куплено %.4f %s из %.4f (%.1f%%)",
			actualQuantity, pair.ToBybitFormat(), orderQuantity, fillRatio*100)
		logger.LogWithTime("💡 Возможные причины: недостаток ликвидности, большой спред, волатильность")
	} else {
		logger.LogWithTime("✅ Полное исполнение: куплено %.4f %s из %.4f (%.1f%%)",
			actualQuantity, pair.ToBybitFormat(), orderQuantity, fillRatio*100)
	}

	// 4. Рассчитываем цену тейк-профита
	takeProfitPrice := trade.CalculateTakeProfitPrice(h.config.ProfitRatio)
	logger.LogWithTime("🎯 Лимитный ордер на продажу: %.4f %s по цене %.4f (тейк-профит)",
		actualQuantity, pair.ToBybitFormat(), takeProfitPrice)

	// 5. Размещаем лимитный ордер на продажу с ретраями
	sellOrder := entities.NewLimitOrder(symbol, entities.OrderSideSell, actualQuantity, takeProfitPrice)

	var sellResult *entities.OrderResult
	maxRetries := h.config.RetryAttempts
	retryDelay := time.Duration(h.config.RetryDelay) * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		logger.LogWithTime("📤 Попытка %d/%d размещения ордера на продажу", attempt, maxRetries)

		sellResult, err = h.exchangeService.PlaceOrder(ctx, sellOrder)
		if err == nil && sellResult.Success {
			logger.LogWithTime("✅ Ордер на продажу успешно размещен с попытки %d", attempt)
			break
		}

		if attempt < maxRetries {
			logger.LogWithTime("⚠️ Попытка %d неудачна, ждем %v перед повтором: %v",
				attempt, retryDelay, err)
			time.Sleep(retryDelay)
		} else {
			if err != nil {
				return fmt.Errorf("ошибка размещения ордера на продажу после %d попыток: %w", maxRetries, err)
			}
			return fmt.Errorf("неудачное размещение ордера на продажу после %d попыток: %s", maxRetries, sellResult.Error)
		}
	}

	// 5. Сохраняем полную информацию о хеджировании
	now := time.Now()
	hedgedTrade := &entities.HedgedTrade{
		FreqtradeTradeID: trade.ID,
		Pair:             trade.Pair,
		HedgeTime:        now,
		BybitOrderID:     sellResult.OrderID,

		// Информация об исходной сделке Freqtrade
		FreqtradeOpenPrice:   trade.OpenRate,
		FreqtradeAmount:      trade.Amount,
		FreqtradeProfitRatio: trade.ProfitRatio,

		// Информация о хеджирующей позиции
		HedgeOpenPrice:       trade.CurrentRate,
		HedgeAmount:          actualQuantity,
		HedgeTakeProfitPrice: takeProfitPrice,

		// Статус ордера
		OrderStatus:     entities.OrderStatusPending,
		LastStatusCheck: &now,
		ClosePrice:      nil,
		CloseTime:       nil,
	}

	if err := h.hedgeRepo.SaveHedgedTrade(ctx, hedgedTrade); err != nil {
		return fmt.Errorf("ошибка сохранения хеджированной сделки: %w", err)
	}

	return nil
}

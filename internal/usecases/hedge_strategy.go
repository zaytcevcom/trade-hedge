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
		logger.LogWithTime("⚠️ ЧАСТИЧНОЕ ИСПОЛНЕНИЕ: куплено %.6f %s из %.6f (%.1f%%)",
			actualQuantity, pair.ToBybitFormat(), orderQuantity, fillRatio*100)
	} else {
		logger.LogWithTime("✅ Полное исполнение: куплено %.6f %s из %.6f (%.1f%%)",
			actualQuantity, pair.ToBybitFormat(), orderQuantity, fillRatio*100)
	}

	// 4. Рассчитываем цену тейк-профита
	takeProfitPrice := trade.CalculateTakeProfitPrice(h.config.ProfitRatio)

	// 5. Размещаем лимитный ордер на продажу фактически купленного количества
	sellOrder := entities.NewLimitOrder(symbol, entities.OrderSideSell, actualQuantity, takeProfitPrice)
	sellResult, err := h.exchangeService.PlaceOrder(ctx, sellOrder)
	if err != nil {
		return fmt.Errorf("ошибка размещения ордера на продажу: %w", err)
	}

	if !sellResult.Success {
		return fmt.Errorf("неудачное размещение ордера на продажу: %s", sellResult.Error)
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

package usecases

import (
	"context"
	"fmt"
	"math"
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

// GetExchangeService возвращает сервис для работы с биржей
func (h *HedgeStrategyUseCase) GetExchangeService() services.ExchangeService {
	return h.exchangeService
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

	// 3. Сортируем сделки по максимальной просадке (от большей к меньшей)
	entities.SortTradesByDrawdown(unhedgedTrades)
	logger.LogWithTime("📊 Отсортировали %d сделок по просадке (от большей к меньшей)", len(unhedgedTrades))

	// Логируем детали сортировки для всех сделок
	logger.LogWithTime("📋 Детали сортировки сделок:")
	for i, trade := range unhedgedTrades {
		drawdownPercent := trade.ProfitRatio * -100
		logger.LogWithTime("   %d. %s: просадка %.2f%%", i+1, trade.Pair, drawdownPercent)
	}

	// 4. Находим и пытаемся хеджировать подходящие сделки
	return h.findAndHedgeTrade(ctx, unhedgedTrades)
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

// findAndHedgeTrade находит и пытается хеджировать подходящую сделку
func (h *HedgeStrategyUseCase) findAndHedgeTrade(ctx context.Context, trades []*entities.Trade) error {
	var lastError error
	var triedPairs []string

	logger.LogWithTime("🎯 Начинаем поиск сделок для хеджирования (отсортированы по просадке)")

	// Пытаемся найти подходящую сделку для хеджирования
	for i, trade := range trades {
		drawdownPercent := trade.ProfitRatio * -100 // Конвертируем в проценты

		if !trade.ShouldBeHedged(h.config.MaxLossPercent) {
			logger.LogWithTime("⏭️ [%d/%d] Пропускаем пару %s (просадка: %.2f%% < порог %.2f%%)",
				i+1, len(trades), trade.Pair, drawdownPercent, h.config.MaxLossPercent)
			continue
		}

		pair := valueobjects.NewTradingPair(trade.Pair)
		triedPairs = append(triedPairs, pair.String())

		// Логируем просадку для каждой сделки
		logger.LogWithTime("🔍 [%d/%d] Пробуем хеджировать пару %s (просадка: %.2f%%)...",
			i+1, len(trades), pair.String(), drawdownPercent)

		// Пытаемся выполнить хеджирование
		err := h.hedgeTrade(ctx, trade)
		if err == nil {
			// Успешно хеджировали
			logger.LogWithTime("✅ Успешно хеджировали пару %s", pair.String())
			return nil
		}

		// Проверяем тип ошибки
		if strategyErr, ok := err.(*errors.StrategyError); ok {
			if strategyErr.Type == errors.ErrorTypeInsufficientBalanceForMinLimit {
				// Это ожидаемая ошибка - пара не подходит по минимальному лимиту
				logger.LogWithTime("⚠️ Пара %s не подходит по минимальному лимиту, пробуем следующую...", pair.String())
				lastError = err
				continue // Продолжаем искать другие пары
			}
		}

		// Другие ошибки - возвращаем их
		logger.LogWithTime("❌ Ошибка хеджирования пары %s: %v", pair.String(), err)
		return err
	}

	// Если дошли до сюда, значит все подходящие пары не удалось хеджировать
	if lastError != nil {
		logger.LogWithTime("⚠️ Все подходящие пары (%v) не удалось хеджировать", triedPairs)
		return lastError
	}

	// Нет подходящих сделок для хеджирования
	logger.LogWithTime("ℹ️ Обработано %d сделок, подходящих для хеджирования не найдено", len(trades))
	return errors.NewNoLossyTradesError(h.config.MaxLossPercent)
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

	// Проверяем, достаточно ли баланса для указанной в настройках суммы позиции
	// Если баланса недостаточно - пропускаем пару, НЕ корректируем размер позиции
	if !balance.HasSufficientBalance(requiredAmount) {
		logger.LogWithTime("⚠️ ВНИМАНИЕ: Недостаточно баланса для запрошенной позиции")
		logger.LogWithTime("💡 Требуется: %.2f %s, доступно: %.2f %s",
			requiredAmount, h.config.BaseCurrency, balance.Available, h.config.BaseCurrency)
		logger.LogWithTime("💡 Пропускаем пару %s - недостаточно баланса для указанной суммы позиции", pair.String())
		return errors.NewInsufficientBalanceError(requiredAmount, balance.Available, h.config.BaseCurrency)
	}

	// Используем фиксированный размер позиции из настроек (без автоматической корректировки)
	adjustedPositionAmount := h.config.PositionAmount

	// Рассчитываем количество валюты для покупки на фиксированную сумму
	orderQuantity := entities.CalculateQuantityFromAmount(adjustedPositionAmount, trade.CurrentRate)

	// Получаем минимальный лимит ордера для конкретной пары от Bybit API
	instrumentInfo, err := h.exchangeService.GetInstrumentInfo(ctx, symbol)
	if err != nil {
		logger.LogWithTime("⚠️ Не удалось получить информацию об инструменте %s: %v", symbol, err)
		logger.LogWithTime("💡 Используем безопасное значение по умолчанию: 100 USDT")
		// Используем безопасное значение по умолчанию
		instrumentInfo = &services.InstrumentInfo{
			MinOrderAmt: 100.0,
		}
	}

	// Проверяем корректность полученного минимального лимита
	minOrderValue := instrumentInfo.MinOrderAmt
	if minOrderValue <= 0 {
		logger.LogWithTime("⚠️ ВНИМАНИЕ: Bybit вернул некорректный минимальный лимит: %.2f USDT", minOrderValue)
		logger.LogWithTime("💡 Используем безопасное значение по умолчанию: 100 USDT")
		minOrderValue = 100.0
	}

	// Проверяем минимальное количество валюты
	minOrderQty := instrumentInfo.MinOrderQty
	if minOrderQty <= 0 {
		logger.LogWithTime("⚠️ ВНИМАНИЕ: Bybit вернул некорректное минимальное количество: %.6f", minOrderQty)
		logger.LogWithTime("💡 Используем безопасное значение по умолчанию: 0.001")
		minOrderQty = 0.001
	}

	// Округляем количество до правильной точности согласно basePrecision от Bybit
	stepSize := instrumentInfo.StepSize
	if stepSize > 0 {
		// Округляем до ближайшего кратного stepSize
		orderQuantity = math.Round(orderQuantity/stepSize) * stepSize
		logger.LogWithTime("🔧 Количество скорректировано до шага %.6f: %.6f → %.6f", stepSize, entities.CalculateQuantityFromAmount(adjustedPositionAmount, trade.CurrentRate), orderQuantity)
	}

	orderValue := adjustedPositionAmount

	// Проверяем минимальную сумму ордера
	if orderValue < minOrderValue {
		logger.LogWithTime("⚠️ ВНИМАНИЕ: Стоимость ордера %.2f %s меньше минимального лимита %.2f %s для пары %s",
			orderValue, h.config.BaseCurrency, minOrderValue, h.config.BaseCurrency, pair.String())
		logger.LogWithTime("💡 Минимальный лимит получен от Bybit API: %s", symbol)

		logger.LogWithTime("💡 Пропускаем пару %s - размер позиции меньше минимального лимита", pair.String())
		return errors.NewInsufficientBalanceForMinLimitError(minOrderValue, adjustedPositionAmount, h.config.BaseCurrency)
	}

	// Проверяем минимальное количество валюты
	if orderQuantity < minOrderQty {
		logger.LogWithTime("⚠️ ВНИМАНИЕ: Количество валюты %.6f %s меньше минимального лимита %.6f для пары %s",
			orderQuantity, pair.ToBybitFormat(), minOrderQty, pair.String())
		logger.LogWithTime("💡 Минимальное количество получено от Bybit API: %s", symbol)

		logger.LogWithTime("💡 Пропускаем пару %s - количество меньше минимального лимита", pair.String())
		return errors.NewInsufficientBalanceForMinLimitError(minOrderValue, adjustedPositionAmount, h.config.BaseCurrency)
	}

	logger.LogWithTime("✅ Стоимость ордера %.2f %s соответствует минимальному лимиту %.2f %s",
		orderValue, h.config.BaseCurrency, minOrderValue, h.config.BaseCurrency)
	logger.LogWithTime("✅ Количество валюты %.6f %s соответствует минимальному лимиту %.6f",
		orderQuantity, pair.ToBybitFormat(), minOrderQty)
	logger.LogWithTime("💡 Минимальные лимиты получены от Bybit API: %s", symbol)

	logger.LogPlain("💰 Баланс %s: доступно %.4f, требуется %.4f\n",
		h.config.BaseCurrency, balance.Available, requiredAmount)
	logger.LogPlain("📊 Исходная сделка Freqtrade: %.6f %s по цене %.4f (убыток %.2f%%)\n",
		trade.Amount, pair.String(), trade.OpenRate, trade.ProfitRatio*100)
	logger.LogPlain("🛒 Хеджирующая покупка: %.6f %s на сумму %.2f %s по цене %.4f\n",
		orderQuantity, pair.ToBybitFormat(), adjustedPositionAmount, h.config.BaseCurrency, trade.CurrentRate)

	// 2. Размещаем лимитный ордер на покупку с небольшим запасом по цене
	// Используем лимитный ордер вместо рыночного для лучшего контроля над минимальными лимитами
	limitPrice := trade.CurrentRate * 1.001 // +0.1% запас для гарантированного исполнения

	// Расчет цены для лимитного ордера

	// Округляем цену до правильного шага согласно tickSize от Bybit
	tickSize := instrumentInfo.TickSize
	if tickSize > 0 {
		// Округляем до ближайшего кратного tickSize
		limitPrice = math.Round(limitPrice/tickSize) * tickSize
		logger.LogWithTime("🔧 Цена скорректирована до шага %.8f: %.8f → %.8f", tickSize, trade.CurrentRate*1.001, limitPrice)
	}

	// Объявляем переменную для ордера
	var buyOrder *entities.Order

	// Проверяем, что цена не стала нулевой или слишком маленькой после округления
	// Для очень дешевых активов (цена < 0.0001) используем лимитный ордер с текущей рыночной ценой
	if limitPrice <= 0 || limitPrice < 0.0001 {
		logger.LogWithTime("⚠️ ВНИМАНИЕ: Цена слишком маленькая (%.8f), используем лимитный ордер с текущей рыночной ценой", limitPrice)
		// Для очень дешевых активов используем текущую рыночную цену с небольшим запасом
		marketPrice := trade.CurrentRate * 1.001 // +0.1% запас для гарантированного исполнения
		buyOrder = entities.NewLimitOrder(symbol, entities.OrderSideBuy, orderQuantity, marketPrice)
		logger.LogWithTime("🎯 Лимитный ордер на покупку: %.6f %s по цене %.8f (текущая рыночная +0.1%%)", orderQuantity, pair.ToBybitFormat(), marketPrice)
	} else {
		buyOrder = entities.NewLimitOrder(symbol, entities.OrderSideBuy, orderQuantity, limitPrice)
		logger.LogWithTime("🎯 Лимитный ордер на покупку: %.6f %s по цене %.8f (с запасом +0.1%%)",
			orderQuantity, pair.ToBybitFormat(), limitPrice)
	}

	// Проверка параметров ордера на покупку

	// Проверка на пустые или некорректные значения
	if symbol == "" {
		return fmt.Errorf("символ ордера пустой")
	}
	if buyOrder.Quantity <= 0 {
		return fmt.Errorf("количество ордера должно быть больше 0: %.6f", buyOrder.Quantity)
	}
	// Для рыночных ордеров цена не проверяется (она всегда 0)
	if buyOrder.Type == entities.OrderTypeLimit && buyOrder.Price <= 0 {
		return fmt.Errorf("цена лимитного ордера должна быть больше 0: %.4f", buyOrder.Price)
	}

	// Размещение ордера на покупку

	buyResult, err := h.exchangeService.PlaceOrder(ctx, buyOrder)
	if err != nil {
		return fmt.Errorf("ошибка размещения ордера на покупку: %w", err)
	}

	if !buyResult.Success {
		return fmt.Errorf("неудачное размещение ордера на покупку: %s", buyResult.Error)
	}

	// 3. Ожидаем полного исполнения ордера на покупку с повторными попытками
	logger.LogWithTime("⏳ Ожидание исполнения ордера на покупку...")

	var buyOrderStatus *services.OrderStatusInfo
	maxWaitAttempts := 30 // Максимум 30 попыток (30 секунд)
	waitDelay := time.Second

	for attempt := 1; attempt <= maxWaitAttempts; attempt++ {
		time.Sleep(waitDelay)

		buyOrderStatus, err = h.exchangeService.GetOrderStatus(ctx, buyResult.OrderID, symbol)
		if err != nil {
			logger.LogWithTime("⚠️ Попытка %d/%d получения статуса ордера: %v", attempt, maxWaitAttempts, err)
			continue
		}

		// Проверяем, исполнен ли ордер полностью
		if buyOrderStatus.Status == entities.OrderStatusFilled {
			logger.LogWithTime("✅ Ордер на покупку полностью исполнен!")
			break
		} else if buyOrderStatus.Status == entities.OrderStatusPartiallyFilled {
			logger.LogWithTime("⏳ Частичное исполнение: %v из %v", buyOrderStatus.FilledQty, orderQuantity)
			// Продолжаем ждать полного исполнения
		} else if buyOrderStatus.Status.IsCompleted() && buyOrderStatus.Status != entities.OrderStatusFilled {
			return fmt.Errorf("ордер на покупку завершен неуспешно: %s", buyOrderStatus.Status)
		}

		if attempt == maxWaitAttempts {
			return fmt.Errorf("превышено время ожидания исполнения ордера на покупку (30 секунд)")
		}
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

	// 4. Проверяем баланс XRP перед размещением ордера на продажу
	logger.LogWithTime("🔍 Проверка баланса %s для размещения ордера на продажу...", pair.BaseCurrency())

	// Получаем баланс базовой валюты торговой пары (например, XRP для XRP/USDT)
	baseCurrencyBalance, err := h.exchangeService.GetBalance(ctx, pair.BaseCurrency())
	if err != nil {
		logger.LogWithTime("⚠️ Не удалось получить баланс %s: %v", pair.BaseCurrency(), err)
		logger.LogWithTime("💡 Продолжаем с фактически купленным количеством")
	} else {
		// Проверяем, достаточно ли XRP для продажи
		if baseCurrencyBalance.Available < actualQuantity {
			logger.LogWithTime("⚠️ Недостаточно %s для продажи: доступно %.4f, требуется %.4f",
				pair.BaseCurrency(), baseCurrencyBalance.Available, actualQuantity)
			logger.LogWithTime("💡 Корректируем количество для продажи на доступное")
			actualQuantity = baseCurrencyBalance.Available

			if actualQuantity <= 0 {
				return fmt.Errorf("недостаточно %s для размещения ордера на продажу", pair.BaseCurrency())
			}
		} else {
			logger.LogWithTime("✅ Баланс %s достаточен: доступно %.4f, требуется %.4f",
				pair.BaseCurrency(), baseCurrencyBalance.Available, actualQuantity)
		}
	}

	// 5. Рассчитываем цену тейк-профита
	takeProfitPrice := trade.CalculateTakeProfitPrice(h.config.ProfitRatio)

	logger.LogWithTime("🔍 Расчет цены тейк-профита:")
	logger.LogWithTime("   Исходная цена: %.8f", trade.CurrentRate)
	logger.LogWithTime("   Коэффициент прибыли: %.4f", h.config.ProfitRatio)
	logger.LogWithTime("   Рассчитанная цена тейк-профита: %.8f", takeProfitPrice)

	// Округляем цену тейк-профита до правильного шага согласно tickSize от Bybit
	if tickSize > 0 {
		// Округляем до ближайшего кратного tickSize
		takeProfitPrice = math.Round(takeProfitPrice/tickSize) * tickSize
		logger.LogWithTime("🔧 Цена тейк-профита скорректирована до шага %.8f: %.8f → %.8f", tickSize, trade.CalculateTakeProfitPrice(h.config.ProfitRatio), takeProfitPrice)
	}

	// Проверяем, что цена тейк-профита не стала нулевой
	if takeProfitPrice <= 0 {
		logger.LogWithTime("⚠️ ВНИМАНИЕ: Цена тейк-профита стала нулевой, используем минимальную цену выше текущей")
		// Используем минимальную цену выше текущей для гарантии прибыли
		takeProfitPrice = trade.CurrentRate * 1.001 // +0.1% минимальная прибыль
		logger.LogWithTime("🔧 Цена тейк-профита скорректирована на минимальную прибыль: %.8f", takeProfitPrice)
	}

	logger.LogWithTime("🎯 Лимитный ордер на продажу: %.4f %s по цене %.8f (тейк-профит)",
		actualQuantity, pair.ToBybitFormat(), takeProfitPrice)

	// 6. Размещаем лимитный ордер на продажу с ретраями
	sellOrder := entities.NewLimitOrder(symbol, entities.OrderSideSell, actualQuantity, takeProfitPrice)

	// Проверка параметров ордера на продажу

	// Проверка на пустые или некорректные значения для ордера на продажу
	if sellOrder.Quantity <= 0 {
		return fmt.Errorf("количество ордера на продажу должно быть больше 0: %.6f", sellOrder.Quantity)
	}
	if sellOrder.Price <= 0 {
		return fmt.Errorf("цена ордера на продажу должна быть больше 0: %.8f", sellOrder.Price)
	}

	var sellResult *entities.OrderResult
	maxRetries := h.config.RetryAttempts
	retryDelay := 2 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		logger.LogWithTime("📤 Попытка %d/%d размещения ордера на продажу", attempt, maxRetries)

		sellResult, err = h.exchangeService.PlaceOrder(ctx, sellOrder)
		if err != nil {
			logger.LogWithTime("⚠️ Попытка %d неудачна: %v", attempt, err)
			if attempt < maxRetries {
				logger.LogWithTime("⏳ Ждем %v перед повтором...", retryDelay)
				time.Sleep(retryDelay)
				continue
			}
			return fmt.Errorf("неудачное размещение ордера на продажу после %d попыток: %w", maxRetries, err)
		}

		if sellResult.Success {
			logger.LogWithTime("✅ Ордер на продажу успешно размещен с попытки %d", attempt)
			break
		} else {
			logger.LogWithTime("⚠️ Попытка %d неудачна: %s", attempt, sellResult.Error)
			if attempt < maxRetries {
				logger.LogWithTime("⏳ Ждем %v перед повтором...", retryDelay)
				time.Sleep(retryDelay)
				continue
			}
			return fmt.Errorf("неудачное размещение ордера на продажу после %d попыток: %s", maxRetries, sellResult.Error)
		}
	}

	// 7. Сохраняем полную информацию о хеджировании
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

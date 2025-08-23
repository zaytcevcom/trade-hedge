package entities

import (
	"strconv"
	"time"
)

// Trade представляет торговую сделку из Freqtrade
type Trade struct {
	ID          int     // ID сделки
	Pair        string  // Валютная пара
	IsOpen      bool    // Открыта ли сделка
	ProfitRatio float64 // Текущий коэффициент прибыли/убытка
	CurrentRate float64 // Текущая цена
	OpenRate    float64 // Цена открытия
	Amount      float64 // Количество валюты
}

// HedgedTrade представляет хеджированную сделку в базе данных
type HedgedTrade struct {
	FreqtradeTradeID int       // ID сделки в Freqtrade
	Pair             string    // Валютная пара (например, BTC/USDT)
	HedgeTime        time.Time // Время хеджирования
	BybitOrderID     string    // ID ордера в Bybit

	// Информация об исходной сделке Freqtrade
	FreqtradeOpenPrice   float64 // Цена открытия в Freqtrade
	FreqtradeAmount      float64 // Количество валюты в Freqtrade
	FreqtradeProfitRatio float64 // Коэффициент прибыли/убытка на момент хеджирования

	// Информация о хеджирующей позиции
	HedgeOpenPrice       float64 // Цена открытия хеджирующей позиции
	HedgeAmount          float64 // Количество валюты в хеджирующей позиции
	HedgeTakeProfitPrice float64 // Цена тейк-профита

	// Статус ордера
	OrderStatus     OrderStatus // Текущий статус ордера на Bybit
	LastStatusCheck *time.Time  // Время последней проверки статуса
	ClosePrice      *float64    // Цена закрытия (если исполнен)
	CloseTime       *time.Time  // Время закрытия (если исполнен)
}

// IsActive проверяет, активна ли хеджированная сделка
func (ht *HedgedTrade) IsActive() bool {
	return !ht.OrderStatus.IsCompleted()
}

// CalculateProfit рассчитывает прибыль от хеджирования (если закрыто)
func (ht *HedgedTrade) CalculateProfit() *float64 {
	if ht.ClosePrice == nil {
		return nil // Сделка еще не закрыта
	}

	profit := (*ht.ClosePrice - ht.HedgeOpenPrice) * ht.HedgeAmount
	return &profit
}

// ShouldBeHedged проверяет, нужно ли хеджировать сделку
func (t *Trade) ShouldBeHedged(maxLossPercent float64) bool {
	// ProfitRatio отрицательный при убытке, поэтому сравниваем с отрицательным порогом
	threshold := -(maxLossPercent / 100)
	return t.ProfitRatio < threshold
}

// SortTradesByDrawdown сортирует сделки по максимальной просадке (от большей к меньшей)
// ProfitRatio отрицательный при убытке, поэтому сортируем по возрастанию (от -0.05 к -0.02)
func SortTradesByDrawdown(trades []*Trade) {
	if len(trades) <= 1 {
		return
	}

	// Сортируем по убыванию просадки (от большей к меньшей)
	// Поскольку ProfitRatio отрицательный при убытке, сортируем по возрастанию
	for i := 0; i < len(trades)-1; i++ {
		for j := i + 1; j < len(trades); j++ {
			// Если просадка i-й сделки меньше просадки j-й сделки, меняем местами
			// ProfitRatio отрицательный, поэтому сравниваем наоборот
			if trades[i].ProfitRatio > trades[j].ProfitRatio {
				trades[i], trades[j] = trades[j], trades[i]
			}
		}
	}
}

// CalculateTakeProfitPrice рассчитывает цену тейк-профита
func (t *Trade) CalculateTakeProfitPrice(profitRatio float64) float64 {
	takeProfitPercent := t.ProfitRatio * -100 * profitRatio // убыток в процентах * коэффициент
	rawPrice := t.CurrentRate * (1 + takeProfitPercent/100)

	// Для очень маленьких цен используем 8 знаков, для обычных - 4 знака
	var multiplier float64
	if t.CurrentRate < 0.0001 {
		multiplier = 100000000.0 // 10^8 для 8 знаков
	} else {
		multiplier = 10000.0 // 10^4 для 4 знаков
	}

	roundedPrice := float64(int(rawPrice*multiplier+0.5)) / multiplier

	// Дополнительная проверка - форматируем строку и парсим обратно для гарантии точности
	precision := 8
	if t.CurrentRate >= 0.0001 {
		precision = 4
	}

	priceStr := strconv.FormatFloat(roundedPrice, 'f', precision, 64)
	finalPrice, _ := strconv.ParseFloat(priceStr, 64)

	return finalPrice
}

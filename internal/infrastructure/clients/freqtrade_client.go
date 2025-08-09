package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"trade-hedge/internal/domain/entities"
	"trade-hedge/internal/infrastructure/config"
	"trade-hedge/internal/pkg/logger"
)

// FreqtradeClient клиент для работы с Freqtrade API
type FreqtradeClient struct {
	config *config.FreqtradeConfig
	client *http.Client
}

// FreqtradeTradeResponse ответ от Freqtrade API
type FreqtradeTradeResponse struct {
	TradeID     int     `json:"trade_id"`
	Pair        string  `json:"pair"`
	IsOpen      bool    `json:"is_open"`
	ProfitRatio float64 `json:"profit_ratio"`
	CurrentRate float64 `json:"current_rate"`
	OpenRate    float64 `json:"open_rate"`
	Amount      float64 `json:"amount"`
}

// NewFreqtradeClient создает новый клиент Freqtrade
func NewFreqtradeClient(config *config.FreqtradeConfig) *FreqtradeClient {
	return &FreqtradeClient{
		config: config,
		client: &http.Client{},
	}
}

// GetActiveTrades получает активные сделки из Freqtrade
func (f *FreqtradeClient) GetActiveTrades(ctx context.Context) ([]*entities.Trade, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", f.config.APIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.SetBasicAuth(f.config.Username, f.config.Password)
	req.Header.Add("accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("неверный статус код: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	// Логируем только размер ответа для отладки
	logger.LogWithTime("🔍 Получен ответ от Freqtrade API (%d байт)", len(body))

	// Парсим как прямой массив (endpoint /status)
	var apiTrades []FreqtradeTradeResponse
	if err := json.Unmarshal(body, &apiTrades); err == nil {
		logger.LogWithTime("✅ Найдено сделок: %d", len(apiTrades))
		return f.convertTradesToEntities(apiTrades), nil
	}

	// Пробуем парсить как одиночный объект
	var singleTrade FreqtradeTradeResponse
	if err := json.Unmarshal(body, &singleTrade); err == nil {
		logger.LogWithTime("✅ Найдена 1 сделка как одиночный объект")
		return f.convertTradesToEntities([]FreqtradeTradeResponse{singleTrade}), nil
	}

	return nil, fmt.Errorf("ошибка парсинга JSON ответа Freqtrade: %w", err)
}

// convertTradesToEntities конвертирует API ответы в доменные сущности
func (f *FreqtradeClient) convertTradesToEntities(apiTrades []FreqtradeTradeResponse) []*entities.Trade {
	trades := make([]*entities.Trade, 0, len(apiTrades))
	for _, apiTrade := range apiTrades {
		if apiTrade.IsOpen { // Только открытые сделки
			trade := &entities.Trade{
				ID:          apiTrade.TradeID,
				Pair:        apiTrade.Pair,
				IsOpen:      apiTrade.IsOpen,
				ProfitRatio: apiTrade.ProfitRatio,
				CurrentRate: apiTrade.CurrentRate,
				OpenRate:    apiTrade.OpenRate,
				Amount:      apiTrade.Amount,
			}
			trades = append(trades, trade)
		}
	}
	return trades
}

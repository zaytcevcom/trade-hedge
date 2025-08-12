package webui

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"trade-hedge/internal/domain/entities"
)

// TradeStats статистика по сделкам
type TradeStats struct {
	Total       int     `json:"total"`
	Active      int     `json:"active"`
	Completed   int     `json:"completed"`
	TotalProfit float64 `json:"totalProfit"`
}

// APIResponse универсальный ответ API
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Updated int         `json:"updated,omitempty"`
}

// TradesResponse ответ с данными о сделках
type TradesResponse struct {
	Trades []TradeView `json:"trades"`
	Stats  TradeStats  `json:"stats"`
}

// TradeView представление сделки для веб-интерфейса
type TradeView struct {
	FreqtradeTradeID     int        `json:"freqtrade_trade_id"`
	Pair                 string     `json:"pair"`
	HedgeTime            time.Time  `json:"hedge_time"`
	BybitOrderID         string     `json:"bybit_order_id"`
	FreqtradeOpenPrice   float64    `json:"freqtrade_open_price"`
	FreqtradeAmount      float64    `json:"freqtrade_amount"`
	FreqtradeProfitRatio float64    `json:"freqtrade_profit_ratio"`
	HedgeOpenPrice       float64    `json:"hedge_open_price"`
	HedgeAmount          float64    `json:"hedge_amount"`
	HedgeTakeProfitPrice float64    `json:"hedge_take_profit_price"`
	OrderStatus          string     `json:"order_status"`
	LastStatusCheck      *time.Time `json:"last_status_check"`
	ClosePrice           *float64   `json:"close_price"`
	CloseTime            *time.Time `json:"close_time"`
	Profit               *float64   `json:"profit"`
}

// PageData данные для рендеринга страниц
type PageData struct {
	Title  string
	Config interface{}
}

// handleDashboard главная страница дашборда
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title: "Дашборд",
	}

	// Выполняем layout с dashboard content
	if err := s.executeTemplate(w, "dashboard.html", data); err != nil {
		// Логируем ошибку, но не пытаемся изменить заголовки если они уже отправлены
		log.Printf("❌ Ошибка рендеринга шаблона dashboard.html: %v", err)
		return
	}
}

// handleTrades страница со списком сделок
func (s *Server) handleTrades(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title: "Сделки",
	}

	if err := s.executeTemplate(w, "trades.html", data); err != nil {
		// Логируем ошибку, но не пытаемся изменить заголовки если они уже отправлены
		log.Printf("❌ Ошибка рендеринга шаблона trades.html: %v", err)
		return
	}
}

// handleConfig страница конфигурации
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title:  "Конфигурация",
		Config: s.fullConfig,
	}

	if err := s.executeTemplate(w, "config.html", data); err != nil {
		// Логируем ошибку, но не пытаемся изменить заголовки если они уже отправлены
		log.Printf("❌ Ошибка рендеринга шаблона config.html: %v", err)
		return
	}
}

// executeTemplate выполняет шаблон с layout безопасно
func (s *Server) executeTemplate(w http.ResponseWriter, templateName string, data interface{}) error {
	// Рендерим в буфер сначала чтобы поймать ошибки до отправки заголовков
	var buf bytes.Buffer
	if err := s.templates.ExecuteTemplate(&buf, "layout.html", data); err != nil {
		return err
	}

	// Если рендеринг успешен, отправляем результат
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := w.Write(buf.Bytes())
	return err
}

// handleAPITrades API для получения данных о сделках
func (s *Server) handleAPITrades(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Получаем параметры фильтрации
	statusParam := r.URL.Query().Get("status")

	var status *string
	if statusParam != "" {
		status = &statusParam
	}

	// Используем универсальный метод для получения сделок
	trades, err := s.hedgeRepo.GetHedgedTrades(ctx, status)
	if err != nil {
		s.sendError(w, "Ошибка получения сделок", http.StatusInternalServerError)
		return
	}

	// Преобразуем в представление для веб-интерфейса
	tradeViews := s.convertToTradeViews(trades)

	// Рассчитываем статистику
	stats := s.calculateStats(trades)

	response := TradesResponse{
		Trades: tradeViews,
		Stats:  stats,
	}

	s.sendJSON(w, response)
}

// handleAPIStatus API для получения статуса системы
func (s *Server) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"database":  "connected",
		"freqtrade": "connected",
		"bybit":     "connected",
		"webui":     "running",
		"lastCheck": time.Now(),
	}

	s.sendJSON(w, APIResponse{
		Success: true,
		Data:    status,
	})
}

// handleAPIExecute API для выполнения стратегии хеджирования
func (s *Server) handleAPIExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	err := s.hedgeUseCase.ExecuteHedgeStrategy(ctx)
	if err != nil {
		s.sendJSON(w, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	s.sendJSON(w, APIResponse{
		Success: true,
		Message: "Стратегия хеджирования выполнена успешно",
	})
}

// handleAPICheckStatus API для проверки статусов ордеров
func (s *Server) handleAPICheckStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Получаем количество активных ордеров до проверки
	pendingStatus := "PENDING"
	activeBefore, _ := s.hedgeRepo.GetHedgedTrades(ctx, &pendingStatus)
	beforeCount := len(activeBefore)

	err := s.statusCheckerUseCase.CheckAllActiveOrders(ctx)
	if err != nil {
		s.sendJSON(w, APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	// Получаем количество активных ордеров после проверки
	activeAfter, _ := s.hedgeRepo.GetHedgedTrades(ctx, &pendingStatus)
	afterCount := len(activeAfter)

	// Количество обновленных ордеров = количество закрытых
	updated := beforeCount - afterCount
	if updated < 0 {
		updated = 0
	}

	s.sendJSON(w, APIResponse{
		Success: true,
		Message: "Статусы ордеров проверены",
		Updated: updated,
	})
}

// getAllTrades получает все сделки (включая закрытые)
func (s *Server) getAllTrades(ctx context.Context) []*entities.HedgedTrade {
	// Получаем все сделки включая закрытые
	trades, err := s.hedgeRepo.GetHedgedTrades(ctx, nil) // nil = все сделки
	if err != nil {
		// В случае ошибки возвращаем пустой массив
		return []*entities.HedgedTrade{}
	}
	return trades
}

// convertToTradeViews преобразует сделки в представление для веб-интерфейса
func (s *Server) convertToTradeViews(trades []*entities.HedgedTrade) []TradeView {
	views := make([]TradeView, len(trades))

	for i, trade := range trades {
		view := TradeView{
			FreqtradeTradeID:     trade.FreqtradeTradeID,
			Pair:                 trade.Pair,
			HedgeTime:            trade.HedgeTime,
			BybitOrderID:         trade.BybitOrderID,
			FreqtradeOpenPrice:   trade.FreqtradeOpenPrice,
			FreqtradeAmount:      trade.FreqtradeAmount,
			FreqtradeProfitRatio: trade.FreqtradeProfitRatio,
			HedgeOpenPrice:       trade.HedgeOpenPrice,
			HedgeAmount:          trade.HedgeAmount,
			HedgeTakeProfitPrice: trade.HedgeTakeProfitPrice,
			OrderStatus:          trade.OrderStatus.String(),
			LastStatusCheck:      trade.LastStatusCheck,
			ClosePrice:           trade.ClosePrice,
			CloseTime:            trade.CloseTime,
		}

		// Рассчитываем прибыль, если ордер закрыт
		if profit := trade.CalculateProfit(); profit != nil {
			view.Profit = profit
		}

		views[i] = view
	}

	return views
}

// calculateStats рассчитывает статистику по сделкам
func (s *Server) calculateStats(trades []*entities.HedgedTrade) TradeStats {
	stats := TradeStats{
		Total: len(trades),
	}

	for _, trade := range trades {
		if trade.IsActive() {
			stats.Active++
		} else {
			stats.Completed++
			if profit := trade.CalculateProfit(); profit != nil {
				stats.TotalProfit += *profit
			}
		}
	}

	return stats
}

// sendJSON отправляет JSON ответ
func (s *Server) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Ошибка кодирования JSON", http.StatusInternalServerError)
	}
}

// sendError отправляет ошибку в JSON формате
func (s *Server) sendError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	response := APIResponse{
		Success: false,
		Message: message,
	}

	json.NewEncoder(w).Encode(response)
}

package clients

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
	"trade-hedge/internal/domain/entities"
	"trade-hedge/internal/domain/services"
	"trade-hedge/internal/infrastructure/config"
)

// BybitClient клиент для работы с Bybit API
type BybitClient struct {
	config *config.BybitConfig
	client *http.Client
}

// BybitOrderResponse ответ от Bybit API
type BybitOrderResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		OrderID string `json:"orderId"`
	} `json:"result"`
}

// BybitErrorResponse ошибка от Bybit API
type BybitErrorResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
}

// BybitBalanceResponse ответ от Bybit UNIFIED API с балансом
type BybitBalanceResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			AccountType           string `json:"accountType"`
			TotalEquity           string `json:"totalEquity"`
			TotalWalletBalance    string `json:"totalWalletBalance"`
			TotalAvailableBalance string `json:"totalAvailableBalance"`
			Coin                  []struct {
				Coin                string `json:"coin"`
				WalletBalance       string `json:"walletBalance"`
				AvailableToWithdraw string `json:"availableToWithdraw"`
				Equity              string `json:"equity"`
				UsdValue            string `json:"usdValue"`
			} `json:"coin"`
		} `json:"list"`
	} `json:"result"`
}

// BybitOrderStatusResponse ответ от Bybit API со статусом ордера
type BybitOrderStatusResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			OrderID     string `json:"orderId"`
			Symbol      string `json:"symbol"`
			OrderStatus string `json:"orderStatus"`
			Side        string `json:"side"`
			OrderType   string `json:"orderType"`
			Price       string `json:"price"`
			Qty         string `json:"qty"`
			CumExecQty  string `json:"cumExecQty"`
			LeavesQty   string `json:"leavesQty"`
			AvgPrice    string `json:"avgPrice"`
			CreatedTime string `json:"createdTime"`
			UpdatedTime string `json:"updatedTime"`
		} `json:"list"`
	} `json:"result"`
}

// NewBybitClient создает новый клиент Bybit
func NewBybitClient(config *config.BybitConfig) *BybitClient {
	return &BybitClient{
		config: config,
		client: &http.Client{},
	}
}

// PlaceOrder размещает ордер на Bybit
func (b *BybitClient) PlaceOrder(ctx context.Context, order *entities.Order) (*entities.OrderResult, error) {
	timestamp := time.Now().UnixMilli()
	recvWindow := "5000"

	params := map[string]interface{}{
		"category":    "spot", // Обязательно для V5 API
		"symbol":      order.Symbol,
		"side":        string(order.Side),
		"orderType":   string(order.Type), // В V5 API это orderType, не type
		"qty":         strconv.FormatFloat(order.Quantity, 'f', 6, 64),
		"timeInForce": "GTC",
	}

	// Для лимитных ордеров добавляем цену
	if order.Type == entities.OrderTypeLimit {
		params["price"] = strconv.FormatFloat(order.Price, 'f', 6, 64)
	}

	paramStr, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации параметров: %w", err)
	}

	// Генерация подписи
	signature := hmac.New(sha256.New, []byte(b.config.APISecret))
	signature.Write([]byte(fmt.Sprintf("%d%s%s%s", timestamp, b.config.APIKey, recvWindow, paramStr)))
	sign := hex.EncodeToString(signature.Sum(nil))

	// Создание запроса (без category в URL для V5 API)
	reqBody, _ := json.Marshal(params)
	req, err := http.NewRequestWithContext(ctx, "POST", b.config.SpotURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Add("X-BAPI-API-KEY", b.config.APIKey)
	req.Header.Add("X-BAPI-SIGN", sign)
	req.Header.Add("X-BAPI-SIGN-TYPE", "2")
	req.Header.Add("X-BAPI-TIMESTAMP", fmt.Sprintf("%d", timestamp))
	req.Header.Add("X-BAPI-RECV-WINDOW", recvWindow)
	req.Header.Add("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	// Проверка на ошибку
	var errResp BybitErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.RetCode != 0 {
		return &entities.OrderResult{
			Success: false,
			Error:   fmt.Sprintf("ошибка Bybit: %s (код: %d)", errResp.RetMsg, errResp.RetCode),
		}, nil
	}

	// Парсинг успешного ответа
	var result BybitOrderResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	return &entities.OrderResult{
		OrderID: result.Result.OrderID,
		Success: true,
		Error:   "",
	}, nil
}

// GetBalance получает баланс по указанной валюте
func (b *BybitClient) GetBalance(ctx context.Context, asset string) (*entities.Balance, error) {
	timestamp := time.Now().UnixMilli()
	recvWindow := "5000"

	// Создаем параметры запроса (используем UNIFIED аккаунт)
	params := fmt.Sprintf("accountType=UNIFIED&coin=%s", asset)

	// Генерация подписи для GET запроса
	signature := hmac.New(sha256.New, []byte(b.config.APISecret))
	signature.Write([]byte(fmt.Sprintf("%d%s%s%s", timestamp, b.config.APIKey, recvWindow, params)))
	sign := hex.EncodeToString(signature.Sum(nil))

	// Создание запроса
	url := fmt.Sprintf("%s?%s", b.config.BalanceURL, params)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Add("X-BAPI-API-KEY", b.config.APIKey)
	req.Header.Add("X-BAPI-SIGN", sign)
	req.Header.Add("X-BAPI-TIMESTAMP", fmt.Sprintf("%d", timestamp))
	req.Header.Add("X-BAPI-RECV-WINDOW", recvWindow)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	// Логируем размер ответа для отладки
	fmt.Printf("🔍 Bybit Balance API ответ (%d байт)\n", len(body))

	// Проверка на ошибку
	var errResp BybitErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.RetCode != 0 {
		return nil, fmt.Errorf("ошибка Bybit: %s (код: %d)", errResp.RetMsg, errResp.RetCode)
	}

	// Парсинг успешного ответа
	var result BybitBalanceResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	// Поиск баланса нужной валюты в UNIFIED account
	for _, account := range result.Result.List {
		for _, coinBalance := range account.Coin {
			if strings.EqualFold(coinBalance.Coin, asset) {
				walletBalance, _ := strconv.ParseFloat(coinBalance.WalletBalance, 64)
				availableBalance, _ := strconv.ParseFloat(coinBalance.AvailableToWithdraw, 64)

				// Если AvailableToWithdraw пустой, используем WalletBalance
				if coinBalance.AvailableToWithdraw == "" {
					availableBalance = walletBalance
				}

				return &entities.Balance{
					Asset:     asset,
					Available: availableBalance, // Доступный для вывода/торговли
					Total:     walletBalance,    // Общий баланс кошелька
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("валюта %s не найдена в балансе UNIFIED аккаунта", asset)
}

// GetOrderStatus получает статус ордера по ID
func (b *BybitClient) GetOrderStatus(ctx context.Context, orderID, symbol string) (*services.OrderStatusInfo, error) {
	timestamp := time.Now().UnixMilli()
	recvWindow := "5000"

	// Создаем параметры запроса
	params := fmt.Sprintf("category=spot&orderId=%s", orderID)

	// Генерация подписи для GET запроса
	signature := hmac.New(sha256.New, []byte(b.config.APISecret))
	signature.Write([]byte(fmt.Sprintf("%d%s%s%s", timestamp, b.config.APIKey, recvWindow, params)))
	sign := hex.EncodeToString(signature.Sum(nil))

	// Создание запроса
	url := fmt.Sprintf("%s?%s", b.config.OrderStatusURL, params)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Add("X-BAPI-API-KEY", b.config.APIKey)
	req.Header.Add("X-BAPI-SIGN", sign)
	req.Header.Add("X-BAPI-TIMESTAMP", fmt.Sprintf("%d", timestamp))
	req.Header.Add("X-BAPI-RECV-WINDOW", recvWindow)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	// Проверка на ошибку
	var errResp BybitErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.RetCode != 0 {
		return nil, fmt.Errorf("ошибка Bybit: %s (код: %d)", errResp.RetMsg, errResp.RetCode)
	}

	// Парсинг успешного ответа
	var result BybitOrderStatusResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	if len(result.Result.List) == 0 {
		return nil, fmt.Errorf("ордер %s не найден", orderID)
	}

	orderData := result.Result.List[0]

	// Конвертируем статус Bybit в наш enum
	status := entities.OrderStatusFromString(orderData.OrderStatus)

	// Парсим численные значения
	filledQty, _ := strconv.ParseFloat(orderData.CumExecQty, 64)
	remainingQty, _ := strconv.ParseFloat(orderData.LeavesQty, 64)

	statusInfo := &services.OrderStatusInfo{
		OrderID:      orderData.OrderID,
		Status:       status,
		FilledQty:    filledQty,
		RemainingQty: remainingQty,
	}

	// Если ордер исполнен, добавляем информацию о цене и времени
	if status == entities.OrderStatusFilled && orderData.AvgPrice != "" {
		avgPrice, err := strconv.ParseFloat(orderData.AvgPrice, 64)
		if err == nil {
			statusInfo.FilledPrice = &avgPrice
		}

		// Парсим время обновления как время исполнения
		if orderData.UpdatedTime != "" {
			if updatedTimeMs, err := strconv.ParseInt(orderData.UpdatedTime, 10, 64); err == nil {
				filledTime := time.UnixMilli(updatedTimeMs)
				statusInfo.FilledTime = &filledTime
			}
		}
	}

	return statusInfo, nil
}

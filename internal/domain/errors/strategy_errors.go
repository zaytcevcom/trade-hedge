package errors

import "fmt"

// StrategyError базовый тип для ошибок стратегии
type StrategyError struct {
	Type    ErrorType
	Message string
}

// ErrorType тип ошибки стратегии
type ErrorType int

const (
	// ErrorTypeNoTrades нет сделок для хеджирования
	ErrorTypeNoTrades ErrorType = iota
	// ErrorTypeNoLossyTrades нет убыточных сделок
	ErrorTypeNoLossyTrades
	// ErrorTypeInsufficientBalance недостаточно средств
	ErrorTypeInsufficientBalance
	// ErrorTypeInsufficientBalanceForMinLimit недостаточно средств для минимального лимита
	ErrorTypeInsufficientBalanceForMinLimit
	// ErrorTypeExchangeError ошибка биржи
	ErrorTypeExchangeError
)

// Error реализует интерфейс error
func (e *StrategyError) Error() string {
	return e.Message
}

// IsExpected проверяет, является ли ошибка ожидаемой (не критической)
func (e *StrategyError) IsExpected() bool {
	return e.Type == ErrorTypeNoTrades ||
		e.Type == ErrorTypeNoLossyTrades ||
		e.Type == ErrorTypeInsufficientBalanceForMinLimit
}

// NewNoTradesError создает ошибку "нет сделок"
func NewNoTradesError() *StrategyError {
	return &StrategyError{
		Type:    ErrorTypeNoTrades,
		Message: "Нет сделок для хеджирования",
	}
}

// NewNoLossyTradesError создает ошибку "нет убыточных сделок"
func NewNoLossyTradesError(maxLossPercent float64) *StrategyError {
	return &StrategyError{
		Type:    ErrorTypeNoLossyTrades,
		Message: fmt.Sprintf("Нет сделок с убытком > %.2f%%", maxLossPercent),
	}
}

// NewInsufficientBalanceError создает ошибку недостатка средств
func NewInsufficientBalanceError(required, available float64, currency string) *StrategyError {
	return &StrategyError{
		Type:    ErrorTypeInsufficientBalance,
		Message: fmt.Sprintf("Недостаточно средств для покупки: нужно %.4f %s, доступно %.4f %s", required, currency, available, currency),
	}
}

// NewInsufficientBalanceForMinLimitError создает ошибку недостатка средств для минимального лимита
func NewInsufficientBalanceForMinLimitError(minLimit, available float64, currency string) *StrategyError {
	return &StrategyError{
		Type:    ErrorTypeInsufficientBalanceForMinLimit,
		Message: fmt.Sprintf("Недостаточно средств для минимального лимита ордера: требуется %.2f %s, доступно %.2f %s", minLimit, currency, available, currency),
	}
}

// NewExchangeError создает ошибку биржи
func NewExchangeError(message string) *StrategyError {
	return &StrategyError{
		Type:    ErrorTypeExchangeError,
		Message: fmt.Sprintf("Ошибка биржи: %s", message),
	}
}

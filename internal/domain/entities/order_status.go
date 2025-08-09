package entities

// OrderStatus представляет статус ордера
type OrderStatus string

const (
	// OrderStatusPending ордер размещен, но не исполнен
	OrderStatusPending OrderStatus = "PENDING"

	// OrderStatusFilled ордер полностью исполнен
	OrderStatusFilled OrderStatus = "FILLED"

	// OrderStatusPartiallyFilled ордер частично исполнен
	OrderStatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"

	// OrderStatusCancelled ордер отменен
	OrderStatusCancelled OrderStatus = "CANCELLED"

	// OrderStatusRejected ордер отклонен
	OrderStatusRejected OrderStatus = "REJECTED"

	// OrderStatusUnknown неизвестный статус
	OrderStatusUnknown OrderStatus = "UNKNOWN"
)

// IsCompleted проверяет, завершен ли ордер (успешно или неуспешно)
func (s OrderStatus) IsCompleted() bool {
	return s == OrderStatusFilled ||
		s == OrderStatusCancelled ||
		s == OrderStatusRejected
}

// IsSuccessful проверяет, успешно ли исполнен ордер
func (s OrderStatus) IsSuccessful() bool {
	return s == OrderStatusFilled
}

// String возвращает строковое представление статуса
func (s OrderStatus) String() string {
	return string(s)
}

// FromString создает OrderStatus из строки
func OrderStatusFromString(status string) OrderStatus {
	switch status {
	case "PENDING", "NEW", "New", "OPEN", "Open":
		return OrderStatusPending
	case "FILLED", "Filled", "CLOSED", "Closed":
		return OrderStatusFilled
	case "PARTIALLY_FILLED", "PartiallyFilled", "PARTIAL", "Partial":
		return OrderStatusPartiallyFilled
	case "CANCELLED", "Cancelled", "CANCELED", "Canceled":
		return OrderStatusCancelled
	case "REJECTED", "Rejected":
		return OrderStatusRejected
	default:
		return OrderStatusUnknown
	}
}

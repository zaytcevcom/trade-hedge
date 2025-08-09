package database

import (
	"context"
	"fmt"
	"time"
	"trade-hedge/internal/domain/entities"
	"trade-hedge/internal/infrastructure/config"

	"github.com/jackc/pgx/v4/pgxpool"
)

// PostgreSQLTradeRepository реализует репозиторий для работы с PostgreSQL
type PostgreSQLTradeRepository struct {
	pool *pgxpool.Pool
}

// NewPostgreSQLTradeRepository создает новый экземпляр репозитория
func NewPostgreSQLTradeRepository(config *config.Config) (*PostgreSQLTradeRepository, error) {
	pool, err := pgxpool.Connect(context.Background(), config.GetDatabaseConnectionString())
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к PostgreSQL: %w", err)
	}

	repo := &PostgreSQLTradeRepository{pool: pool}

	// Инициализируем таблицы
	if err := repo.initTables(); err != nil {
		return nil, fmt.Errorf("ошибка инициализации таблиц: %w", err)
	}

	return repo, nil
}

// Close закрывает соединение с базой данных
func (r *PostgreSQLTradeRepository) Close() {
	r.pool.Close()
}

// initTables создает необходимые таблицы
func (r *PostgreSQLTradeRepository) initTables() error {
	// Создаем новую таблицу с расширенной информацией
	query := `
		CREATE TABLE IF NOT EXISTS hedged_trades (
			freqtrade_trade_id INTEGER PRIMARY KEY,
			pair TEXT NOT NULL,
			hedge_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			bybit_order_id TEXT,
			
			-- Информация об исходной сделке Freqtrade
			freqtrade_open_price FLOAT NOT NULL,
			freqtrade_amount FLOAT NOT NULL,
			freqtrade_profit_ratio FLOAT NOT NULL,
			
			-- Информация о хеджирующей позиции
			hedge_open_price FLOAT NOT NULL,
			hedge_amount FLOAT NOT NULL,
			hedge_take_profit_price FLOAT NOT NULL
		)`

	_, err := r.pool.Exec(context.Background(), query)
	if err != nil {
		return err
	}

	// Добавляем новые колонки к существующей таблице (для совместимости)
	alterQueries := []string{
		"ALTER TABLE hedged_trades ADD COLUMN IF NOT EXISTS freqtrade_open_price FLOAT",
		"ALTER TABLE hedged_trades ADD COLUMN IF NOT EXISTS freqtrade_amount FLOAT",
		"ALTER TABLE hedged_trades ADD COLUMN IF NOT EXISTS freqtrade_profit_ratio FLOAT",
		"ALTER TABLE hedged_trades ADD COLUMN IF NOT EXISTS hedge_open_price FLOAT",
		"ALTER TABLE hedged_trades ADD COLUMN IF NOT EXISTS hedge_amount FLOAT",
		"ALTER TABLE hedged_trades ADD COLUMN IF NOT EXISTS hedge_take_profit_price FLOAT",
		"ALTER TABLE hedged_trades ADD COLUMN IF NOT EXISTS order_status TEXT DEFAULT 'PENDING'",
		"ALTER TABLE hedged_trades ADD COLUMN IF NOT EXISTS last_status_check TIMESTAMP",
		"ALTER TABLE hedged_trades ADD COLUMN IF NOT EXISTS close_price FLOAT",
		"ALTER TABLE hedged_trades ADD COLUMN IF NOT EXISTS close_time TIMESTAMP",
	}

	for _, alterQuery := range alterQueries {
		_, err = r.pool.Exec(context.Background(), alterQuery)
		if err != nil {
			// Игнорируем ошибки добавления колонок (они могут уже существовать)
			continue
		}
	}

	return nil
}

// IsTradeHedged проверяет, была ли сделка хеджирована
func (r *PostgreSQLTradeRepository) IsTradeHedged(ctx context.Context, tradeID int) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM hedged_trades WHERE freqtrade_trade_id = $1",
		tradeID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("ошибка проверки хеджирования: %w", err)
	}
	return count > 0, nil
}

// SaveHedgedTrade сохраняет информацию о хеджированной сделке
func (r *PostgreSQLTradeRepository) SaveHedgedTrade(ctx context.Context, hedgedTrade *entities.HedgedTrade) error {
	query := `
		INSERT INTO hedged_trades 
		(freqtrade_trade_id, pair, bybit_order_id, hedge_time,
		 freqtrade_open_price, freqtrade_amount, freqtrade_profit_ratio,
		 hedge_open_price, hedge_amount, hedge_take_profit_price,
		 order_status, last_status_check, close_price, close_time) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := r.pool.Exec(ctx, query,
		hedgedTrade.FreqtradeTradeID,
		hedgedTrade.Pair,
		hedgedTrade.BybitOrderID,
		hedgedTrade.HedgeTime,
		hedgedTrade.FreqtradeOpenPrice,
		hedgedTrade.FreqtradeAmount,
		hedgedTrade.FreqtradeProfitRatio,
		hedgedTrade.HedgeOpenPrice,
		hedgedTrade.HedgeAmount,
		hedgedTrade.HedgeTakeProfitPrice,
		hedgedTrade.OrderStatus.String(),
		hedgedTrade.LastStatusCheck,
		hedgedTrade.ClosePrice,
		hedgedTrade.CloseTime)

	if err != nil {
		return fmt.Errorf("ошибка сохранения хеджированной сделки: %w", err)
	}

	return nil
}

// GetActiveHedgedTrades получает все активные хеджированные сделки
func (r *PostgreSQLTradeRepository) GetActiveHedgedTrades(ctx context.Context) ([]*entities.HedgedTrade, error) {
	query := `
		SELECT freqtrade_trade_id, pair, bybit_order_id, hedge_time,
			   freqtrade_open_price, freqtrade_amount, freqtrade_profit_ratio,
			   hedge_open_price, hedge_amount, hedge_take_profit_price,
			   order_status, last_status_check, close_price, close_time
		FROM hedged_trades 
		WHERE order_status NOT IN ('FILLED', 'CANCELLED', 'REJECTED')
		ORDER BY hedge_time DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения активных хеджированных сделок: %w", err)
	}
	defer rows.Close()

	var hedgedTrades []*entities.HedgedTrade
	for rows.Next() {
		trade := &entities.HedgedTrade{}
		var orderStatusStr string

		err := rows.Scan(
			&trade.FreqtradeTradeID,
			&trade.Pair,
			&trade.BybitOrderID,
			&trade.HedgeTime,
			&trade.FreqtradeOpenPrice,
			&trade.FreqtradeAmount,
			&trade.FreqtradeProfitRatio,
			&trade.HedgeOpenPrice,
			&trade.HedgeAmount,
			&trade.HedgeTakeProfitPrice,
			&orderStatusStr,
			&trade.LastStatusCheck,
			&trade.ClosePrice,
			&trade.CloseTime)

		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования хеджированной сделки: %w", err)
		}

		trade.OrderStatus = entities.OrderStatusFromString(orderStatusStr)
		hedgedTrades = append(hedgedTrades, trade)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по результатам: %w", err)
	}

	return hedgedTrades, nil
}

// UpdateHedgedTradeStatus обновляет статус хеджированной сделки
func (r *PostgreSQLTradeRepository) UpdateHedgedTradeStatus(ctx context.Context, orderID string, status entities.OrderStatus, closePrice *float64, closeTime *time.Time) error {
	query := `
		UPDATE hedged_trades 
		SET order_status = $1, last_status_check = $2, close_price = $3, close_time = $4
		WHERE bybit_order_id = $5`

	now := time.Now()
	_, err := r.pool.Exec(ctx, query, status.String(), now, closePrice, closeTime, orderID)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса хеджированной сделки: %w", err)
	}

	return nil
}

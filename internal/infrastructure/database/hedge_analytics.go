package database

import (
	"context"
	"fmt"
	"trade-hedge/internal/domain/entities"
)

// GetHedgedTradesAnalytics получает аналитику по хеджированным сделкам
func (r *PostgreSQLTradeRepository) GetHedgedTradesAnalytics(ctx context.Context) ([]*entities.HedgedTrade, error) {
	query := `
		SELECT 
			freqtrade_trade_id, pair, hedge_time, bybit_order_id,
			freqtrade_open_price, freqtrade_amount, freqtrade_profit_ratio,
			hedge_open_price, hedge_amount, hedge_take_profit_price
		FROM hedged_trades 
		ORDER BY hedge_time DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения аналитики: %w", err)
	}
	defer rows.Close()

	var hedgedTrades []*entities.HedgedTrade

	for rows.Next() {
		trade := &entities.HedgedTrade{}
		err := rows.Scan(
			&trade.FreqtradeTradeID,
			&trade.Pair,
			&trade.HedgeTime,
			&trade.BybitOrderID,
			&trade.FreqtradeOpenPrice,
			&trade.FreqtradeAmount,
			&trade.FreqtradeProfitRatio,
			&trade.HedgeOpenPrice,
			&trade.HedgeAmount,
			&trade.HedgeTakeProfitPrice,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
		}

		hedgedTrades = append(hedgedTrades, trade)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по строкам: %w", err)
	}

	return hedgedTrades, nil
}

// GetHedgedTradesCount получает количество хеджированных сделок
func (r *PostgreSQLTradeRepository) GetHedgedTradesCount(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM hedged_trades").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения количества хеджированных сделок: %w", err)
	}
	return count, nil
}

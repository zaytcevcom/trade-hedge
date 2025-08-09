# Trade Hedge API Documentation

## 🔌 Web API Endpoints

Trade Hedge предоставляет REST API для мониторинга и управления системой хеджирования.

### 📊 Статус системы

#### `GET /api/status`

Получение текущего статуса системы.

**Ответ:**
```json
{
  "status": "running",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "uptime": "2h45m30s"
}
```

#### `GET /health`

Health check endpoint для мониторинга.

**Ответ:**
```json
{
  "status": "healthy"
}
```

### 📈 Торговые данные

#### `GET /api/trades`

Получение списка всех хеджированных сделок.

**Параметры запроса:**
- `limit` (int, optional) - Лимит записей (по умолчанию: 50)
- `offset` (int, optional) - Смещение (по умолчанию: 0)
- `status` (string, optional) - Фильтр по статусу (PENDING, FILLED, CANCELLED, REJECTED)
- `pair` (string, optional) - Фильтр по валютной паре

**Пример запроса:**
```bash
curl "http://localhost:8081/api/trades?limit=10&status=FILLED"
```

**Ответ:**
```json
{
  "trades": [
    {
      "freqtrade_trade_id": 12345,
      "pair": "BTC/USDT",
      "hedge_time": "2024-01-15T10:25:00Z",
      "bybit_order_id": "ord-123456",
      "freqtrade_open_price": 42000.0,
      "freqtrade_amount": 0.001,
      "freqtrade_profit_ratio": -0.05,
      "hedge_open_price": 41900.0,
      "hedge_amount": 0.001,
      "hedge_take_profit_price": 42100.0,
      "order_status": "FILLED",
      "last_status_check": "2024-01-15T10:30:00Z",
      "close_price": 42050.0,
      "close_time": "2024-01-15T10:35:00Z"
    }
  ],
  "total": 1,
  "limit": 10,
  "offset": 0
}
```

#### `GET /api/trades/stats`

Получение статистики по хеджированным сделкам.

**Ответ:**
```json
{
  "total_trades": 150,
  "active_trades": 5,
  "completed_trades": 145,
  "total_profit": 2500.75,
  "success_rate": 0.85,
  "avg_profit_per_trade": 16.67,
  "stats_by_status": {
    "PENDING": 3,
    "FILLED": 140,
    "CANCELLED": 5,
    "REJECTED": 2
  }
}
```

### ⚙️ Конфигурация

#### `GET /api/config`

Получение текущей конфигурации системы (без секретных данных).

**Ответ:**
```json
{
  "strategy": {
    "position_amount": 100.0,
    "max_loss_percent": 5.0,
    "profit_ratio": 0.02,
    "base_currency": "USDT",
    "check_interval": 60
  },
  "webui": {
    "enabled": true,
    "host": "0.0.0.0",
    "port": 8081
  },
  "database": {
    "max_connections": 10,
    "connection_timeout": 30
  }
}
```

### 🔄 Управление

#### `POST /api/hedge/manual`

Запуск ручного хеджирования (для тестирования).

**Тело запроса:**
```json
{
  "trade_id": 12345
}
```

**Ответ:**
```json
{
  "success": true,
  "message": "Хеджирование успешно выполнено",
  "order_id": "ord-789012"
}
```

#### `POST /api/status/check`

Принудительная проверка статусов всех активных ордеров.

**Ответ:**
```json
{
  "checked_orders": 5,
  "updated_orders": 2,
  "errors": 0,
  "message": "Проверка статусов завершена"
}
```

## 🔒 Безопасность

### Аутентификация

В текущей версии API не требует аутентификации. Для production использования рекомендуется:

1. Использовать reverse proxy (nginx) с базовой аутентификацией
2. Настроить firewall для ограничения доступа
3. Использовать HTTPS

### Пример nginx конфигурации:

```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    location / {
        auth_basic "Trade Hedge";
        auth_basic_user_file /etc/nginx/.htpasswd;
        proxy_pass http://localhost:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 📝 Коды ошибок

| Код | Описание |
|-----|----------|
| 200 | Успешный запрос |
| 400 | Неверные параметры запроса |
| 404 | Ресурс не найден |
| 500 | Внутренняя ошибка сервера |

## 🔧 Примеры использования

### Мониторинг активных сделок

```bash
#!/bin/bash
# Скрипт для мониторинга активных сделок

API_URL="http://localhost:8081"

# Получаем статистику
echo "📊 Статистика системы:"
curl -s "${API_URL}/api/trades/stats" | jq .

echo -e "\n📈 Активные сделки:"
curl -s "${API_URL}/api/trades?status=PENDING" | jq '.trades[]'

echo -e "\n⚡ Статус системы:"
curl -s "${API_URL}/api/status" | jq .
```

### Автоматическая проверка здоровья

```bash
#!/bin/bash
# Health check скрипт для мониторинга

if curl -f -s http://localhost:8081/health > /dev/null; then
    echo "✅ Trade Hedge работает нормально"
    exit 0
else
    echo "❌ Trade Hedge недоступен"
    exit 1
fi
```

---

> Для получения более подробной информации см. [README.md](README.md)

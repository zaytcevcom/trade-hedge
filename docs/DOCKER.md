# 🐳 Docker развертывание Trade Hedge

Полное руководство по развертыванию системы хеджирования с помощью Docker.

## 🚀 Быстрый старт

### 1. Подготовка

```bash
# Клонируйте репозиторий
git clone <repository-url>
cd trade-hedge

# Создайте конфигурацию из примера
cp config/env.example .env

# Отредактируйте .env - заполните API ключи!
nano .env
```

### 2. Запуск

```bash
# Запуск полного стека (приложение + PostgreSQL)
docker compose up -d

# Проверка статуса
docker compose ps

# Просмотр логов
docker compose logs -f trade-hedge
```

### 3. Доступ

- **Веб-интерфейс**: http://localhost:8081
- **База данных**: localhost:5432 (логин: postgres)
- **Adminer** (с профилем tools): http://localhost:8080

## 📋 Профили запуска

### Базовый стек
```bash
docker compose up -d
```
Включает: приложение + PostgreSQL

### С инструментами разработки
```bash
docker compose --profile tools up -d
```
Добавляет: Adminer для управления БД

## ⚙️ Конфигурация

### Обязательные переменные

В файле `.env` **обязательно** заполните:

```bash
# Freqtrade API
FREQTRADE_API_URL=http://your-freqtrade:8080/api/v1/trades
FREQTRADE_USERNAME=your_username
FREQTRADE_PASSWORD=your_password

# Bybit API
BYBIT_API_KEY=your_api_key
BYBIT_API_SECRET=your_secret
```

### Опциональные переменные

```bash
# Стратегия
STRATEGY_POSITION_AMOUNT=100.0
STRATEGY_MAX_LOSS_PERCENT=3.0
STRATEGY_CHECK_INTERVAL=300

# База данных
DB_PASSWORD=secure_password

# Веб-интерфейс
WEBUI_ENABLED=true
WEBUI_PORT=8081
```

## 🔧 Команды управления

### Основные
```bash
# Запуск
docker-compose up -d

# Остановка
docker-compose down

# Перезапуск
docker-compose restart

# Просмотр логов
docker-compose logs -f

# Статус сервисов
docker-compose ps
```

### Обновление
```bash
# Пересборка образа
docker-compose build trade-hedge

# Обновление с пересборкой
docker-compose up -d --build
```

### Очистка
```bash
# Остановка с удалением volumes (ОСТОРОЖНО!)
docker-compose down -v

# Полная очистка
docker system prune -f
```

## 🛠 Makefile команды

Для удобства используйте Makefile:

```bash
# Быстрый старт
make quickstart

# Docker команды
make docker-up          # Запуск
make docker-down        # Остановка
make docker-logs        # Логи


# Утилиты
make backup-db         # Бэкап базы данных
make status           # Статус контейнеров
```

## 📁 Структура volumes

```
trade-hedge/
├── postgres_data/     # Данные PostgreSQL
├── logs/             # Логи приложения
├── prometheus_data/  # Данные Prometheus
└── grafana_data/     # Данные Grafana
```

## 🌐 Сетевая конфигурация

Все сервисы работают в изолированной сети `trade-hedge-network`.

### Порты

| Сервис | Внутренний | Внешний | Описание |
|--------|------------|---------|----------|
| trade-hedge | 8081 | 8081 | Веб-интерфейс |
| postgres | 5432 | 5432 | База данных |
| adminer | 8080 | 8080 | Управление БД |

## 🔍 Мониторинг

### Health Checks

Все сервисы имеют health checks:

```bash
# Проверка состояния
docker-compose ps

# Детальная информация
docker inspect trade-hedge-app --format='{{.State.Health.Status}}'
```

### Логи

```bash
# Все логи
docker-compose logs -f

# Конкретный сервис
docker-compose logs -f trade-hedge
docker-compose logs -f postgres

# С ограничением по времени
docker-compose logs --since=1h trade-hedge
```

## 🔒 Безопасность

### Production рекомендации

1. **Измените пароли по умолчанию**
2. **Используйте SSL для базы данных**
3. **Настройте файрвол**
4. **Регулярные бэкапы**
5. **Мониторинг ресурсов**

### Пример production .env

```bash
# Безопасные пароли
DB_PASSWORD=very_secure_password_123!

# SSL для БД
DB_SSL_MODE=require

# Ограничение веб-интерфейса
WEBUI_HOST=127.0.0.1  # Только локально
```

## 🚨 Troubleshooting

### Проблемы с запуском

```bash
# Проверка логов
docker-compose logs trade-hedge

# Проверка конфигурации
docker-compose config

# Пересоздание контейнеров
docker-compose up -d --force-recreate
```

### Проблемы с базой данных

```bash
# Проверка подключения к БД
docker-compose exec postgres psql -U postgres -d trade_hedge -c "SELECT 1;"

# Восстановление БД
docker-compose down
docker volume rm trade-hedge_postgres_data
docker-compose up -d
```

### Проблемы с сетью

```bash
# Проверка сети
docker network ls
docker network inspect trade-hedge_trade-hedge-network

# Пересоздание сети
docker-compose down
docker network prune
docker-compose up -d
```

## 📈 Масштабирование

### Горизонтальное масштабирование

```yaml
# docker-compose.override.yml
version: '3.8'
services:
  trade-hedge:
    deploy:
      replicas: 3
```

### Ресурсы

```yaml
services:
  trade-hedge:
    deploy:
      resources:
        limits:
          memory: 512M
          cpus: '0.5'
        reservations:
          memory: 256M
          cpus: '0.25'
```

## 🔄 CI/CD интеграция

### GitHub Actions пример

```yaml
- name: Deploy with Docker Compose
  run: |
    cp env.example .env
    # Заполните секреты
    echo "BYBIT_API_KEY=${{ secrets.BYBIT_API_KEY }}" >> .env
    docker-compose up -d
```

### Автоматическое обновление

```bash
# Скрипт обновления
#!/bin/bash
cd /path/to/trade-hedge
git pull
docker-compose build
docker-compose up -d
```

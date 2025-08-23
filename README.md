# Trade Hedge

Система автоматического хеджирования торговых позиций, интегрированная с Freqtrade и Bybit.

## 🚀 Быстрый старт

```bash
# 1. Настройка конфигурации
make quickstart

# 2. Заполните API ключи
nano deploy/local/.env

# 3. Запуск приложения
make docker-up

# 4. Откройте веб-интерфейс
open http://localhost:8081
```

## 📁 Структура проекта

```
trade-hedge/
├── cmd/                    # Приложения
│   └── trade-hedge/        # Основное приложение
│       └── main.go         # Точка входа
├── internal/               # Приватный код (Clean Architecture)
│   ├── domain/             # Доменный слой
│   ├── usecases/           # Бизнес-логика
│   ├── infrastructure/     # Внешние зависимости
│   └── adapters/           # Адаптеры и контроллеры
├── config/                 # Конфигурационные файлы
│   ├── config.yaml.example # Пример YAML конфигурации
│   └── env.example         # Пример переменных окружения
├── docs/                   # Документация
│   ├── README.md           # Подробная документация
│   ├── DOCKER.md           # Docker развертывание
│   └── API.md              # API документация
├── deploy/                 # Развертывание
│   ├── local/              # Локальная разработка (docker-compose.yml + .env)
│   └── prod/               # Production развертывание (готовые образы)
├── Dockerfile             # Сборка образа
└── Makefile               # Команды для разработки
```

## 📚 Документация

- **[docs/README.md](docs/README.md)** - Подробная документация
- **[docs/DOCKER.md](docs/DOCKER.md)** - Docker развертывание  
- **[docs/API.md](docs/API.md)** - API документация
- **[config/config.yaml.example](config/config.yaml.example)** - Пример конфигурации
- **[config/env.example](config/env.example)** - Переменные окружения

## 🚀 Production развертывание

```bash
# На production сервере скачайте файлы:
wget https://raw.githubusercontent.com/your-org/trade-hedge/main/deploy/prod/docker-compose.yml
wget https://raw.githubusercontent.com/your-org/trade-hedge/main/deploy/prod/.env.example
wget https://raw.githubusercontent.com/your-org/trade-hedge/main/deploy/prod/init.sql

# Настройте и запустите:
cp .env.example .env
nano .env  # Укажите Docker образ и API ключи
docker compose up -d
```

## 🛠️ Команды

```bash
make help           # Показать все доступные команды
make build          # Сборка приложения
make run            # Локальный запуск
make docker-up      # Docker запуск
make test           # Тесты
make clean          # Очистка
```

## 🏗️ Архитектура

Проект построен на принципах **Clean Architecture**:

- **Domain** - Бизнес-логика и сущности
- **Use Cases** - Сценарии использования
- **Infrastructure** - Внешние зависимости (БД, API)
- **Adapters** - Адаптеры и контроллеры

## 📋 Возможности

- ✅ Автоматическое хеджирование убыточных позиций
- ✅ Интеграция с Freqtrade и Bybit
- ✅ Веб-интерфейс для мониторинга
- ✅ Строгая проверка баланса (без автоматической корректировки размера позиции)
- ✅ Управление рисками с фиксированным размером позиции
- ✅ Приоритизация сделок по просадке (сначала самые убыточные)
- ✅ Отображение размера ордеров в долларах
- ✅ Отслеживание статуса ордеров
- ✅ Docker контейнеризация

## 🔧 Технологии

- **Go 1.21+** - Основной язык

## 📝 Последние изменения

См. [CHANGELOG.md](CHANGELOG.md) для подробной информации об изменениях.

**Последнее обновление**: 
- Убрана автоматическая корректировка размера позиции - теперь система использует строго фиксированный размер из настроек
- Добавлена сортировка сделок по просадке для приоритизации самых убыточных позиций
- Добавлено отображение размера ордеров в долларах на всех страницах

## ⚠️ Важные замечания

### Минимальные лимиты Bybit
При настройке `position_amount` в конфигурации учитывайте минимальные лимиты Bybit:
- **Минимальная стоимость ордера**: 5 USDT для большинства пар (включая SOLUSDT)
- **Рекомендуемый размер**: 100+ USDT для надежности
- **Ошибка 170140**: "Order value exceeded lower limit" возникает при слишком маленькой сумме

Подробности см. в [docs/BYBIT_LIMITS.md](docs/BYBIT_LIMITS.md)
- **PostgreSQL** - База данных
- **Docker** - Контейнеризация
- **Clean Architecture** - Архитектурный паттерн

---

> Подробная документация доступна в папке [docs/](docs/)

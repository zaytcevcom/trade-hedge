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
- ✅ Проверка баланса и управление рисками
- ✅ Отслеживание статуса ордеров
- ✅ Docker контейнеризация

## 🔧 Технологии

- **Go 1.21+** - Основной язык
- **PostgreSQL** - База данных
- **Docker** - Контейнеризация
- **Clean Architecture** - Архитектурный паттерн

---

> Подробная документация доступна в папке [docs/](docs/)

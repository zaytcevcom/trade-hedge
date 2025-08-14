# Trade Hedge Makefile
# Удобные команды для разработки и развертывания

.PHONY: help build run test clean docker-build docker-up docker-down logs clean-cache clean-docker clean-all rebuild

# Помощь
help:
	@echo "Trade Hedge - Система автоматического хеджирования убытков"
	@echo ""
	@echo "Доступные команды:"
	@echo "  build          - Собрать бинарный файл"
	@echo "  run            - Запустить приложение локально"
	@echo "  test           - Запустить тесты"
	@echo "  clean          - Очистить артефакты сборки"
	@echo ""
	@echo "Docker команды:"
	@echo "  docker-build   - Собрать Docker образ"
	@echo "  docker-up      - Запустить production стек"
	@echo "  docker-down    - Остановить production стек"
	@echo "  docker-logs    - Показать логи контейнеров"
	@echo ""
	@echo "Команды пересборки:"
	@echo "  rebuild-dev     - Быстрая пересборка (только код)"
	@echo "  rebuild-local   - Пересборка и перезапуск локального стека"
	@echo "  rebuild-local-full - Полная пересборка с очисткой кэшей"
	@echo "  rebuild-docker  - Пересборка только Docker образа"
	@echo "  rebuild         - Полная пересборка с очисткой всех кэшей"
	@echo ""
	@echo "Утилиты:"
	@echo "  deps           - Установить/обновить зависимости"
	@echo "  lint           - Запустить линтер"
	@echo "  fmt            - Форматировать код"
	@echo ""
	@echo "Очистка кэшей:"
	@echo "  clean-cache    - Очистить кэши Go"
	@echo "  clean-docker   - Очистить кэши Docker"
	@echo "  clean-all      - Очистить все кэши"

# Локальная сборка
build:
	@echo "🔨 Сборка приложения..."
	go build -o trade-hedge ./cmd/trade-hedge

# Запуск локально
run: build
	@echo "🚀 Запуск приложения..."
	./trade-hedge

# Тесты
test:
	@echo "🧪 Запуск тестов..."
	go test -v ./...

# Очистка
clean:
	@echo "🧹 Очистка артефактов..."
	rm -f trade-hedge
	go clean

# Docker сборка
docker-build:
	@echo "🐳 Сборка Docker образа..."
	docker build -t trade-hedge:latest .

# Запуск локального стека
docker-up:
	@echo "🚀 Запуск локального стека..."
	@if [ ! -f deploy/local/.env ]; then \
		echo "❌ Файл deploy/local/.env не найден. Создайте его из config/env.example"; \
		echo "cp config/env.example deploy/local/.env"; \
		exit 1; \
	fi
	cd deploy/local && docker compose up -d



# Остановка локального стека
docker-down:
	@echo "🛑 Остановка локального стека..."
	cd deploy/local && docker compose down

nuke:
	cd deploy/local && docker compose down -v --remove-orphans
	cd deploy/local && docker compose pull

# Очистка кэшей Go
clean-cache:
	@echo "🧹 Очистка кэшей Go..."
	go clean -cache -modcache

# Очистка кэшей Docker
clean-docker:
	@echo "🐳 Очистка кэшей Docker..."
	docker system prune -af

# Полная очистка кэшей (Go + Docker)
clean-all: clean-cache clean-docker
	@echo "✨ Все кэши очищены!"

# Полная пересборка с очисткой кэшей
rebuild: clean-all build docker-build
	@echo "🚀 Полная пересборка завершена!"

# Быстрая пересборка для разработки (без очистки кэшей)
rebuild-dev: build
	@echo "🔨 Быстрая пересборка завершена!"

# Пересборка и перезапуск локального стека
rebuild-local: build
	@echo "🔄 Пересборка и перезапуск локального стека..."
	cd deploy/local && docker compose down
	cd deploy/local && docker compose up -d --build

# Полная пересборка локального стека с очисткой кэшей
rebuild-local-full: clean-all build
	@echo "🔄 Полная пересборка и перезапуск локального стека..."
	cd deploy/local && docker compose down
	cd deploy/local && docker compose up -d --build

# Пересборка только Docker образа
rebuild-docker: docker-down
	@echo "🔨 Пересборка Docker образа..."
	cd deploy/local && docker compose build --no-cache
	cd deploy/local && docker compose up -d

# Логи Docker
docker-logs:
	@echo "📋 Логи контейнеров..."
	cd deploy/local && docker compose logs -f

# Логи конкретного сервиса
logs-app:
	cd deploy/local && docker compose logs -f trade-hedge

logs-db:
	cd deploy/local && docker compose logs -f postgres

deploy-dockerhub:
	docker build --platform=linux/amd64 -t zaytcevcom/trade:1.0.0 .
	docker push zaytcevcom/trade:1.0.0


# Установка зависимостей
deps:
	@echo "📦 Установка зависимостей..."
	go mod download
	go mod tidy

# Линтер
lint:
	@echo "🔍 Запуск линтера..."
	golangci-lint run

# Форматирование кода
fmt:
	@echo "✨ Форматирование кода..."
	go fmt ./...

# Создание .env файлов из примера
setup-env:
	@if [ ! -f deploy/local/.env ]; then \
		cp config/env.example deploy/local/.env; \
		echo "✅ Создан файл deploy/local/.env из примера"; \
		echo "📝 Отредактируйте deploy/local/.env и заполните API ключи"; \
	else \
		echo "⚠️  Файл deploy/local/.env уже существует"; \
	fi
	@if [ ! -f config/config.yaml ]; then \
		cp config/config.yaml.example config/config.yaml; \
		echo "✅ Создан файл config/config.yaml из примера"; \
	else \
		echo "⚠️  Файл config/config.yaml уже существует"; \
	fi

# Проверка статуса
status:
	@echo "📊 Статус контейнеров:"
	docker compose ps

# Бэкап БД
backup-db:
	@echo "💾 Создание бэкапа БД..."
	docker compose exec postgres pg_dump -U postgres trade_hedge > backup_$(shell date +%Y%m%d_%H%M%S).sql

# Восстановление БД
restore-db:
	@echo "📥 Восстановление БД из бэкапа..."
	@read -p "Введите путь к файлу бэкапа: " backup_file; \
	docker compose exec -T postgres psql -U postgres trade_hedge < $$backup_file

# Быстрый старт для новых пользователей
quickstart: setup-env
	@echo "🚀 Быстрый старт Trade Hedge"
	@echo ""
	@echo "1️⃣  Отредактируйте файл deploy/local/.env (особенно API ключи)"
	@echo "2️⃣  Опционально: отредактируйте config/config.yaml"
	@echo "3️⃣  Запустите: make docker-up"
	@echo "4️⃣  Откройте: http://localhost:8081"
	@echo ""
	@echo "📚 Подробная документация в README.md"

# Используем официальный образ Go для сборки
FROM golang:1.21-alpine AS builder

# Устанавливаем необходимые пакеты
RUN apk add --no-cache git ca-certificates tzdata

# Создаем рабочую директорию
WORKDIR /app

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o trade-hedge ./cmd/trade-hedge

# Финальный образ
FROM alpine:3.18

# Устанавливаем необходимые пакеты
RUN apk --no-cache add ca-certificates tzdata wget

# Создаем пользователя для безопасности
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Создаем рабочую директорию
WORKDIR /app

# Копируем бинарный файл из builder
COPY --from=builder /app/trade-hedge .

# Создаем директории для логов и конфигурации
RUN mkdir -p /app/logs /app/config
COPY config/config.yaml.example /app/config/config.yaml.example

# Устанавливаем права доступа
RUN chown -R appuser:appgroup /app

# Переключаемся на непривилегированного пользователя
USER appuser

# Открываем порт для веб-интерфейса
EXPOSE 8081

# Добавляем метки
LABEL maintainer="trade-hedge"
LABEL version="1.0"
LABEL description="Trade Hedge - Automated Loss Hedging System"

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/api/status || exit 1

# Запускаем приложение
CMD ["./trade-hedge"]

-- Инициализация базы данных Trade Hedge
-- Этот скрипт выполняется при первом запуске PostgreSQL контейнера

-- Создаем расширения (если нужны)
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Устанавливаем часовой пояс
SET timezone = 'UTC';

-- Создаем схему для приложения (опционально)
-- CREATE SCHEMA IF NOT EXISTS trade_hedge;

-- Примечание: Таблицы будут созданы автоматически приложением
-- при первом запуске через методы в internal/infrastructure/database/postgresql.go

-- Базовые настройки производительности
ALTER SYSTEM SET log_statement = 'all';
ALTER SYSTEM SET log_min_duration_statement = 1000;

-- Применяем изменения
SELECT pg_reload_conf();

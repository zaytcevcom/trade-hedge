package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

// Config содержит всю конфигурацию приложения
type Config struct {
	Freqtrade FreqtradeConfig `yaml:"freqtrade"`
	Bybit     BybitConfig     `yaml:"bybit"`
	Database  DatabaseConfig  `yaml:"database"`
	Strategy  StrategyConfig  `yaml:"strategy"`
	WebUI     WebUIConfig     `yaml:"webui"`
}

// FreqtradeConfig конфигурация для подключения к Freqtrade
type FreqtradeConfig struct {
	APIURL   string `yaml:"api_url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// BybitConfig конфигурация для подключения к Bybit
type BybitConfig struct {
	APIKey         string `yaml:"api_key"`
	APISecret      string `yaml:"api_secret"`
	SpotURL        string `yaml:"spot_url"`
	BalanceURL     string `yaml:"balance_url"`
	OrderStatusURL string `yaml:"order_status_url"`
}

// DatabaseConfig конфигурация базы данных
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

// StrategyConfig конфигурация торговой стратегии
type StrategyConfig struct {
	PositionAmount float64 `yaml:"position_amount"` // Фиксированная сумма позиции в базовой валюте
	MaxLossPercent float64 `yaml:"max_loss_percent"`
	ProfitRatio    float64 `yaml:"profit_ratio"`
	BaseCurrency   string  `yaml:"base_currency"`
	CheckInterval  int     `yaml:"check_interval"` // Интервал проверки в секундах (0 = одноразовое выполнение)
	RetryAttempts  int     `yaml:"retry_attempts"` // Количество попыток размещения ордера
	RetryDelay     int     `yaml:"retry_delay"`    // Задержка между попытками в секундах
}

// WebUIConfig конфигурация веб-интерфейса
type WebUIConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Host    string `yaml:"host"`
}

// LoadConfig загружает конфигурацию из YAML файла с поддержкой переменных окружения
func LoadConfig(path string) (*Config, error) {
	config := &Config{}

	// Устанавливаем значения по умолчанию
	config.setDefaults()

	// Загружаем из файла (если существует)
	if _, err := os.Stat(path); err == nil {
		if err := config.loadFromFile(path); err != nil {
			return nil, fmt.Errorf("ошибка загрузки из файла: %w", err)
		}
	}

	// Переопределяем переменными окружения
	config.loadFromEnv()

	// Валидируем конфигурацию
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("ошибка валидации конфигурации: %w", err)
	}

	return config, nil
}

// setDefaults устанавливает значения по умолчанию
func (c *Config) setDefaults() {
	c.Database.Host = "localhost"
	c.Database.Port = 5432
	c.Database.User = "postgres"
	c.Database.DBName = "trade_hedge"
	c.Database.SSLMode = "disable"

	c.Strategy.PositionAmount = 50.0
	c.Strategy.MaxLossPercent = 3.0
	c.Strategy.ProfitRatio = 0.7
	c.Strategy.BaseCurrency = "USDT"
	c.Strategy.CheckInterval = 300
	c.Strategy.RetryAttempts = 3
	c.Strategy.RetryDelay = 2

	c.WebUI.Enabled = false
	c.WebUI.Host = "localhost"
	c.WebUI.Port = 8081
}

// loadFromFile загружает конфигурацию из YAML файла
func (c *Config) loadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(c); err != nil {
		return fmt.Errorf("ошибка парсинга YAML: %w", err)
	}

	return nil
}

// loadFromEnv загружает настройки из переменных окружения
func (c *Config) loadFromEnv() {
	// Freqtrade
	if v := os.Getenv("FREQTRADE_API_URL"); v != "" {
		c.Freqtrade.APIURL = v
	}
	if v := os.Getenv("FREQTRADE_USERNAME"); v != "" {
		c.Freqtrade.Username = v
	}
	if v := os.Getenv("FREQTRADE_PASSWORD"); v != "" {
		c.Freqtrade.Password = v
	}

	// Bybit
	if v := os.Getenv("BYBIT_API_KEY"); v != "" {
		c.Bybit.APIKey = v
	}
	if v := os.Getenv("BYBIT_API_SECRET"); v != "" {
		c.Bybit.APISecret = v
	}
	if v := os.Getenv("BYBIT_SPOT_URL"); v != "" {
		c.Bybit.SpotURL = v
	}
	if v := os.Getenv("BYBIT_BALANCE_URL"); v != "" {
		c.Bybit.BalanceURL = v
	}
	if v := os.Getenv("BYBIT_ORDER_STATUS_URL"); v != "" {
		c.Bybit.OrderStatusURL = v
	}

	// Database
	if v := os.Getenv("DB_HOST"); v != "" {
		c.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Database.Port = port
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		c.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		c.Database.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		c.Database.DBName = v
	}
	if v := os.Getenv("DB_SSL_MODE"); v != "" {
		c.Database.SSLMode = v
	}

	// Strategy
	if v := os.Getenv("STRATEGY_POSITION_AMOUNT"); v != "" {
		if amount, err := strconv.ParseFloat(v, 64); err == nil {
			c.Strategy.PositionAmount = amount
		}
	}
	if v := os.Getenv("STRATEGY_MAX_LOSS_PERCENT"); v != "" {
		if percent, err := strconv.ParseFloat(v, 64); err == nil {
			c.Strategy.MaxLossPercent = percent
		}
	}
	if v := os.Getenv("STRATEGY_PROFIT_RATIO"); v != "" {
		if ratio, err := strconv.ParseFloat(v, 64); err == nil {
			c.Strategy.ProfitRatio = ratio
		}
	}
	if v := os.Getenv("STRATEGY_BASE_CURRENCY"); v != "" {
		c.Strategy.BaseCurrency = v
	}
	if v := os.Getenv("STRATEGY_CHECK_INTERVAL"); v != "" {
		if interval, err := strconv.Atoi(v); err == nil {
			c.Strategy.CheckInterval = interval
		}
	}
	if v := os.Getenv("STRATEGY_RETRY_ATTEMPTS"); v != "" {
		if attempts, err := strconv.Atoi(v); err == nil {
			c.Strategy.RetryAttempts = attempts
		}
	}
	if v := os.Getenv("STRATEGY_RETRY_DELAY"); v != "" {
		if delay, err := strconv.Atoi(v); err == nil {
			c.Strategy.RetryDelay = delay
		}
	}

	// WebUI
	if v := os.Getenv("WEBUI_ENABLED"); v != "" {
		c.WebUI.Enabled = strings.ToLower(v) == "true"
	}
	if v := os.Getenv("WEBUI_HOST"); v != "" {
		c.WebUI.Host = v
	}
	if v := os.Getenv("WEBUI_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.WebUI.Port = port
		}
	}
}

// Validate проверяет корректность конфигурации
func (c *Config) Validate() error {
	// Валидация Freqtrade
	if strings.TrimSpace(c.Freqtrade.APIURL) == "" {
		return fmt.Errorf("freqtrade.api_url не может быть пустым")
	}
	if _, err := url.Parse(c.Freqtrade.APIURL); err != nil {
		return fmt.Errorf("freqtrade.api_url содержит некорректный URL: %w", err)
	}
	if strings.TrimSpace(c.Freqtrade.Username) == "" {
		return fmt.Errorf("freqtrade.username не может быть пустым")
	}
	if strings.TrimSpace(c.Freqtrade.Password) == "" {
		return fmt.Errorf("freqtrade.password не может быть пустым")
	}

	// Валидация Bybit
	if strings.TrimSpace(c.Bybit.APIKey) == "" {
		return fmt.Errorf("bybit.api_key не может быть пустым")
	}
	if strings.TrimSpace(c.Bybit.APISecret) == "" {
		return fmt.Errorf("bybit.api_secret не может быть пустым")
	}

	urls := map[string]string{
		"bybit.spot_url":         c.Bybit.SpotURL,
		"bybit.balance_url":      c.Bybit.BalanceURL,
		"bybit.order_status_url": c.Bybit.OrderStatusURL,
	}

	for name, urlStr := range urls {
		if strings.TrimSpace(urlStr) == "" {
			return fmt.Errorf("%s не может быть пустым", name)
		}
		if _, err := url.Parse(urlStr); err != nil {
			return fmt.Errorf("%s содержит некорректный URL: %w", name, err)
		}
	}

	// Валидация Database
	if strings.TrimSpace(c.Database.Host) == "" {
		return fmt.Errorf("database.host не может быть пустым")
	}
	if c.Database.Port < 1 || c.Database.Port > 65535 {
		return fmt.Errorf("database.port должен быть в диапазоне 1-65535, получен: %d", c.Database.Port)
	}
	if strings.TrimSpace(c.Database.User) == "" {
		return fmt.Errorf("database.user не может быть пустым")
	}
	if strings.TrimSpace(c.Database.Password) == "" {
		return fmt.Errorf("database.password не может быть пустым")
	}
	if strings.TrimSpace(c.Database.DBName) == "" {
		return fmt.Errorf("database.dbname не может быть пустым")
	}

	// Валидация Strategy
	if c.Strategy.PositionAmount <= 0 {
		return fmt.Errorf("strategy.position_amount должен быть положительным, получен: %.2f", c.Strategy.PositionAmount)
	}
	if c.Strategy.MaxLossPercent <= 0 || c.Strategy.MaxLossPercent >= 100 {
		return fmt.Errorf("strategy.max_loss_percent должен быть в диапазоне (0, 100), получен: %.2f", c.Strategy.MaxLossPercent)
	}
	if c.Strategy.ProfitRatio <= 0 {
		return fmt.Errorf("strategy.profit_ratio должен быть положительным, получен: %.2f", c.Strategy.ProfitRatio)
	}
	if strings.TrimSpace(c.Strategy.BaseCurrency) == "" {
		return fmt.Errorf("strategy.base_currency не может быть пустым")
	}
	if c.Strategy.CheckInterval < 0 {
		return fmt.Errorf("strategy.check_interval не может быть отрицательным, получен: %d", c.Strategy.CheckInterval)
	}
	if c.Strategy.RetryAttempts <= 0 {
		return fmt.Errorf("strategy.retry_attempts должен быть положительным, получен: %d", c.Strategy.RetryAttempts)
	}
	if c.Strategy.RetryDelay < 0 {
		return fmt.Errorf("strategy.retry_delay не может быть отрицательным, получен: %d", c.Strategy.RetryDelay)
	}

	// Валидация WebUI
	if c.WebUI.Enabled {
		if c.WebUI.Port < 1 || c.WebUI.Port > 65535 {
			return fmt.Errorf("webui.port должен быть в диапазоне 1-65535, получен: %d", c.WebUI.Port)
		}
		if strings.TrimSpace(c.WebUI.Host) == "" {
			return fmt.Errorf("webui.host не может быть пустым")
		}
	}

	return nil
}

// GetDatabaseConnectionString возвращает строку подключения к базе данных
func (c *Config) GetDatabaseConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode)
}

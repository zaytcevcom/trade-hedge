package webui

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"trade-hedge/internal/domain/repositories"
	"trade-hedge/internal/infrastructure/config"
	"trade-hedge/internal/pkg/logger"
	"trade-hedge/internal/usecases"
)

//go:embed templates/*
var templateFS embed.FS

// Server веб-сервер для мониторинга
type Server struct {
	webUIConfig          *config.WebUIConfig
	fullConfig           *config.Config
	hedgeRepo            repositories.HedgeRepository
	hedgeUseCase         *usecases.HedgeStrategyUseCase
	statusCheckerUseCase *usecases.StatusCheckerUseCase
	server               *http.Server
	templates            *template.Template
}

// NewServer создает новый веб-сервер
func NewServer(
	webUIConfig *config.WebUIConfig,
	fullConfig *config.Config,
	hedgeRepo repositories.HedgeRepository,
	hedgeUseCase *usecases.HedgeStrategyUseCase,
	statusCheckerUseCase *usecases.StatusCheckerUseCase,
) *Server {
	s := &Server{
		webUIConfig:          webUIConfig,
		fullConfig:           fullConfig,
		hedgeRepo:            hedgeRepo,
		hedgeUseCase:         hedgeUseCase,
		statusCheckerUseCase: statusCheckerUseCase,
	}

	// Загружаем шаблоны
	s.loadTemplates()

	// Настраиваем роуты
	mux := http.NewServeMux()
	s.setupRoutes(mux)

	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", webUIConfig.Host, webUIConfig.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// loadTemplates загружает HTML шаблоны
func (s *Server) loadTemplates() {
	var err error
	s.templates, err = template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Fatalf("❌ Ошибка загрузки шаблонов: %v", err)
	}
}

// setupRoutes настраивает маршруты
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// Статические файлы и основные страницы
	mux.HandleFunc("/", s.handleDashboard)
	mux.HandleFunc("/trades", s.handleTrades)
	mux.HandleFunc("/config", s.handleConfig)

	// API эндпоинты
	mux.HandleFunc("/api/trades", s.handleAPITrades)
	mux.HandleFunc("/api/status", s.handleAPIStatus)
	mux.HandleFunc("/api/execute", s.handleAPIExecute)
	mux.HandleFunc("/api/check-status", s.handleAPICheckStatus)
	mux.HandleFunc("/api/balance", s.handleAPIBalance)
}

// Start запускает веб-сервер
func (s *Server) Start(ctx context.Context) error {
	logger.LogWithTime("🌐 Запуск веб-интерфейса на http://%s:%d", s.webUIConfig.Host, s.webUIConfig.Port)

	// Запускаем сервер в горутине
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.LogWithTime("❌ Ошибка веб-сервера: %v", err)
		}
	}()

	// Ждем сигнала остановки
	<-ctx.Done()

	// Graceful shutdown
	log.Println("🛑 Остановка веб-сервера...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.server.Shutdown(shutdownCtx)
}

// Stop останавливает веб-сервер
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

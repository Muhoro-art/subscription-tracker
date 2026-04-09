package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"subscription-service/internal/config"
	"subscription-service/internal/handler"
	"subscription-service/internal/repository"
	"subscription-service/internal/service"

	_ "subscription-service/docs"
)

// @title Subscription Service API
// @version 1.0
// @description REST API service for aggregating user online subscription data
// @host localhost:8080
// @BasePath /
func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	logger.Info("configuration loaded",
		zap.String("server_port", cfg.Server.Port),
		zap.String("db_host", cfg.Database.Host),
	)

	// Run migrations
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.Database.User, cfg.Database.Password,
		cfg.Database.Host, cfg.Database.Port,
		cfg.Database.Name, cfg.Database.SSLMode,
	)

	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		logger.Fatal("failed to create migrate instance", zap.Error(err))
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		logger.Fatal("failed to run migrations", zap.Error(err))
	}
	logger.Info("migrations applied successfully")

	// Connect to database
	db, err := sqlx.Connect("postgres", cfg.Database.DSN())
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()
	logger.Info("connected to database")

	// Initialize layers
	repo := repository.NewSubscriptionRepository(db, logger)
	svc := service.NewSubscriptionService(repo, logger)
	h := handler.NewSubscriptionHandler(svc, logger)

	// Setup router
	r := gin.Default()
	h.RegisterRoutes(r)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	logger.Info("starting server", zap.String("port", cfg.Server.Port))
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		logger.Fatal("failed to start server", zap.Error(err))
	}
}

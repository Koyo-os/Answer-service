package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Koyo-os/answer-service/internal/config"
	"github.com/Koyo-os/answer-service/internal/entity"
	"github.com/Koyo-os/answer-service/internal/repository"
	"github.com/Koyo-os/answer-service/pkg/logger"
	"github.com/Koyo-os/answer-service/pkg/retrier"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	eventChan := make(chan entity.Event, 100) // Add buffer for better performance

	logCfg := logger.Config{
		LogFile:   "app.log",
		LogLevel:  "debug",
		AppName:   "answer-service",
		AddCaller: true,
	}

	if err := logger.Init(logCfg); err != nil {
		panic(err)
	}

	defer logger.Sync()

	logger := logger.Get()

	cfg := config.NewConfig()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	logger.Info("connecting to mariadb...", zap.String("dsn", dsn))

	db, err := retrier.Connect(10, 10, func() (*gorm.DB, error) {
		return gorm.Open(mysql.Open(dsn))
	})
	if err != nil {
		logger.Error("error initialyze database",
			zap.String("dsn", dsn),
			zap.Error(err))

		return
	}

	logger.Info("connected to mariadb", zap.String("dsn", dsn))

	if err := db.AutoMigrate(&entity.Answer{}, &entity.Element{}); err != nil {
		logger.Error("failed to migrate database", zap.Error(err))
		return
	}

	repo := repository.NewRepository(db, logger)

	
}

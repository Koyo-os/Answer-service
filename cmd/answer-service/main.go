package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Koyo-os/answer-service/internal/config"
	"github.com/Koyo-os/answer-service/internal/entity"
	"github.com/Koyo-os/answer-service/internal/repository"
	"github.com/Koyo-os/answer-service/internal/service"
	"github.com/Koyo-os/answer-service/pkg/closer"
	"github.com/Koyo-os/answer-service/pkg/health"
	"github.com/Koyo-os/answer-service/pkg/logger"
	"github.com/Koyo-os/answer-service/pkg/retrier"
	"github.com/Koyo-os/answer-service/pkg/transport/casher"
	"github.com/Koyo-os/answer-service/pkg/transport/consumer"
	"github.com/Koyo-os/answer-service/pkg/transport/listener"
	"github.com/Koyo-os/answer-service/pkg/transport/publisher"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
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

	rabbitmqConns, err := retrier.MultiConnects(2, func() (*amqp.Connection, error) {
		return amqp.Dial(cfg.Urls["rabbitmq"])
	}, &retrier.RetrierOpts{Count: 3, Interval: 5})
	if err != nil {
		logger.Error("error connect to rabbitmq",
			zap.String("url", cfg.Urls["rabbitmq"]),
			zap.Error(err))

		return
	}

	publisher, err := publisher.Init(cfg, logger, rabbitmqConns[0])
	if err != nil {
		logger.Error("error initialize publisher", zap.Error(err))

		return
	}

	consumer, err := consumer.Init(cfg, logger, rabbitmqConns[1])
	if err != nil {
		logger.Error("error initialize consumer", zap.Error(err))

		return
	}

	redisConn, err := retrier.Connect(3, 5, func() (*redis.Client, error) {
		client := redis.NewClient(&redis.Options{
			Addr:     cfg.Urls["redis"],
			DB:       0,
			Password: "",
		})

		return client, client.Ping(context.Background()).Err()
	})
	if err != nil {
		logger.Error("error connect to redis", zap.Error(err))

		return
	}

	casher := casher.Init(redisConn, logger)

	core := service.NewService(casher, publisher, repo, 10 * time.Second)

	listener := listener.NewListener(logger, core, eventChan)

	logger.Info("service ready to start!")

	healther := health.NewHealthChecker(publisher, casher)

	go listener.Run(context.Background())
	go consumer.ConsumeMessages(eventChan)
	go healther.RunServer(":8080")


	<- signalChan

	closer := closer.NewShutdown(publisher, casher, consumer)
	closer.ShutdownAll(context.Background())
}

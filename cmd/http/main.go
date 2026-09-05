package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/yehezkiel1086/secure-go-api/internal/adapter/config"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/handler"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/rabbitmq"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/storage/postgres"
	"github.com/yehezkiel1086/secure-go-api/internal/adapter/storage/postgres/repository"
	"github.com/yehezkiel1086/secure-go-api/internal/core/service"
)

func handleError(msg string, err error) {
	if err != nil {
		slog.Error(msg, "error", err)
		os.Exit(1)
	}
}

// @title           Secure Go API
// @version         1.0
// @description     A production-focused REST API written in Go, structured with Hexagonal Architecture (Ports and Adapters).
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	conf, err := config.New()
	handleError("failed to load .env configs", err)
	slog.Info(".env configs loaded successfully", "app", conf.App.Name, "env", conf.App.Env)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, conf.DB)
	handleError("failed to connect to postgres", err)
	defer pool.Close()
	slog.Info("connected to postgres database successfully")

	err = postgres.Migrate(ctx, pool)
	handleError("failed to run database migrations", err)

	rabbitClient, err := rabbitmq.NewClient(conf.Rabbitmq)
	handleError("failed to connect to rabbitmq", err)
	defer rabbitClient.Close()
	slog.Info("connected to rabbitmq successfully")

	emailPublisher, err := rabbitmq.NewPublisher(rabbitClient)
	handleError("failed to create rabbitmq publisher", err)
	defer emailPublisher.Close()

	emailSender := rabbitmq.NewSMTPEmailSender(conf.SMTP)
	emailConsumer := rabbitmq.NewConsumer(rabbitClient, emailSender)

	// start rabbitmq consumer worker in background
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()

	go func() {
		if err := emailConsumer.Start(workerCtx); err != nil {
			slog.Error("rabbitmq consumer stopped with error", "error", err)
		}
	}()

	userRepo := repository.NewUserRepositoryWithDB(pool)
	authRepo := repository.NewAuthRepositoryWithDB(pool)
	jobRepo := repository.NewJobRepositoryWithDB(pool)

	userService := service.NewUserService(userRepo, emailPublisher)
	authService := service.NewAuthService(authRepo, userRepo, conf.JWT)
	jobService := service.NewJobService(jobRepo)

	userHandler := handler.NewUserHandler(userService)
	authHandler := handler.NewAuthHandler(authService, conf.JWT, conf.App)
	jobHandler := handler.NewJobHandler(jobService)

	router := handler.NewRouter(conf, userHandler, authHandler, jobHandler)

	slog.Info("starting server", "host", conf.HTTP.Host, "port", conf.HTTP.Port)
	if err := router.Run(); err != nil {
		handleError("failed to run server", err)
	}
}

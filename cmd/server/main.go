package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/retry"

	"L3.6/internal/config"
	"L3.6/internal/handler"
	"L3.6/internal/logger"
	"L3.6/internal/repository"
	"L3.6/internal/service"
	"L3.6/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	if err := logger.Init(cfg.LogLevel, cfg.AppName, cfg.Env); err != nil {
		logger.Error("Failed to init logger", "error", err)
		os.Exit(1)
	}
	log := logger.Ctx(context.Background())
	log.Info("Starting transaction service", "port", cfg.HTTPServerPort)

	sqlDB, err := sql.Open("postgres", cfg.PostgresDSN)
	if err != nil {
		log.Error("Failed to open DB for migrations", "error", err)
		os.Exit(1)
	}
	if err := migrations.Apply(sqlDB); err != nil {
		log.Error("Failed to apply migrations", "error", err)
		os.Exit(1)
	}
	sqlDB.Close()
	log.Info("Migrations applied successfully")

	var repo repository.Repository
	strategy := retry.Strategy{Attempts: 5, Delay: 2 * time.Second, Backoff: 2}
	err = retry.Do(func() error {
		var e error
		repo, e = repository.New(cfg.PostgresDSN)
		if e != nil {
			log.Error("DB connection failed", "error", e)
		}
		return e
	}, strategy)
	if err != nil {
		log.Error("Cannot connect to PostgreSQL after retries", "error", err)
		os.Exit(1)
	}
	defer repo.Close()
	log.Info("Connected to PostgreSQL")

	svc := service.NewTransactionService(repo)
	h := handler.NewHandler(svc)
	engine := ginext.New("release")
	engine.Use(ginext.Logger(), ginext.Recovery())
	h.RegisterRoutes(engine)

	srv := &http.Server{
		Addr:    ":" + cfg.HTTPServerPort,
		Handler: engine,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Info("HTTP server listening", "port", cfg.HTTPServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	log.Info("Shutting down gracefully...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Server shutdown error", "error", err)
	}
	log.Info("Server stopped")
}

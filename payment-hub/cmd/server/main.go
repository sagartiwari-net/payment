package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/buyahref/payment-hub/internal/api"
	"github.com/buyahref/payment-hub/internal/config"
	"github.com/buyahref/payment-hub/internal/database"
	"github.com/buyahref/payment-hub/internal/logger"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	db, err := database.Connect(cfg.DatabaseDSN())
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	app := api.NewApp(cfg, log, db)

	go func() {
		addr := fmt.Sprintf(":%s", cfg.AppPort)
		log.Info("starting server", zap.String("addr", addr), zap.String("env", cfg.AppEnv))
		if err := app.Listen(addr); err != nil {
			log.Fatal("server stopped", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server")
	if err := app.Shutdown(); err != nil {
		log.Error("shutdown error", zap.Error(err))
	}
}

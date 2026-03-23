package main

import (
	"log"

	"buhpro/internal/app"
	"buhpro/internal/config"
	"buhpro/internal/platform/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logg := logger.New(cfg.App.LogLevel)

	application, err := app.New(cfg, logg)
	if err != nil {
		logg.Error("application initialization failed", "error", err)
		log.Fatalf("init app: %v", err)
	}

	if err := application.Run(); err != nil {
		logg.Error("application stopped with error", "error", err)
		log.Fatalf("run app: %v", err)
	}
}

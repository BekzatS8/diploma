package main

import (
	"context"
	"log"
	"time"

	commonauth "buhpro/internal/common/auth"
	"buhpro/internal/config"
	authmodule "buhpro/internal/modules/auth"
	"buhpro/internal/platform/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if cfg.Bootstrap.AdminEmail == "" || cfg.Bootstrap.AdminPassword == "" {
		log.Fatal("BOOTSTRAP_ADMIN_EMAIL and BOOTSTRAP_ADMIN_PASSWORD are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DB)
	if err != nil {
		log.Fatalf("init db: %v", err)
	}
	defer pool.Close()

	jwtManager := commonauth.NewJWTManager(cfg.JWT.Issuer, cfg.JWT.AccessSecretKey, cfg.JWT.RefreshSecretKey, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)
	service := authmodule.NewService(authmodule.NewRepository(pool), jwtManager)

	if err := service.BootstrapAdmin(ctx, cfg.Bootstrap.AdminEmail, cfg.Bootstrap.AdminPassword); err != nil {
		log.Fatalf("bootstrap admin: %v", err)
	}

	log.Println("admin bootstrap completed")
}

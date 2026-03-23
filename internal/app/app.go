package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	commonauth "buhpro/internal/common/auth"
	"buhpro/internal/config"
	"buhpro/internal/http/handlers/system"
	"buhpro/internal/http/router"
	authmodule "buhpro/internal/modules/auth"
	profilemodule "buhpro/internal/modules/profile"
	"buhpro/internal/platform/db"
	"buhpro/internal/platform/metrics"
	"buhpro/internal/platform/payments"
	"buhpro/internal/platform/storage"

	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	cfg        config.Config
	log        *slog.Logger
	httpServer *http.Server
	db         *pgxpool.Pool
}

type readinessChecker struct {
	dbPool      *pgxpool.Pool
	healthCheck time.Duration
}

func (r *readinessChecker) Check(ctx context.Context) error {
	return db.Check(ctx, r.dbPool, r.healthCheck)
}

func New(cfg config.Config, log *slog.Logger) (*App, error) {
	startupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if cfg.App.AutoMigrate {
		if err := db.ApplyMigrations(cfg.DB.URL, "migrations"); err != nil {
			return nil, fmt.Errorf("apply migrations: %w", err)
		}
		log.Info("database migrations applied")
	}

	dbPool, err := db.NewPool(startupCtx, cfg.DB)
	if err != nil {
		return nil, fmt.Errorf("init db pool: %w", err)
	}

	jwtManager := commonauth.NewJWTManager(
		cfg.JWT.Issuer,
		cfg.JWT.AccessSecretKey,
		cfg.JWT.RefreshSecretKey,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
	)

	_ = storage.NewMock()
	_ = payments.NewMock()

	authRepo := authmodule.NewRepository(dbPool)
	authService := authmodule.NewService(authRepo, jwtManager)
	profileRepo := profilemodule.NewRepository(dbPool)
	profileService := profilemodule.NewService(profileRepo)

	if cfg.Bootstrap.EnableAdmin {
		if err := authService.BootstrapAdmin(startupCtx, cfg.Bootstrap.AdminEmail, cfg.Bootstrap.AdminPassword); err != nil {
			return nil, fmt.Errorf("bootstrap admin: %w", err)
		}
		log.Info("admin bootstrap processed")
	}

	authHandler := authmodule.NewHandler(authService, profileService)
	profileHandler := profilemodule.NewHandler(profileService)

	metricsCollector := metrics.New()
	systemHandler := system.NewHandler(&readinessChecker{dbPool: dbPool, healthCheck: cfg.DB.HealthTimeout})

	engine := router.New(router.Deps{
		Config:         cfg,
		Logger:         log,
		SystemHandlers: systemHandler,
		JWTManager:     jwtManager,
		AuthHandler:    authHandler,
		ProfileHandler: profileHandler,
		Metrics:        metricsCollector,
	})

	httpServer := &http.Server{
		Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:      engine,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return &App{cfg: cfg, log: log, httpServer: httpServer, db: dbPool}, nil
}

func (a *App) Run() error {
	errCh := make(chan error, 1)

	go func() {
		a.log.Info("starting http server", slog.String("addr", a.httpServer.Addr))
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		a.log.Info("shutdown signal received", slog.String("signal", sig.String()))
	case err := <-errCh:
		return fmt.Errorf("http server error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), a.cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := a.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	a.db.Close()
	a.log.Info("application shutdown completed")

	return nil
}

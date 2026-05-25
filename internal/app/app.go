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
	attachmentsmodule "buhpro/internal/modules/attachments"
	authmodule "buhpro/internal/modules/auth"
	chatsmodule "buhpro/internal/modules/chats"
	coursesmodule "buhpro/internal/modules/courses"
	devpaymentsmodule "buhpro/internal/modules/devpayments"
	leadsmodule "buhpro/internal/modules/leads"
	notificationsmodule "buhpro/internal/modules/notifications"
	ordersmodule "buhpro/internal/modules/orders"
	paymentmodule "buhpro/internal/modules/payment"
	profilemodule "buhpro/internal/modules/profile"
	ratingsmodule "buhpro/internal/modules/ratingsanctions"
	responsesmodule "buhpro/internal/modules/responses"
	reviewsmodule "buhpro/internal/modules/reviews"
	selectionmodule "buhpro/internal/modules/selection"
	uploadsmodule "buhpro/internal/modules/uploads"
	walletsmodule "buhpro/internal/modules/wallets"
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

	storageProvider, err := storage.NewLocal(cfg.Storage.LocalPath, cfg.Storage.PublicBaseURL)
	if err != nil {
		return nil, fmt.Errorf("init storage: %w", err)
	}
	paymentProvider := payments.NewMock()
	if cfg.Payments.Provider == "yookassa" {
		paymentProvider, err = payments.NewYooKassa(payments.YooKassaConfig{
			ShopID:    cfg.Payments.YooKassaShopID,
			SecretKey: cfg.Payments.YooKassaSecretKey,
			ReturnURL: cfg.Payments.YooKassaReturnURL,
		})
		if err != nil {
			return nil, fmt.Errorf("init yookassa payment provider: %w", err)
		}
	}

	authRepo := authmodule.NewRepository(dbPool)
	authService := authmodule.NewService(authRepo, jwtManager)
	uploadsRepo := uploadsmodule.NewRepository(dbPool)
	uploadsService := uploadsmodule.NewService(uploadsRepo, storageProvider)
	profileRepo := profilemodule.NewRepository(dbPool)
	profileService := profilemodule.NewService(profileRepo, uploadsService)
	ordersRepo := ordersmodule.NewRepository(dbPool)
	ordersService := ordersmodule.NewService(ordersRepo, paymentProvider, cfg.Payments.Provider, cfg.Orders.PostingFee, cfg.Orders.DefaultCurrency)
	responsesRepo := responsesmodule.NewRepository(dbPool)
	notificationsRepo := notificationsmodule.NewRepository(dbPool)
	notificationsService := notificationsmodule.NewService(notificationsRepo)
	ratingRepo := ratingsmodule.NewRepository(dbPool, ratingsmodule.RepositoryOptions{
		AutoAssignCourseOnLowRating: cfg.Sanctions.AutoAssignCourseOnLowRating,
		DefaultLowRatingCourseID:    cfg.Sanctions.DefaultLowRatingCourseID,
	})
	ratingService := ratingsmodule.NewService(ratingRepo)
	responsesService := responsesmodule.NewService(responsesRepo, paymentProvider, cfg.Payments.Provider, cfg.Orders.ResponseSubmissionFee, cfg.Orders.DefaultCurrency, ratingService)
	devPaymentsRepo := devpaymentsmodule.NewRepository(dbPool)
	devEndpointsEnabled := cfg.App.Env != "production" && cfg.Dev.EnablePaymentEndpoints
	devPaymentsService := devpaymentsmodule.NewService(devPaymentsRepo, devEndpointsEnabled, notificationsService)
	paymentService := paymentmodule.NewService(ordersService, devPaymentsService)
	selectionRepo := selectionmodule.NewRepository(dbPool)
	selectionService := selectionmodule.NewService(selectionRepo, notificationsService)
	reviewsRepo := reviewsmodule.NewRepository(dbPool)
	reviewsService := reviewsmodule.NewService(reviewsRepo, dbPool, ratingService, notificationsService)
	coursesRepo := coursesmodule.NewRepository(dbPool)
	coursesService := coursesmodule.NewService(coursesRepo, notificationsService, uploadsService, coursesmodule.ServiceOptions{
		ExecutorCreatorMinRating:  cfg.Courses.ExecutorCreatorMinRating,
		ExecutorCreatorMinReviews: cfg.Courses.ExecutorCreatorMinReviews,
	})
	chatsRepo := chatsmodule.NewRepository(dbPool)
	chatsService := chatsmodule.NewService(chatsRepo, notificationsService, uploadsService)
	attachmentsRepo := attachmentsmodule.NewRepository(dbPool)
	attachmentsService := attachmentsmodule.NewService(attachmentsRepo, uploadsService)
	leadsRepo := leadsmodule.NewRepository(dbPool)
	leadsService := leadsmodule.NewService(leadsRepo, storageProvider)
	walletsRepo := walletsmodule.NewRepository(dbPool)
	walletsService := walletsmodule.NewService(walletsRepo, cfg.Orders.DefaultCurrency)

	if cfg.Bootstrap.EnableAdmin {
		if err := authService.BootstrapAdmin(startupCtx, cfg.Bootstrap.AdminEmail, cfg.Bootstrap.AdminPassword); err != nil {
			return nil, fmt.Errorf("bootstrap admin: %w", err)
		}
		log.Info("admin bootstrap processed")
	}

	authHandler := authmodule.NewHandler(authService, profileService)
	profileHandler := profilemodule.NewHandler(profileService)
	ordersHandler := ordersmodule.NewHandler(ordersService)
	responsesHandler := responsesmodule.NewHandler(responsesService)
	devPaymentsHandler := devpaymentsmodule.NewHandler(devPaymentsService)
	paymentHandler := paymentmodule.NewHandler(paymentService)
	selectionHandler := selectionmodule.NewHandler(selectionService)
	reviewsHandler := reviewsmodule.NewHandler(reviewsService)
	ratingHandler := ratingsmodule.NewHandler(ratingService)
	coursesHandler := coursesmodule.NewHandler(coursesService)
	chatsHandler := chatsmodule.NewHandler(chatsService)
	notificationsHandler := notificationsmodule.NewHandler(notificationsService)
	uploadsHandler := uploadsmodule.NewHandler(uploadsService)
	attachmentsHandler := attachmentsmodule.NewHandler(attachmentsService)
	leadsHandler := leadsmodule.NewHandler(leadsService)
	walletsHandler := walletsmodule.NewHandler(walletsService)

	metricsCollector := metrics.New()
	systemHandler := system.NewHandler(&readinessChecker{dbPool: dbPool, healthCheck: cfg.DB.HealthTimeout})

	engine := router.New(router.Deps{
		Config:               cfg,
		Logger:               log,
		SystemHandlers:       systemHandler,
		JWTManager:           jwtManager,
		AuthHandler:          authHandler,
		ProfileHandler:       profileHandler,
		OrdersHandler:        ordersHandler,
		ResponsesHandler:     responsesHandler,
		PaymentHandler:       paymentHandler,
		DevPaymentsHandler:   devPaymentsHandler,
		SelectionHandler:     selectionHandler,
		ReviewsHandler:       reviewsHandler,
		RatingHandler:        ratingHandler,
		CoursesHandler:       coursesHandler,
		ChatsHandler:         chatsHandler,
		NotificationsHandler: notificationsHandler,
		UploadsHandler:       uploadsHandler,
		AttachmentsHandler:   attachmentsHandler,
		LeadsHandler:         leadsHandler,
		WalletsHandler:       walletsHandler,
		Metrics:              metricsCollector,
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

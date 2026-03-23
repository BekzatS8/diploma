package router

import (
	"log/slog"
	"time"

	"buhpro/internal/common/auth"
	"buhpro/internal/config"
	"buhpro/internal/http/handlers/system"
	"buhpro/internal/http/middleware"
	authmodule "buhpro/internal/modules/auth"
	profilemodule "buhpro/internal/modules/profile"
	"buhpro/internal/platform/metrics"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Deps struct {
	Config         config.Config
	Logger         *slog.Logger
	SystemHandlers *system.Handler
	JWTManager     *auth.JWTManager
	AuthHandler    *authmodule.Handler
	ProfileHandler *profilemodule.Handler
	Metrics        *metrics.Metrics
}

func New(deps Deps) *gin.Engine {
	gin.SetMode(mode(deps.Config.App.Env))
	r := gin.New()

	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery(deps.Logger))
	r.Use(middleware.RequestLogging(deps.Logger))
	r.Use(cors.New(cors.Config{
		AllowOrigins:     deps.Config.Server.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	if deps.Config.Metrics.Enabled && deps.Metrics != nil {
		r.Use(deps.Metrics.Middleware())
		r.GET(deps.Config.Metrics.Path, metrics.Handler())
	}

	r.GET("/healthz", deps.SystemHandlers.Healthz)
	r.GET("/readyz", deps.SystemHandlers.Readyz)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/ping", deps.SystemHandlers.Ping)

		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", deps.AuthHandler.Register)
			authGroup.POST("/login", deps.AuthHandler.Login)
			authGroup.POST("/refresh", deps.AuthHandler.Refresh)
			authGroup.Use(middleware.RequireAuth(deps.JWTManager))
			authGroup.POST("/logout", deps.AuthHandler.Logout)
			authGroup.GET("/me", deps.AuthHandler.Me)
		}

		profileGroup := v1.Group("/profile")
		profileGroup.Use(middleware.RequireAuth(deps.JWTManager))
		profileGroup.GET("", deps.ProfileHandler.Get)
		profileGroup.PATCH("", deps.ProfileHandler.Patch)
	}

	return r
}

func mode(env string) string {
	if env == "production" {
		return gin.ReleaseMode
	}
	return gin.DebugMode
}

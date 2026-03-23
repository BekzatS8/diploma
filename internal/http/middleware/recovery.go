package middleware

import (
	"log/slog"
	"net/http"

	"buhpro/internal/common/response"

	"github.com/gin-gonic/gin"
)

func Recovery(log *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, rec any) {
		log.Error("panic recovered",
			slog.Any("panic", rec),
			slog.String("path", c.Request.URL.Path),
			slog.String("method", c.Request.Method),
		)
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Internal server error")
	})
}

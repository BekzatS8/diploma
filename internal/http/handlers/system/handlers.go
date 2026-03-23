package system

import (
	"context"
	"net/http"
	"time"

	"buhpro/internal/common/response"

	"github.com/gin-gonic/gin"
)

type ReadinessChecker interface {
	Check(ctx context.Context) error
}

type Handler struct {
	checker ReadinessChecker
}

func NewHandler(checker ReadinessChecker) *Handler {
	return &Handler{checker: checker}
}

func (h *Handler) Healthz(c *gin.Context) {
	response.JSON(c, http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	if err := h.checker.Check(ctx); err != nil {
		response.JSONError(c, http.StatusServiceUnavailable, "not_ready", "Dependency check failed")
		return
	}

	response.JSON(c, http.StatusOK, gin.H{"status": "ready"})
}

func (h *Handler) Ping(c *gin.Context) {
	response.JSON(c, http.StatusOK, gin.H{"message": "pong"})
}

package middleware

import (
	"net"
	"net/http"
	"strings"

	"buhpro/internal/common/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func ValidateUUIDPathParams(names ...string) gin.HandlerFunc {
	nameSet := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			nameSet[name] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		for _, param := range c.Params {
			if _, ok := nameSet[param.Key]; !ok {
				continue
			}
			if _, err := uuid.Parse(strings.TrimSpace(param.Value)); err != nil {
				response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid UUID path parameter")
				return
			}
		}
		c.Next()
	}
}

func RequireInternalRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		parsed := net.ParseIP(c.ClientIP())
		if parsed == nil || (!parsed.IsLoopback() && !parsed.IsPrivate() && !parsed.IsLinkLocalUnicast()) {
			response.JSONError(c, http.StatusForbidden, "forbidden", "Metrics endpoint is internal-only")
			return
		}
		c.Next()
	}
}

package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const requestIDHeader = "X-Request-ID"

type contextKey string

const RequestIDKey contextKey = "request_id"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(requestIDHeader)
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Set("request_id", requestID)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), RequestIDKey, requestID))
		c.Writer.Header().Set(requestIDHeader, requestID)
		c.Next()
	}
}

func requestIDFromContext(r *http.Request) string {
	if value := r.Context().Value(RequestIDKey); value != nil {
		if requestID, ok := value.(string); ok && requestID != "" {
			return requestID
		}
	}
	if requestID := r.Header.Get(requestIDHeader); requestID != "" {
		return requestID
	}
	return uuid.NewString()
}

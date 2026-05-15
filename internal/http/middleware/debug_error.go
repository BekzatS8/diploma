package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type DebugErrorResponse struct {
	Success   bool           `json:"success"`
	Error     DebugErrorInfo `json:"error"`
	RequestID string         `json:"request_id"`
	Timestamp string         `json:"timestamp"`
	Path      string         `json:"path"`
	Method    string         `json:"method"`
	Status    int            `json:"status"`
}

type DebugErrorInfo struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	ErrorTrace string `json:"error_trace"`
	StackTrace string `json:"stack_trace"`
}

type bufferedResponseWriter struct {
	gin.ResponseWriter
	statusCode int
	body       bytes.Buffer
	size       int
	written    bool
}

func newBufferedResponseWriter(w gin.ResponseWriter) *bufferedResponseWriter {
	return &bufferedResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (bw *bufferedResponseWriter) WriteHeader(statusCode int) {
	if bw.written {
		return
	}
	bw.statusCode = statusCode
	bw.written = true
}

func (bw *bufferedResponseWriter) WriteHeaderNow() {
	if !bw.written {
		bw.WriteHeader(bw.statusCode)
	}
}

func (bw *bufferedResponseWriter) Write(data []byte) (int, error) {
	if !bw.written {
		bw.WriteHeader(http.StatusOK)
	}
	n, err := bw.body.Write(data)
	bw.size += n
	return n, err
}

func (bw *bufferedResponseWriter) WriteString(data string) (int, error) {
	return bw.Write([]byte(data))
}

func (bw *bufferedResponseWriter) Status() int {
	return bw.statusCode
}

func (bw *bufferedResponseWriter) Size() int {
	return bw.size
}

func (bw *bufferedResponseWriter) Written() bool {
	return bw.written
}

func (bw *bufferedResponseWriter) Flush() {
	bw.WriteHeaderNow()
}

func (bw *bufferedResponseWriter) FlushOriginal() {
	bw.ResponseWriter.WriteHeader(bw.statusCode)
	if bw.body.Len() > 0 {
		_, _ = bw.ResponseWriter.Write(bw.body.Bytes())
	}
}

func DebugErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/uploads/") {
			c.Next()
			return
		}

		originalWriter := c.Writer
		bufferedWriter := newBufferedResponseWriter(originalWriter)
		c.Writer = bufferedWriter

		defer func() {
			if panicErr := recover(); panicErr != nil {
				stackTrace := string(debug.Stack())
				errorTrace := fmt.Sprintf("%v", panicErr)
				status := http.StatusInternalServerError

				log.Error().
					Str("request_id", requestIDFromGin(c)).
					Str("method", c.Request.Method).
					Str("path", c.Request.URL.Path).
					Int("status", status).
					Str("error_trace", errorTrace).
					Str("stack_trace", stackTrace).
					Msg("panic recovered")

				writeDebugErrorJSON(originalWriter, c, status, errorTrace, stackTrace)
				c.Abort()
				return
			}

			if bufferedWriter.Status() >= http.StatusBadRequest {
				stackTrace := string(debug.Stack())
				errorTrace := strings.TrimSpace(bufferedWriter.body.String())
				status := bufferedWriter.Status()

				log.Error().
					Str("request_id", requestIDFromGin(c)).
					Str("method", c.Request.Method).
					Str("path", c.Request.URL.Path).
					Int("status", status).
					Str("error_trace", errorTrace).
					Str("stack_trace", stackTrace).
					Msg("http error intercepted")

				writeDebugErrorJSON(originalWriter, c, status, errorTrace, stackTrace)
				return
			}

			bufferedWriter.FlushOriginal()
		}()

		c.Next()
	}
}

func writeDebugErrorJSON(w gin.ResponseWriter, c *gin.Context, status int, errorTrace, stackTrace string) {
	if errorTrace == "" {
		errorTrace = http.StatusText(status)
	}

	message := http.StatusText(status)
	if message == "" {
		message = "HTTP Error"
	}

	response := DebugErrorResponse{
		Success: false,
		Error: DebugErrorInfo{
			Code:       fmt.Sprintf("HTTP_%d", status),
			Message:    message,
			ErrorTrace: errorTrace,
			StackTrace: stackTrace,
		},
		RequestID: requestIDFromGin(c),
		Timestamp: time.Now().Format(time.RFC3339),
		Path:      c.Request.URL.Path,
		Method:    c.Request.Method,
		Status:    status,
	}

	w.Header().Del("Content-Length")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set(requestIDHeader, response.RequestID)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func requestIDFromGin(c *gin.Context) string {
	if value, exists := c.Get("request_id"); exists {
		if requestID, ok := value.(string); ok && requestID != "" {
			return requestID
		}
	}
	return requestIDFromContext(c.Request)
}

package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDebugErrorMiddlewarePassesSuccessfulResponse(t *testing.T) {
	r := testRouter()
	r.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if got := w.Body.String(); got != "ok" {
		t.Fatalf("expected original body, got %q", got)
	}
}

func TestDebugErrorMiddlewareConvertsHTTPError(t *testing.T) {
	r := testRouter()
	r.GET("/legacy", func(c *gin.Context) {
		http.Error(c.Writer, "legacy failure", http.StatusInternalServerError)
	})

	resp := performRequest(r, http.MethodGet, "/legacy")

	if resp.Status != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", resp.Status)
	}
	if resp.Success {
		t.Fatal("expected success=false")
	}
	if resp.Error.Code != "HTTP_500" {
		t.Fatalf("expected HTTP_500, got %q", resp.Error.Code)
	}
	if resp.Error.ErrorTrace != "legacy failure" {
		t.Fatalf("expected original http.Error body, got %q", resp.Error.ErrorTrace)
	}
	if !strings.Contains(resp.Error.StackTrace, "runtime/debug.Stack") {
		t.Fatal("expected stack trace to include runtime/debug.Stack")
	}
}

func TestDebugErrorMiddlewareConvertsJSONError(t *testing.T) {
	r := testRouter()
	r.GET("/json", func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "bad_request"}})
	})

	resp := performRequest(r, http.MethodGet, "/json")

	if resp.Status != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Status)
	}
	if !strings.Contains(resp.Error.ErrorTrace, `"bad_request"`) {
		t.Fatalf("expected original JSON in error_trace, got %q", resp.Error.ErrorTrace)
	}
}

func TestDebugErrorMiddlewareConvertsPanic(t *testing.T) {
	r := testRouter()
	r.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	resp := performRequest(r, http.MethodGet, "/panic")

	if resp.Status != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", resp.Status)
	}
	if resp.Error.ErrorTrace != "boom" {
		t.Fatalf("expected panic value in error_trace, got %q", resp.Error.ErrorTrace)
	}
	if !strings.Contains(resp.Error.StackTrace, "panic") {
		t.Fatal("expected panic stack trace")
	}
}

func testRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.Use(DebugErrorMiddleware())
	return r
}

func performRequest(r *gin.Engine, method, path string) DebugErrorResponse {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	r.ServeHTTP(w, req)

	var resp DebugErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		panic(err)
	}
	return resp
}

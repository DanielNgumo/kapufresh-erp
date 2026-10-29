package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/kapufresh-erp/internal/middleware"
)

func TestCORSPreflightAndOriginAllowlist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.CORS([]string{"https://erp.example.com"}))
	router.GET("/resource", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	preflight := httptest.NewRecorder()
	preflightRequest := httptest.NewRequest(http.MethodOptions, "/resource", nil)
	preflightRequest.Header.Set("Origin", "https://erp.example.com")
	preflightRequest.Header.Set("Access-Control-Request-Method", "PATCH")
	preflightRequest.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	router.ServeHTTP(preflight, preflightRequest)
	if preflight.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", preflight.Code, http.StatusNoContent)
	}
	if got := preflight.Header().Get("Access-Control-Allow-Origin"); got != "https://erp.example.com" {
		t.Fatalf("allow-origin = %q, want allowed origin", got)
	}
	if got := preflight.Header().Get("Access-Control-Allow-Headers"); got != "Authorization, Content-Type" {
		t.Fatalf("allow-headers = %q", got)
	}
	if got := preflight.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("vary = %q, want Origin", got)
	}

	denied := httptest.NewRecorder()
	deniedRequest := httptest.NewRequest(http.MethodGet, "/resource", nil)
	deniedRequest.Header.Set("Origin", "https://untrusted.example")
	router.ServeHTTP(denied, deniedRequest)
	if denied.Code != http.StatusNoContent {
		t.Fatalf("request status = %d, want handler status %d", denied.Code, http.StatusNoContent)
	}
	if got := denied.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("untrusted origin received allow-origin %q", got)
	}

	allowed := httptest.NewRecorder()
	allowedRequest := httptest.NewRequest(http.MethodGet, "/resource", nil)
	allowedRequest.Header.Set("Origin", "https://erp.example.com")
	router.ServeHTTP(allowed, allowedRequest)
	if got := allowed.Header().Get("Access-Control-Allow-Origin"); got != "https://erp.example.com" {
		t.Fatalf("allowed request allow-origin = %q", got)
	}
}

func TestCORSHandlesUntrustedPreflightWithoutGrantingAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.CORS([]string{"https://erp.example.com"}))
	router.GET("/resource", func(c *gin.Context) { c.Status(http.StatusOK) })

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/resource", nil)
	request.Header.Set("Origin", "https://untrusted.example")
	request.Header.Set("Access-Control-Request-Method", "DELETE")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("untrusted origin received allow-origin %q", got)
	}
}

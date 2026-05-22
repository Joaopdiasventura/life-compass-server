package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddlewareAllowsConfiguredOrigin(t *testing.T) {
	handler := CORS(okHandler(), "http://localhost:5173, https://app.example.com")
	request := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("expected configured origin to be allowed")
	}
	if response.Header().Get("Access-Control-Allow-Methods") != corsAllowedMethods {
		t.Fatalf("expected allowed methods header")
	}
	if response.Header().Get("Access-Control-Allow-Headers") != corsAllowedHeaders {
		t.Fatalf("expected allowed headers header")
	}
}

func TestCORSMiddlewareAllowsAnyOriginWithWildcard(t *testing.T) {
	handler := CORS(okHandler(), "*")
	request := httptest.NewRequest(http.MethodOptions, "/transactions", nil)
	request.Header.Set("Origin", "https://frontend.example.com")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected wildcard origin to be allowed")
	}
}

func TestCORSMiddlewareRejectsPreflightForUnconfiguredOrigin(t *testing.T) {
	handler := CORS(okHandler(), "https://app.example.com")
	request := httptest.NewRequest(http.MethodOptions, "/transactions", nil)
	request.Header.Set("Origin", "https://other.example.com")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no CORS origin header")
	}
}

func TestCORSMiddlewareIsDisabledWithoutOrigins(t *testing.T) {
	handler := CORS(okHandler(), "")
	request := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no CORS origin header")
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

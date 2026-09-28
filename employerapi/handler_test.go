package employerapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthIsPublic(t *testing.T) {
	handler := NewHandler(nil, "")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestProtectedRouteRequiresBearerToken(t *testing.T) {
	handler := NewHandler(nil, "")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

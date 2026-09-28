package employerapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthIsPublic(t *testing.T) {
	handler := NewHandler(nil, "secret")
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestEmployerAPIRequiresConfiguredKey(t *testing.T) {
	handler := NewHandler(nil, "secret")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/employer/reference-data", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

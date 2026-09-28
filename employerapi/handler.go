package employerapi

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	store         *Store
	apiKey        string
	allowedOrigin string
}

func NewHandler(store *Store, apiKey string, allowedOrigin ...string) http.Handler {
	origin := ""
	if len(allowedOrigin) > 0 {
		origin = strings.TrimSpace(allowedOrigin[0])
	}
	handler := &Handler{store: store, apiKey: strings.TrimSpace(apiKey), allowedOrigin: origin}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handler.health)
	mux.Handle("GET /api/v1/employer/reference-data", handler.authorize(http.HandlerFunc(handler.referenceData)))
	mux.Handle("POST /api/v1/employer/catalog", handler.authorize(http.HandlerFunc(handler.createCatalog)))
	mux.Handle("GET /api/v1/employer/catalog/{companyID}", handler.authorize(http.HandlerFunc(handler.getCatalog)))
	mux.Handle("PUT /api/v1/employer/catalog/{companyID}", handler.authorize(http.HandlerFunc(handler.updateCatalog)))
	mux.Handle("PATCH /api/v1/employer/opportunities/{opportunityID}/status", handler.authorize(http.HandlerFunc(handler.setOpportunityStatus)))
	mux.Handle("GET /api/v1/employer/companies/{companyID}/applications", handler.authorize(http.HandlerFunc(handler.listApplications)))
	mux.Handle("PATCH /api/v1/employer/applications/{applicationID}", handler.authorize(http.HandlerFunc(handler.saveFeedback)))
	return recoverPanic(handler.withCORS(mux))
}

func (h *Handler) listApplications(writer http.ResponseWriter, request *http.Request) {
	companyID, err := positivePathID(request, "companyID")
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	applications, err := h.store.ListEmployerApplications(request.Context(), companyID)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"applications": applications})
}

func (h *Handler) saveFeedback(writer http.ResponseWriter, request *http.Request) {
	applicationID, err := positivePathID(request, "applicationID")
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var input EmployerFeedbackInput
	if err := decodeJSON(writer, request, &input); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if !contains([]string{"under_review", "interview", "accepted", "rejected"}, input.Status) {
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"error": "status must be under_review, interview, accepted or rejected"})
		return
	}
	if strings.TrimSpace(input.Message) == "" {
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"error": "message is required"})
		return
	}
	feedback, err := h.store.SaveEmployerFeedback(request.Context(), applicationID, input)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, feedback)
}

func (h *Handler) health(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) referenceData(writer http.ResponseWriter, request *http.Request) {
	response, err := h.store.ReferenceData(request.Context())
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, response)
}

func (h *Handler) createCatalog(writer http.ResponseWriter, request *http.Request) {
	var input CatalogRequest
	if err := decodeJSON(writer, request, &input); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := validateCatalog(input, 0); err != nil {
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	response, err := h.store.SaveCatalog(request.Context(), 0, input)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, response)
}

func (h *Handler) updateCatalog(writer http.ResponseWriter, request *http.Request) {
	companyID, err := positivePathID(request, "companyID")
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var input CatalogRequest
	if err := decodeJSON(writer, request, &input); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := validateCatalog(input, companyID); err != nil {
		writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	response, err := h.store.SaveCatalog(request.Context(), companyID, input)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, response)
}

func (h *Handler) getCatalog(writer http.ResponseWriter, request *http.Request) {
	companyID, err := positivePathID(request, "companyID")
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	response, err := h.store.GetCatalog(request.Context(), companyID)
	if err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, response)
}

func (h *Handler) setOpportunityStatus(writer http.ResponseWriter, request *http.Request) {
	opportunityID, err := positivePathID(request, "opportunityID")
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var input opportunityStatusRequest
	if err := decodeJSON(writer, request, &input); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := h.store.SetOpportunityStatus(request.Context(), opportunityID, input.IsActive); err != nil {
		writeError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"id": opportunityID, "is_active": input.IsActive})
}

func (h *Handler) authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if h.apiKey == "" {
			next.ServeHTTP(writer, request)
			return
		}
		provided := strings.TrimSpace(request.Header.Get("X-API-Key"))
		if authorization := strings.TrimSpace(request.Header.Get("Authorization")); strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
			provided = strings.TrimSpace(authorization[len("Bearer "):])
		}
		if subtle.ConstantTimeCompare([]byte(provided), []byte(h.apiKey)) != 1 {
			writeJSON(writer, http.StatusUnauthorized, map[string]string{"error": "invalid employer API key"})
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (h *Handler) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if h.allowedOrigin != "" && origin != "" && (h.allowedOrigin == "*" || origin == h.allowedOrigin) {
			writer.Header().Set("Access-Control-Allow-Origin", h.allowedOrigin)
			writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Key")
			writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, OPTIONS")
			writer.Header().Set("Vary", "Origin")
			if request.Method == http.MethodOptions {
				writer.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(writer, request)
	})
}

func positivePathID(request *http.Request, name string) (int64, error) {
	value, err := strconv.ParseInt(request.PathValue(name), 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return value, nil
}

func decodeJSON(writer http.ResponseWriter, request *http.Request, destination any) error {
	request.Body = http.MaxBytesReader(writer, request.Body, 2<<20)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("request body must contain one JSON object")
	}
	return nil
}

func writeError(writer http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, ErrConflict):
		status = http.StatusConflict
	}
	message := err.Error()
	if status == http.StatusInternalServerError {
		message = "internal server error"
	}
	writeJSON(writer, status, map[string]string{"error": message})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if recover() != nil {
				writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			}
		}()
		next.ServeHTTP(writer, request)
	})
}

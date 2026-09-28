package employerapi

import (
	"context"
	"efr_bot/models"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type accountContextKey struct{}
type Handler struct {
	store         *Store
	allowedOrigin string
}

func NewHandler(store *Store, bootstrapToken string, origins ...string) http.Handler {
	origin := ""
	if len(origins) > 0 {
		origin = strings.TrimSpace(origins[0])
	}
	h := &Handler{store: store, allowedOrigin: origin}
	if store != nil {
		if err := store.BootstrapAdmin(context.Background(), bootstrapToken); err != nil {
			panic(fmt.Sprintf("bootstrap API admin: %v", err))
		}
	}
	m := http.NewServeMux()
	m.HandleFunc("GET /healthz", h.health)
	m.Handle("GET /api/v1/me", h.auth(http.HandlerFunc(h.me)))
	m.Handle("GET /api/v1/reference-data", h.auth(http.HandlerFunc(h.reference)))
	m.Handle("GET /api/v1/education-programs", h.auth(http.HandlerFunc(h.listEducationPrograms)))
	// Employer cabinet: company is always read from the authenticated account.
	m.Handle("GET /api/v1/employer/company", h.role(roleEmployer, http.HandlerFunc(h.employerCompany)))
	m.Handle("PUT /api/v1/employer/company", h.role(roleEmployer, http.HandlerFunc(h.updateEmployerCompany)))
	m.Handle("GET /api/v1/employer/directions", h.role(roleEmployer, http.HandlerFunc(h.listEmployerDirections)))
	m.Handle("POST /api/v1/employer/directions", h.role(roleEmployer, http.HandlerFunc(h.createEmployerDirection)))
	m.Handle("PATCH /api/v1/employer/directions/{directionID}", h.role(roleEmployer, http.HandlerFunc(h.updateEmployerDirection)))
	m.Handle("PUT /api/v1/employer/directions/{directionID}/education-programs", h.role(roleEmployer, http.HandlerFunc(h.replaceEmployerPrograms)))
	m.Handle("PUT /api/v1/employer/directions/{directionID}/interest-tags", h.role(roleEmployer, http.HandlerFunc(h.replaceEmployerTags)))
	m.Handle("GET /api/v1/employer/opportunities", h.role(roleEmployer, http.HandlerFunc(h.listEmployerOpportunities)))
	m.Handle("POST /api/v1/employer/opportunities", h.role(roleEmployer, http.HandlerFunc(h.createEmployerOpportunity)))
	m.Handle("PATCH /api/v1/employer/opportunities/{opportunityID}", h.role(roleEmployer, http.HandlerFunc(h.updateEmployerOpportunity)))
	m.Handle("GET /api/v1/employer/applications", h.role(roleEmployer, http.HandlerFunc(h.listEmployerApplications)))
	m.Handle("PATCH /api/v1/employer/applications/{applicationID}", h.role(roleEmployer, http.HandlerFunc(h.saveFeedback)))
	// Backoffice endpoints own all mutable catalog data.
	m.Handle("GET /api/v1/admin/accounts", h.role(roleAdmin, http.HandlerFunc(h.listAccounts)))
	m.Handle("POST /api/v1/admin/accounts", h.role(roleAdmin, http.HandlerFunc(h.createAccount)))
	m.Handle("PATCH /api/v1/admin/accounts/{accountID}", h.role(roleAdmin, http.HandlerFunc(h.updateAccount)))
	m.Handle("POST /api/v1/admin/accounts/{accountID}/rotate-token", h.role(roleAdmin, http.HandlerFunc(h.rotateToken)))
	m.Handle("GET /api/v1/admin/companies", h.role(roleAdmin, http.HandlerFunc(h.listCompanies)))
	m.Handle("POST /api/v1/admin/companies", h.role(roleAdmin, http.HandlerFunc(h.createCompany)))
	m.Handle("PUT /api/v1/admin/companies/{companyID}", h.role(roleAdmin, http.HandlerFunc(h.updateCompany)))
	m.Handle("PATCH /api/v1/admin/companies/{companyID}/archive", h.role(roleAdmin, http.HandlerFunc(h.archiveCompany)))
	m.Handle("POST /api/v1/admin/companies/{companyID}/directions", h.role(roleAdmin, http.HandlerFunc(h.adminCreateDirection)))
	m.Handle("PATCH /api/v1/admin/companies/{companyID}/directions/{directionID}", h.role(roleAdmin, http.HandlerFunc(h.adminUpdateDirection)))
	m.Handle("PATCH /api/v1/admin/companies/{companyID}/directions/{directionID}/archive", h.role(roleAdmin, http.HandlerFunc(h.archiveDirection)))
	m.Handle("PUT /api/v1/admin/companies/{companyID}/directions/{directionID}/education-programs", h.role(roleAdmin, http.HandlerFunc(h.adminReplacePrograms)))
	m.Handle("PUT /api/v1/admin/companies/{companyID}/directions/{directionID}/interest-tags", h.role(roleAdmin, http.HandlerFunc(h.adminReplaceTags)))
	m.Handle("POST /api/v1/admin/companies/{companyID}/opportunities", h.role(roleAdmin, http.HandlerFunc(h.adminCreateOpportunity)))
	m.Handle("PATCH /api/v1/admin/companies/{companyID}/opportunities/{opportunityID}", h.role(roleAdmin, http.HandlerFunc(h.adminUpdateOpportunity)))
	m.Handle("PATCH /api/v1/admin/companies/{companyID}/opportunities/{opportunityID}/archive", h.role(roleAdmin, http.HandlerFunc(h.archiveOpportunity)))
	m.Handle("GET /api/v1/admin/universities", h.role(roleAdmin, http.HandlerFunc(h.listUniversities)))
	m.Handle("POST /api/v1/admin/universities", h.role(roleAdmin, http.HandlerFunc(h.createUniversity)))
	m.Handle("PUT /api/v1/admin/universities/{universityID}", h.role(roleAdmin, http.HandlerFunc(h.updateUniversity)))
	m.Handle("PATCH /api/v1/admin/universities/{universityID}/archive", h.role(roleAdmin, http.HandlerFunc(h.archiveUniversity)))
	m.Handle("GET /api/v1/admin/education-programs", h.role(roleAdmin, http.HandlerFunc(h.listAdminPrograms)))
	m.Handle("POST /api/v1/admin/education-programs", h.role(roleAdmin, http.HandlerFunc(h.createProgram)))
	m.Handle("PUT /api/v1/admin/education-programs/{programID}", h.role(roleAdmin, http.HandlerFunc(h.updateProgram)))
	m.Handle("PATCH /api/v1/admin/education-programs/{programID}/archive", h.role(roleAdmin, http.HandlerFunc(h.archiveProgram)))
	m.Handle("GET /api/v1/admin/admission-scores", h.role(roleAdmin, http.HandlerFunc(h.listScores)))
	m.Handle("PUT /api/v1/admin/admission-scores", h.role(roleAdmin, http.HandlerFunc(h.saveScore)))
	m.Handle("PUT /api/v1/admin/admission-rules", h.role(roleAdmin, http.HandlerFunc(h.saveRule)))
	m.Handle("GET /api/v1/admin/admission-rules", h.role(roleAdmin, http.HandlerFunc(h.listRules)))
	m.Handle("GET /api/v1/admin/exam-combinations", h.role(roleAdmin, http.HandlerFunc(h.listCombinations)))
	m.Handle("POST /api/v1/admin/exam-combinations", h.role(roleAdmin, http.HandlerFunc(h.createExamCombination)))
	m.Handle("PUT /api/v1/admin/exam-combinations/{combinationID}", h.role(roleAdmin, http.HandlerFunc(h.updateExamCombination)))
	m.Handle("POST /api/v1/admin/roadmap-templates", h.role(roleAdmin, http.HandlerFunc(h.createRoadmapTemplate)))
	m.Handle("GET /api/v1/admin/roadmap-templates", h.role(roleAdmin, http.HandlerFunc(h.listTemplates)))
	return recoverPanic(h.withCORS(m))
}
func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
func (h *Handler) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(strings.ToLower(v), "bearer ") {
			writeJSON(w, 401, map[string]string{"error": "Bearer token is required"})
			return
		}
		if h.store == nil {
			writeJSON(w, 500, map[string]string{"error": "API store is not configured"})
			return
		}
		a, e := h.store.Authenticate(r.Context(), strings.TrimSpace(v[7:]))
		if e != nil {
			writeJSON(w, 401, map[string]string{"error": "invalid API token"})
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), accountContextKey{}, a)))
	})
}
func (h *Handler) role(role string, next http.Handler) http.Handler {
	return h.auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if account(r).Role != role {
			writeJSON(w, 403, map[string]string{"error": "insufficient API role"})
			return
		}
		next.ServeHTTP(w, r)
	}))
}
func account(r *http.Request) models.APIAccount {
	a, _ := r.Context().Value(accountContextKey{}).(models.APIAccount)
	return a
}
func companyID(r *http.Request) (int64, error) {
	a := account(r)
	if a.CompanyID == nil || *a.CompanyID <= 0 {
		return 0, fmt.Errorf("employer account has no company")
	}
	return *a.CompanyID, nil
}
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, accountView(account(r), ""))
}
func (h *Handler) reference(w http.ResponseWriter, r *http.Request) {
	x, e := h.store.ReferenceData(r.Context())
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, x)
}
func (h *Handler) listEducationPrograms(w http.ResponseWriter, r *http.Request) {
	x, e := h.store.ListPrograms(r.Context())
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"education_programs": x})
}
func (h *Handler) employerCompany(w http.ResponseWriter, r *http.Request) {
	id, e := companyID(r)
	if e != nil {
		writeError(w, e)
		return
	}
	x, e := h.store.GetCompany(r.Context(), id)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, x)
}
func (h *Handler) updateEmployerCompany(w http.ResponseWriter, r *http.Request) {
	id, e := companyID(r)
	if e != nil {
		writeError(w, e)
		return
	}
	var x CompanyInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.SaveCompany(r.Context(), id, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, out)
}
func (h *Handler) listEmployerDirections(w http.ResponseWriter, r *http.Request) {
	id, e := companyID(r)
	if e != nil {
		writeError(w, e)
		return
	}
	x, e := h.store.ListDirections(r.Context(), id)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"directions": x})
}
func (h *Handler) createEmployerDirection(w http.ResponseWriter, r *http.Request) {
	id, e := companyID(r)
	if e != nil {
		writeError(w, e)
		return
	}
	var x CareerDirectionInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.SaveDirection(r.Context(), id, 0, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 201, out)
}
func (h *Handler) updateEmployerDirection(w http.ResponseWriter, r *http.Request) {
	cid, e := companyID(r)
	if e != nil {
		writeError(w, e)
		return
	}
	id, e := pathID(r, "directionID")
	if e != nil {
		bad(w, e)
		return
	}
	var x CareerDirectionInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.SaveDirection(r.Context(), cid, id, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, out)
}
func (h *Handler) replaceEmployerPrograms(w http.ResponseWriter, r *http.Request) {
	cid, e := companyID(r)
	if e != nil {
		writeError(w, e)
		return
	}
	id, e := pathID(r, "directionID")
	if e != nil {
		bad(w, e)
		return
	}
	var x DirectionProgramsInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	if e = h.store.ReplaceDirectionPrograms(r.Context(), id, cid, x.EducationProgramIDs); e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 204, nil)
}
func (h *Handler) replaceEmployerTags(w http.ResponseWriter, r *http.Request) {
	cid, e := companyID(r)
	if e != nil {
		writeError(w, e)
		return
	}
	id, e := pathID(r, "directionID")
	if e != nil {
		bad(w, e)
		return
	}
	var x DirectionTagsInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	if e = h.store.ReplaceDirectionTags(r.Context(), id, cid, x.InterestTags); e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 204, nil)
}
func (h *Handler) listEmployerOpportunities(w http.ResponseWriter, r *http.Request) {
	cid, e := companyID(r)
	if e != nil {
		writeError(w, e)
		return
	}
	x, e := h.store.ListOpportunities(r.Context(), cid)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"opportunities": x})
}
func (h *Handler) createEmployerOpportunity(w http.ResponseWriter, r *http.Request) {
	cid, e := companyID(r)
	if e != nil {
		writeError(w, e)
		return
	}
	var x OpportunityInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.SaveOpportunity(r.Context(), cid, 0, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 201, out)
}
func (h *Handler) updateEmployerOpportunity(w http.ResponseWriter, r *http.Request) {
	cid, e := companyID(r)
	if e != nil {
		writeError(w, e)
		return
	}
	id, e := pathID(r, "opportunityID")
	if e != nil {
		bad(w, e)
		return
	}
	var x OpportunityInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.SaveOpportunity(r.Context(), cid, id, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, out)
}
func (h *Handler) listEmployerApplications(w http.ResponseWriter, r *http.Request) {
	cid, e := companyID(r)
	if e != nil {
		writeError(w, e)
		return
	}
	x, e := h.store.ListEmployerApplications(r.Context(), cid)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"applications": x})
}
func (h *Handler) saveFeedback(w http.ResponseWriter, r *http.Request) {
	cid, e := companyID(r)
	if e != nil {
		writeError(w, e)
		return
	}
	id, e := pathID(r, "applicationID")
	if e != nil {
		bad(w, e)
		return
	}
	var x EmployerFeedbackInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	if x.Message == "" {
		bad(w, fmt.Errorf("message is required"))
		return
	}
	out, e := h.store.SaveEmployerFeedback(r.Context(), cid, id, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, out)
}
func (h *Handler) listAccounts(w http.ResponseWriter, r *http.Request) {
	x, e := h.store.ListAccounts(r.Context())
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"accounts": x})
}
func (h *Handler) createAccount(w http.ResponseWriter, r *http.Request) {
	var x CreateAccountInput
	if e := decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.CreateAccount(r.Context(), x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 201, out)
}
func (h *Handler) updateAccount(w http.ResponseWriter, r *http.Request) {
	id, e := pathID(r, "accountID")
	if e != nil {
		bad(w, e)
		return
	}
	var x UpdateAccountInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.UpdateAccount(r.Context(), id, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, out)
}
func (h *Handler) rotateToken(w http.ResponseWriter, r *http.Request) {
	id, e := pathID(r, "accountID")
	if e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.RotateAccountToken(r.Context(), id)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, out)
}
func (h *Handler) listCompanies(w http.ResponseWriter, r *http.Request) {
	x, e := h.store.ListCompanies(r.Context())
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"companies": x})
}
func (h *Handler) createCompany(w http.ResponseWriter, r *http.Request) {
	var x CompanyInput
	if e := decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.SaveCompany(r.Context(), 0, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 201, out)
}
func (h *Handler) updateCompany(w http.ResponseWriter, r *http.Request) {
	id, e := pathID(r, "companyID")
	if e != nil {
		bad(w, e)
		return
	}
	var x CompanyInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.SaveCompany(r.Context(), id, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, out)
}
func archiveRequest(w http.ResponseWriter, r *http.Request, store *Store, entity, pathName string) {
	id, e := pathID(r, pathName)
	if e != nil {
		bad(w, e)
		return
	}
	var input struct {
		IsActive bool `json:"is_active"`
	}
	if e = decodeJSON(w, r, &input); e != nil {
		bad(w, e)
		return
	}
	if e = store.ArchiveEntity(r.Context(), entity, id, input.IsActive); e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 204, nil)
}
func (h *Handler) archiveCompany(w http.ResponseWriter, r *http.Request) {
	archiveRequest(w, r, h.store, "company", "companyID")
}
func (h *Handler) archiveDirection(w http.ResponseWriter, r *http.Request) {
	archiveRequest(w, r, h.store, "direction", "directionID")
}
func (h *Handler) archiveOpportunity(w http.ResponseWriter, r *http.Request) {
	archiveRequest(w, r, h.store, "opportunity", "opportunityID")
}
func (h *Handler) archiveUniversity(w http.ResponseWriter, r *http.Request) {
	archiveRequest(w, r, h.store, "university", "universityID")
}
func (h *Handler) archiveProgram(w http.ResponseWriter, r *http.Request) {
	archiveRequest(w, r, h.store, "program", "programID")
}
func (h *Handler) adminCreateDirection(w http.ResponseWriter, r *http.Request) {
	cid, e := pathID(r, "companyID")
	if e != nil {
		bad(w, e)
		return
	}
	var x CareerDirectionInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.SaveDirection(r.Context(), cid, 0, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}
func (h *Handler) adminUpdateDirection(w http.ResponseWriter, r *http.Request) {
	cid, e := pathID(r, "companyID")
	if e != nil {
		bad(w, e)
		return
	}
	id, e := pathID(r, "directionID")
	if e != nil {
		bad(w, e)
		return
	}
	var x CareerDirectionInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.SaveDirection(r.Context(), cid, id, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, out)
}
func (h *Handler) adminReplacePrograms(w http.ResponseWriter, r *http.Request) {
	cid, e := pathID(r, "companyID")
	if e != nil {
		bad(w, e)
		return
	}
	id, e := pathID(r, "directionID")
	if e != nil {
		bad(w, e)
		return
	}
	var x DirectionProgramsInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	if e = h.store.ReplaceDirectionPrograms(r.Context(), id, cid, x.EducationProgramIDs); e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 204, nil)
}
func (h *Handler) adminReplaceTags(w http.ResponseWriter, r *http.Request) {
	cid, e := pathID(r, "companyID")
	if e != nil {
		bad(w, e)
		return
	}
	id, e := pathID(r, "directionID")
	if e != nil {
		bad(w, e)
		return
	}
	var x DirectionTagsInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	if e = h.store.ReplaceDirectionTags(r.Context(), id, cid, x.InterestTags); e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 204, nil)
}
func (h *Handler) adminCreateOpportunity(w http.ResponseWriter, r *http.Request) {
	cid, e := pathID(r, "companyID")
	if e != nil {
		bad(w, e)
		return
	}
	var x OpportunityInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.SaveOpportunity(r.Context(), cid, 0, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}
func (h *Handler) adminUpdateOpportunity(w http.ResponseWriter, r *http.Request) {
	cid, e := pathID(r, "companyID")
	if e != nil {
		bad(w, e)
		return
	}
	id, e := pathID(r, "opportunityID")
	if e != nil {
		bad(w, e)
		return
	}
	var x OpportunityInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.SaveOpportunity(r.Context(), cid, id, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, out)
}
func (h *Handler) createUniversity(w http.ResponseWriter, r *http.Request) {
	var x UniversityInput
	if e := decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.SaveUniversity(r.Context(), 0, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 201, out)
}
func (h *Handler) updateUniversity(w http.ResponseWriter, r *http.Request) {
	id, e := pathID(r, "universityID")
	if e != nil {
		bad(w, e)
		return
	}
	var x UniversityInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.SaveUniversity(r.Context(), id, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, out)
}
func (h *Handler) listUniversities(w http.ResponseWriter, r *http.Request) {
	x, e := h.store.ListUniversities(r.Context())
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"universities": x})
}
func (h *Handler) listAdminPrograms(w http.ResponseWriter, r *http.Request) {
	x, e := h.store.ListPrograms(r.Context())
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"education_programs": x})
}
func (h *Handler) listScores(w http.ResponseWriter, r *http.Request) {
	x, e := h.store.ListAdmissionScores(r.Context())
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"admission_scores": x})
}
func (h *Handler) listRules(w http.ResponseWriter, r *http.Request) {
	x, e := h.store.ListAdmissionRules(r.Context())
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"admission_rules": x})
}
func (h *Handler) listCombinations(w http.ResponseWriter, r *http.Request) {
	x, e := h.store.ListExamCombinations(r.Context())
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"exam_combinations": x})
}
func (h *Handler) listTemplates(w http.ResponseWriter, r *http.Request) {
	x, e := h.store.ListRoadmapTemplates(r.Context())
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"roadmap_templates": x})
}
func (h *Handler) createProgram(w http.ResponseWriter, r *http.Request) {
	var x EducationProgramInput
	if e := decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.SaveProgram(r.Context(), 0, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 201, out)
}
func (h *Handler) updateProgram(w http.ResponseWriter, r *http.Request) {
	id, e := pathID(r, "programID")
	if e != nil {
		bad(w, e)
		return
	}
	var x EducationProgramInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.SaveProgram(r.Context(), id, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 200, out)
}
func (h *Handler) saveScore(w http.ResponseWriter, r *http.Request) {
	var x AdmissionScoreInput
	if e := decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	if e := h.store.SaveAdmissionScore(r.Context(), x); e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 204, nil)
}
func (h *Handler) saveRule(w http.ResponseWriter, r *http.Request) {
	var x AdmissionRuleInput
	if e := decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	if e := h.store.SaveAdmissionRule(r.Context(), x); e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, 204, nil)
}
func (h *Handler) createExamCombination(w http.ResponseWriter, r *http.Request) {
	var x ExamCombinationInput
	if e := decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.SaveExamCombination(r.Context(), 0, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}
func (h *Handler) updateExamCombination(w http.ResponseWriter, r *http.Request) {
	id, e := pathID(r, "combinationID")
	if e != nil {
		bad(w, e)
		return
	}
	var x ExamCombinationInput
	if e = decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	out, e := h.store.SaveExamCombination(r.Context(), id, x)
	if e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
func (h *Handler) createRoadmapTemplate(w http.ResponseWriter, r *http.Request) {
	var x RoadmapTemplateInput
	if e := decodeJSON(w, r, &x); e != nil {
		bad(w, e)
		return
	}
	if e := h.store.SaveRoadmapTemplate(r.Context(), x); e != nil {
		writeError(w, e)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}
func pathID(r *http.Request, n string) (int64, error) {
	x, e := strconv.ParseInt(r.PathValue(n), 10, 64)
	if e != nil || x <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", n)
	}
	return x, nil
}
func decodeJSON(w http.ResponseWriter, r *http.Request, d any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	x := json.NewDecoder(r.Body)
	x.DisallowUnknownFields()
	if e := x.Decode(d); e != nil {
		return fmt.Errorf("invalid JSON: %w", e)
	}
	if e := x.Decode(&struct{}{}); !errors.Is(e, io.EOF) {
		return fmt.Errorf("request body must contain one JSON object")
	}
	return nil
}
func bad(w http.ResponseWriter, e error) { writeJSON(w, 400, map[string]string{"error": e.Error()}) }
func writeError(w http.ResponseWriter, e error) {
	code := 500
	if errors.Is(e, ErrNotFound) {
		code = 404
	}
	if errors.Is(e, ErrConflict) {
		code = 409
	}
	if errors.Is(e, ErrForbidden) {
		code = 403
	}
	msg := e.Error()
	if code == 500 {
		msg = "internal server error"
	}
	writeJSON(w, code, map[string]string{"error": msg})
}
func writeJSON(w http.ResponseWriter, c int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(c)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}
func (h *Handler) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o := r.Header.Get("Origin")
		if h.allowedOrigin != "" && o != "" && (h.allowedOrigin == "*" || h.allowedOrigin == o) {
			w.Header().Set("Access-Control-Allow-Origin", h.allowedOrigin)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, OPTIONS")
			if r.Method == "OPTIONS" {
				w.WriteHeader(204)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover() != nil {
				writeJSON(w, 500, map[string]string{"error": "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

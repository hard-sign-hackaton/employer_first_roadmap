package employerapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"efr_bot/database"
	"efr_bot/models"
	"efr_bot/repositories"
)

func TestRBACAndEmployerCatalogIsolation(t *testing.T) {
	if os.Getenv("RUN_POSTGRES_INTEGRATION") != "1" {
		t.Skip("PostgreSQL integration test is disabled")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, err := database.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback().Error })
	store := NewStore(tx)
	companyA, err := store.SaveCompany(ctx, 0, CompanyInput{Name: "RBAC A"})
	if err != nil {
		t.Fatal(err)
	}
	companyB, err := store.SaveCompany(ctx, 0, CompanyInput{Name: "RBAC B"})
	if err != nil {
		t.Fatal(err)
	}
	adminToken := "admin-test-token"
	if err := tx.Create(&models.APIAccount{Login: "admin-rbac", TokenHash: tokenHash(adminToken), Role: roleAdmin, IsActive: true}).Error; err != nil {
		t.Fatal(err)
	}
	employer, err := store.CreateAccount(ctx, CreateAccountInput{Login: "a-manager", Role: roleEmployer, CompanyID: &companyA.ID})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(store, "")

	assertAPIStatus(t, h, http.MethodGet, "/api/v1/admin/companies", employer.Token, nil, http.StatusForbidden)
	assertAPIStatus(t, h, http.MethodPut, "/api/v1/employer/company", employer.Token, []byte(`{"name":"RBAC A","description":"updated","website_url":""}`), http.StatusOK)
	assertAPIStatus(t, h, http.MethodPut, "/api/v1/admin/companies/"+fmt.Sprint(companyB.ID), employer.Token, []byte(`{"name":"hack"}`), http.StatusForbidden)
	assertAPIStatus(t, h, http.MethodPost, "/api/v1/admin/accounts", adminToken, []byte(`{"login":"b-manager","role":"employer","company_id":`+fmt.Sprint(companyB.ID)+`}`), http.StatusCreated)
}

func TestEmployerCanLinkOnlyExistingEducationProgram(t *testing.T) {
	if os.Getenv("RUN_POSTGRES_INTEGRATION") != "1" {
		t.Skip("PostgreSQL integration test is disabled")
	}
	ctx := context.Background()
	db, err := database.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	tx := db.Begin()
	t.Cleanup(func() { _ = tx.Rollback().Error })
	store := NewStore(tx)
	c, err := store.SaveCompany(ctx, 0, CompanyInput{Name: "Links Co"})
	if err != nil {
		t.Fatal(err)
	}
	employer, err := store.CreateAccount(ctx, CreateAccountInput{Login: "links", Role: roleEmployer, CompanyID: &c.ID})
	if err != nil {
		t.Fatal(err)
	}
	d, err := store.SaveDirection(ctx, c.ID, 0, CareerDirectionInput{Name: "Developer"})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(store, "")
	assertAPIStatus(t, h, http.MethodPut, "/api/v1/employer/directions/"+fmt.Sprint(d.ID)+"/education-programs", employer.Token, []byte(`{"education_program_ids":[999999]}`), http.StatusNotFound)
	assertAPIStatus(t, h, http.MethodPost, "/api/v1/admin/education-programs", employer.Token, []byte(`{"university_id":1,"name":"forbidden"}`), http.StatusForbidden)
}

func TestAdminCanPopulateEducationCatalog(t *testing.T) {
	if os.Getenv("RUN_POSTGRES_INTEGRATION") != "1" {
		t.Skip("PostgreSQL integration test is disabled")
	}
	ctx := context.Background()
	db, err := database.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	tx := db.Begin()
	t.Cleanup(func() { _ = tx.Rollback().Error })
	region := models.Region{Name: "Admin region"}
	subject := models.ExamSubject{Name: "Admin exam"}
	if err := tx.Create(&region).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Create(&subject).Error; err != nil {
		t.Fatal(err)
	}
	adminToken := "admin-catalog-token"
	if err := tx.Create(&models.APIAccount{Login: "admin-catalog", TokenHash: tokenHash(adminToken), Role: roleAdmin, IsActive: true}).Error; err != nil {
		t.Fatal(err)
	}
	h := NewHandler(NewStore(tx), "")
	universityResponse := request(t, h, http.MethodPost, "/api/v1/admin/universities", adminToken, []byte(`{"region_id":`+fmt.Sprint(region.ID)+`,"name":"Admin University","description":"","website_url":"https://example.test"}`))
	if universityResponse.Code != http.StatusCreated {
		t.Fatalf("university: %d %s", universityResponse.Code, universityResponse.Body.String())
	}
	var university UniversityResponse
	if err := json.NewDecoder(universityResponse.Body).Decode(&university); err != nil {
		t.Fatal(err)
	}
	programResponse := request(t, h, http.MethodPost, "/api/v1/admin/education-programs", adminToken, []byte(`{"university_id":`+fmt.Sprint(university.ID)+`,"code":"01.03.02","name":"Admin Program","description":""}`))
	if programResponse.Code != http.StatusCreated {
		t.Fatalf("program: %d %s", programResponse.Code, programResponse.Body.String())
	}
	var program EducationProgramResponse
	if err := json.NewDecoder(programResponse.Body).Decode(&program); err != nil {
		t.Fatal(err)
	}
	assertAPIStatus(t, h, http.MethodPost, "/api/v1/admin/exam-combinations", adminToken, []byte(`{"education_program_id":`+fmt.Sprint(program.ID)+`,"admission_year":2026,"items":[{"exam_subject_id":`+fmt.Sprint(subject.ID)+`}]}`), http.StatusCreated)
	assertAPIStatus(t, h, http.MethodPut, "/api/v1/admin/admission-scores", adminToken, []byte(`{"education_program_id":`+fmt.Sprint(program.ID)+`,"admission_year":2025,"budget_passing_score":250}`), http.StatusNoContent)
}

func TestAdminArchiveHidesCompanyFromNewBotSearch(t *testing.T) {
	if os.Getenv("RUN_POSTGRES_INTEGRATION") != "1" {
		t.Skip("PostgreSQL integration test is disabled")
	}
	ctx := context.Background()
	db, err := database.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	tx := db.Begin()
	t.Cleanup(func() { _ = tx.Rollback().Error })
	store := NewStore(tx)
	company, err := store.SaveCompany(ctx, 0, CompanyInput{Name: "Archive Me Co"})
	if err != nil {
		t.Fatal(err)
	}
	token := "archive-admin"
	if err := tx.Create(&models.APIAccount{Login: "archive-admin", TokenHash: tokenHash(token), Role: roleAdmin, IsActive: true}).Error; err != nil {
		t.Fatal(err)
	}
	h := NewHandler(store, "")
	assertAPIStatus(t, h, http.MethodGet, "/api/v1/admin/companies", token, nil, http.StatusOK)
	assertAPIStatus(t, h, http.MethodPatch, "/api/v1/admin/companies/"+fmt.Sprint(company.ID)+"/archive", token, []byte(`{"is_active":false}`), http.StatusNoContent)
	found, err := repositories.NewGormCareerRepository(tx).FindCompanies(ctx, "Archive Me Co", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Fatalf("archived company is still offered to bot: %#v", found)
	}
}

func request(t *testing.T, h http.Handler, method, path, token string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, bytes.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func assertAPIStatus(t *testing.T, h http.Handler, method, path, token string, body []byte, want int) {
	t.Helper()
	w := request(t, h, method, path, token, body)
	if w.Code != want {
		t.Fatalf("%s %s = %d, want %d: %s", method, path, w.Code, want, w.Body.String())
	}
}

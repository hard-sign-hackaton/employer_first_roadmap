package employerapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"efr_bot/models"
	"encoding/base64"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"strings"
	"time"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("conflict")
	ErrForbidden = errors.New("forbidden")
)

const (
	roleAdmin    = "admin"
	roleEmployer = "employer"
)

type Store struct{ db *gorm.DB }

func NewStore(db *gorm.DB) *Store { return &Store{db: db} }
func tokenHash(t string) string   { x := sha256.Sum256([]byte(t)); return fmt.Sprintf("%x", x) }
func newToken() (string, error) {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func accountView(a models.APIAccount, t string) AccountResponse {
	return AccountResponse{ID: a.ID, Login: a.Login, Role: a.Role, CompanyID: a.CompanyID, IsActive: a.IsActive, Token: t}
}
func (s *Store) BootstrapAdmin(c context.Context, t string) error {
	t = strings.TrimSpace(t)
	if t == "" {
		return nil
	}
	var a models.APIAccount
	e := s.db.WithContext(c).Where("role=? AND login=?", roleAdmin, "bootstrap-admin").First(&a).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return s.db.WithContext(c).Create(&models.APIAccount{Login: "bootstrap-admin", TokenHash: tokenHash(t), Role: roleAdmin, IsActive: true}).Error
	}
	if e != nil {
		return e
	}
	return s.db.WithContext(c).Model(&a).Updates(map[string]any{"token_hash": tokenHash(t), "is_active": true}).Error
}
func (s *Store) Authenticate(c context.Context, t string) (models.APIAccount, error) {
	var a models.APIAccount
	e := s.db.WithContext(c).Where("token_hash=? AND is_active=?", tokenHash(t), true).First(&a).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return a, fmt.Errorf("%w: API token", ErrNotFound)
	}
	return a, e
}
func (s *Store) CreateAccount(c context.Context, in CreateAccountInput) (AccountResponse, error) {
	in.Login = strings.TrimSpace(in.Login)
	if in.Login == "" || (in.Role != roleAdmin && in.Role != roleEmployer) || (in.Role == roleEmployer && in.CompanyID == nil) || (in.Role == roleAdmin && in.CompanyID != nil) {
		return AccountResponse{}, fmt.Errorf("invalid account fields")
	}
	if in.CompanyID != nil {
		var n int64
		if e := s.db.WithContext(c).Model(&models.Company{}).Where("id=?", *in.CompanyID).Count(&n).Error; e != nil {
			return AccountResponse{}, e
		}
		if n == 0 {
			return AccountResponse{}, fmt.Errorf("%w: company", ErrNotFound)
		}
	}
	t, e := newToken()
	if e != nil {
		return AccountResponse{}, e
	}
	a := models.APIAccount{Login: in.Login, Role: in.Role, CompanyID: in.CompanyID, TokenHash: tokenHash(t), IsActive: true}
	if e = s.db.WithContext(c).Create(&a).Error; e != nil {
		return AccountResponse{}, fmt.Errorf("%w: %v", ErrConflict, e)
	}
	return accountView(a, t), nil
}
func (s *Store) ListAccounts(c context.Context) ([]AccountResponse, error) {
	var as []models.APIAccount
	if e := s.db.WithContext(c).Order("id").Find(&as).Error; e != nil {
		return nil, e
	}
	o := make([]AccountResponse, 0, len(as))
	for _, a := range as {
		o = append(o, accountView(a, ""))
	}
	return o, nil
}
func (s *Store) UpdateAccount(c context.Context, id int64, in UpdateAccountInput) (AccountResponse, error) {
	var a models.APIAccount
	if e := s.db.WithContext(c).First(&a, id).Error; errors.Is(e, gorm.ErrRecordNotFound) {
		return AccountResponse{}, fmt.Errorf("%w: account", ErrNotFound)
	} else if e != nil {
		return AccountResponse{}, e
	}
	if in.Login != nil {
		a.Login = strings.TrimSpace(*in.Login)
	}
	if in.IsActive != nil {
		a.IsActive = *in.IsActive
	}
	if in.CompanyID != nil {
		if a.Role != roleEmployer {
			return AccountResponse{}, fmt.Errorf("admin account cannot have company")
		}
		a.CompanyID = in.CompanyID
	}
	if e := s.db.WithContext(c).Save(&a).Error; e != nil {
		return AccountResponse{}, e
	}
	return accountView(a, ""), nil
}
func (s *Store) RotateAccountToken(c context.Context, id int64) (AccountResponse, error) {
	var a models.APIAccount
	if e := s.db.WithContext(c).First(&a, id).Error; errors.Is(e, gorm.ErrRecordNotFound) {
		return AccountResponse{}, fmt.Errorf("%w: account", ErrNotFound)
	} else if e != nil {
		return AccountResponse{}, e
	}
	t, e := newToken()
	if e != nil {
		return AccountResponse{}, e
	}
	a.TokenHash = tokenHash(t)
	if e = s.db.WithContext(c).Save(&a).Error; e != nil {
		return AccountResponse{}, e
	}
	return accountView(a, t), nil
}

func companyView(x models.Company) CompanyResponse {
	return CompanyResponse{ID: x.ID, Name: x.Name, Description: x.Description, WebsiteURL: x.WebsiteURL, IsActive: x.IsActive}
}
func (s *Store) SaveCompany(c context.Context, id int64, in CompanyInput) (CompanyResponse, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return CompanyResponse{}, fmt.Errorf("name is required")
	}
	var x models.Company
	var e error
	if id == 0 {
		x = models.Company{Name: in.Name, Description: strings.TrimSpace(in.Description), WebsiteURL: strings.TrimSpace(in.WebsiteURL)}
		e = s.db.WithContext(c).Create(&x).Error
	} else {
		e = s.db.WithContext(c).First(&x, id).Error
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return CompanyResponse{}, fmt.Errorf("%w: company", ErrNotFound)
		}
		if e == nil {
			x.Name = in.Name
			x.Description = strings.TrimSpace(in.Description)
			x.WebsiteURL = strings.TrimSpace(in.WebsiteURL)
			e = s.db.WithContext(c).Save(&x).Error
		}
	}
	if e != nil {
		return CompanyResponse{}, fmt.Errorf("%w: %v", ErrConflict, e)
	}
	return companyView(x), nil
}
func (s *Store) GetCompany(c context.Context, id int64) (CompanyResponse, error) {
	var x models.Company
	if e := s.db.WithContext(c).First(&x, id).Error; errors.Is(e, gorm.ErrRecordNotFound) {
		return CompanyResponse{}, fmt.Errorf("%w: company", ErrNotFound)
	} else if e != nil {
		return CompanyResponse{}, e
	}
	return companyView(x), nil
}
func (s *Store) ListCompanies(c context.Context) ([]CompanyResponse, error) {
	var xs []models.Company
	if e := s.db.WithContext(c).Order("name").Find(&xs).Error; e != nil {
		return nil, e
	}
	o := make([]CompanyResponse, 0, len(xs))
	for _, x := range xs {
		o = append(o, companyView(x))
	}
	return o, nil
}
func (s *Store) ArchiveEntity(c context.Context, entity string, id int64, active bool) error {
	var model any
	switch entity {
	case "company":
		model = &models.Company{}
	case "direction":
		model = &models.CareerDirection{}
	case "opportunity":
		model = &models.CompanyOpportunity{}
	case "university":
		model = &models.University{}
	case "program":
		model = &models.EducationProgram{}
	default:
		return fmt.Errorf("unknown archive entity")
	}
	result := s.db.WithContext(c).Model(model).Where("id = ?", id).Update("is_active", active)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("%w: %s", ErrNotFound, entity)
	}
	return nil
}
func (s *Store) ListUniversities(c context.Context) ([]models.University, error) {
	var x []models.University
	return x, s.db.WithContext(c).Order("name").Find(&x).Error
}
func (s *Store) ListExamCombinations(c context.Context) ([]models.ExamCombination, error) {
	var x []models.ExamCombination
	return x, s.db.WithContext(c).Preload("Items").Order("admission_year DESC, id").Find(&x).Error
}
func (s *Store) ListAdmissionScores(c context.Context) ([]models.AdmissionScoreHistory, error) {
	var x []models.AdmissionScoreHistory
	return x, s.db.WithContext(c).Order("admission_year DESC").Find(&x).Error
}
func (s *Store) ListAdmissionRules(c context.Context) ([]models.AdmissionCampaignRule, error) {
	var x []models.AdmissionCampaignRule
	return x, s.db.WithContext(c).Order("admission_year DESC").Find(&x).Error
}
func (s *Store) ListRoadmapTemplates(c context.Context) ([]models.RoadmapTemplate, error) {
	var x []models.RoadmapTemplate
	return x, s.db.WithContext(c).Preload("Steps").Order("career_direction_id, version DESC").Find(&x).Error
}

func (s *Store) direction(c context.Context, id, companyID int64) (models.CareerDirection, error) {
	var x models.CareerDirection
	q := s.db.WithContext(c).Where("id=?", id)
	if companyID > 0 {
		q = q.Where("company_id=?", companyID)
	}
	e := q.First(&x).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return x, fmt.Errorf("%w: career direction", ErrNotFound)
	}
	return x, e
}
func (s *Store) SaveDirection(c context.Context, companyID, id int64, in CareerDirectionInput) (CareerDirectionResponse, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return CareerDirectionResponse{}, fmt.Errorf("name is required")
	}
	var x models.CareerDirection
	var e error
	if id == 0 {
		x = models.CareerDirection{CompanyID: companyID, Name: in.Name, Description: strings.TrimSpace(in.Description)}
		e = s.db.WithContext(c).Create(&x).Error
	} else {
		x, e = s.direction(c, id, companyID)
		if e == nil {
			x.Name = in.Name
			x.Description = strings.TrimSpace(in.Description)
			e = s.db.WithContext(c).Save(&x).Error
		}
	}
	if e != nil {
		return CareerDirectionResponse{}, fmt.Errorf("%w: %v", ErrConflict, e)
	}
	return s.Direction(c, x.ID, companyID)
}
func (s *Store) Direction(c context.Context, id, companyID int64) (CareerDirectionResponse, error) {
	x, e := s.direction(c, id, companyID)
	if e != nil {
		return CareerDirectionResponse{}, e
	}
	o := CareerDirectionResponse{ID: x.ID, CompanyID: x.CompanyID, Name: x.Name, Description: x.Description, IsActive: x.IsActive, InterestTags: []TagWeight{}, EducationProgramIDs: []int64{}}
	var ts []models.CareerDirectionInterestTag
	if e = s.db.WithContext(c).Where("career_direction_id=?", id).Find(&ts).Error; e != nil {
		return o, e
	}
	for _, v := range ts {
		o.InterestTags = append(o.InterestTags, TagWeight{InterestTagID: v.InterestTagID, Weight: v.Weight})
	}
	var ps []models.CareerDirectionEducationProgram
	if e = s.db.WithContext(c).Where("career_direction_id=?", id).Find(&ps).Error; e != nil {
		return o, e
	}
	for _, v := range ps {
		o.EducationProgramIDs = append(o.EducationProgramIDs, v.EducationProgramID)
	}
	return o, nil
}
func (s *Store) ListDirections(c context.Context, companyID int64) ([]CareerDirectionResponse, error) {
	var xs []models.CareerDirection
	if e := s.db.WithContext(c).Where("company_id=?", companyID).Order("name").Find(&xs).Error; e != nil {
		return nil, e
	}
	o := make([]CareerDirectionResponse, 0, len(xs))
	for _, x := range xs {
		v, e := s.Direction(c, x.ID, companyID)
		if e != nil {
			return nil, e
		}
		o = append(o, v)
	}
	return o, nil
}
func idsUnique(xs []int64) []int64 {
	m := map[int64]bool{}
	o := []int64{}
	for _, x := range xs {
		if x > 0 && !m[x] {
			m[x] = true
			o = append(o, x)
		}
	}
	return o
}
func (s *Store) ReplaceDirectionPrograms(c context.Context, id, companyID int64, ids []int64) error {
	if _, e := s.direction(c, id, companyID); e != nil {
		return e
	}
	ids = idsUnique(ids)
	if len(ids) == 0 {
		return fmt.Errorf("at least one education program is required")
	}
	var n int64
	if e := s.db.WithContext(c).Model(&models.EducationProgram{}).Where("id IN ?", ids).Count(&n).Error; e != nil {
		return e
	}
	if n != int64(len(ids)) {
		return fmt.Errorf("%w: education program", ErrNotFound)
	}
	return s.db.WithContext(c).Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("career_direction_id=?", id).Delete(&models.CareerDirectionEducationProgram{}).Error; e != nil {
			return e
		}
		for _, p := range ids {
			if e := tx.Create(&models.CareerDirectionEducationProgram{CareerDirectionID: id, EducationProgramID: p}).Error; e != nil {
				return e
			}
		}
		return nil
	})
}
func (s *Store) ReplaceDirectionTags(c context.Context, id, companyID int64, ts []TagWeight) error {
	if _, e := s.direction(c, id, companyID); e != nil {
		return e
	}
	return s.db.WithContext(c).Transaction(func(tx *gorm.DB) error {
		if e := tx.Where("career_direction_id=?", id).Delete(&models.CareerDirectionInterestTag{}).Error; e != nil {
			return e
		}
		for _, v := range ts {
			if v.InterestTagID <= 0 || v.Weight < 0 {
				return fmt.Errorf("invalid interest tag")
			}
			var n int64
			if e := tx.Model(&models.InterestTag{}).Where("id=?", v.InterestTagID).Count(&n).Error; e != nil {
				return e
			}
			if n == 0 {
				return fmt.Errorf("%w: interest tag", ErrNotFound)
			}
			if e := tx.Create(&models.CareerDirectionInterestTag{CareerDirectionID: id, InterestTagID: v.InterestTagID, Weight: v.Weight}).Error; e != nil {
				return e
			}
		}
		return nil
	})
}

func (s *Store) opportunity(c context.Context, id, companyID int64) (models.CompanyOpportunity, error) {
	var x models.CompanyOpportunity
	q := s.db.WithContext(c).Where("id=?", id)
	if companyID > 0 {
		q = q.Where("company_id=?", companyID)
	}
	e := q.First(&x).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return x, fmt.Errorf("%w: opportunity", ErrNotFound)
	}
	return x, e
}
func opportunityView(x models.CompanyOpportunity) OpportunityResponse {
	return OpportunityResponse{ID: x.ID, OpportunityInput: OpportunityInput{CareerDirectionID: x.CareerDirectionID, Type: x.Type, Name: x.Name, Description: x.Description, URL: x.URL, ActiveListingURL: x.ActiveListingURL, SourceCheckedAt: x.SourceCheckedAt, WorkFormat: x.WorkFormat, MinStudyYear: x.MinStudyYear, RegionID: x.RegionID, IsActive: x.IsActive}, UpdatedAt: x.UpdatedAt}
}
func (s *Store) SaveOpportunity(c context.Context, companyID, id int64, in OpportunityInput) (OpportunityResponse, error) {
	if _, e := s.direction(c, in.CareerDirectionID, companyID); e != nil {
		return OpportunityResponse{}, e
	}
	if strings.TrimSpace(in.Name) == "" || in.MinStudyYear < 1 || in.MinStudyYear > 6 || (in.WorkFormat != "" && in.WorkFormat != "onsite" && in.WorkFormat != "hybrid" && in.WorkFormat != "remote") {
		return OpportunityResponse{}, fmt.Errorf("invalid opportunity")
	}
	var x models.CompanyOpportunity
	var e error
	if id == 0 {
		x = models.CompanyOpportunity{CompanyID: companyID}
	} else {
		x, e = s.opportunity(c, id, companyID)
		if e != nil {
			return OpportunityResponse{}, e
		}
	}
	x.CareerDirectionID = in.CareerDirectionID
	x.Type = in.Type
	x.Name = strings.TrimSpace(in.Name)
	x.Description = strings.TrimSpace(in.Description)
	x.URL = strings.TrimSpace(in.URL)
	x.ActiveListingURL = strings.TrimSpace(in.ActiveListingURL)
	x.SourceCheckedAt = in.SourceCheckedAt
	x.WorkFormat = in.WorkFormat
	if x.WorkFormat == "" {
		x.WorkFormat = "onsite"
	}
	x.MinStudyYear = in.MinStudyYear
	x.RegionID = in.RegionID
	x.IsActive = in.IsActive
	if x.ID == 0 {
		e = s.db.WithContext(c).Create(&x).Error
	} else {
		e = s.db.WithContext(c).Save(&x).Error
	}
	if e != nil {
		return OpportunityResponse{}, e
	}
	return opportunityView(x), nil
}
func (s *Store) ListOpportunities(c context.Context, companyID int64) ([]OpportunityResponse, error) {
	var xs []models.CompanyOpportunity
	if e := s.db.WithContext(c).Where("company_id=?", companyID).Order("id").Find(&xs).Error; e != nil {
		return nil, e
	}
	o := make([]OpportunityResponse, 0, len(xs))
	for _, x := range xs {
		o = append(o, opportunityView(x))
	}
	return o, nil
}

func (s *Store) SaveUniversity(c context.Context, id int64, in UniversityInput) (UniversityResponse, error) {
	if in.RegionID <= 0 || strings.TrimSpace(in.Name) == "" {
		return UniversityResponse{}, fmt.Errorf("invalid university")
	}
	var x models.University
	var e error
	if id == 0 {
		x = models.University{RegionID: in.RegionID, Name: strings.TrimSpace(in.Name), Description: strings.TrimSpace(in.Description), WebsiteURL: strings.TrimSpace(in.WebsiteURL)}
		e = s.db.WithContext(c).Create(&x).Error
	} else {
		e = s.db.WithContext(c).First(&x, id).Error
		if e == nil {
			x.RegionID = in.RegionID
			x.Name = strings.TrimSpace(in.Name)
			x.Description = strings.TrimSpace(in.Description)
			x.WebsiteURL = strings.TrimSpace(in.WebsiteURL)
			e = s.db.WithContext(c).Save(&x).Error
		}
	}
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return UniversityResponse{}, fmt.Errorf("%w: university", ErrNotFound)
	}
	if e != nil {
		return UniversityResponse{}, e
	}
	return UniversityResponse{ID: x.ID, UniversityInput: in}, nil
}
func (s *Store) SaveProgram(c context.Context, id int64, in EducationProgramInput) (EducationProgramResponse, error) {
	if in.UniversityID <= 0 || strings.TrimSpace(in.Name) == "" {
		return EducationProgramResponse{}, fmt.Errorf("invalid education program")
	}
	var x models.EducationProgram
	var e error
	if id == 0 {
		x = models.EducationProgram{UniversityID: in.UniversityID, Code: strings.TrimSpace(in.Code), Name: strings.TrimSpace(in.Name), Description: strings.TrimSpace(in.Description)}
		e = s.db.WithContext(c).Create(&x).Error
	} else {
		e = s.db.WithContext(c).First(&x, id).Error
		if e == nil {
			x.UniversityID = in.UniversityID
			x.Code = strings.TrimSpace(in.Code)
			x.Name = strings.TrimSpace(in.Name)
			x.Description = strings.TrimSpace(in.Description)
			e = s.db.WithContext(c).Save(&x).Error
		}
	}
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return EducationProgramResponse{}, fmt.Errorf("%w: education program", ErrNotFound)
	}
	if e != nil {
		return EducationProgramResponse{}, e
	}
	return EducationProgramResponse{ID: x.ID, EducationProgramInput: in}, nil
}
func (s *Store) ListPrograms(c context.Context) ([]EducationProgramResponse, error) {
	var rows []models.EducationProgram
	if err := s.db.WithContext(c).Order("name").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]EducationProgramResponse, 0, len(rows))
	for _, x := range rows {
		out = append(out, EducationProgramResponse{ID: x.ID, EducationProgramInput: EducationProgramInput{UniversityID: x.UniversityID, Code: x.Code, Name: x.Name, Description: x.Description}})
	}
	return out, nil
}
func (s *Store) SaveExamCombination(c context.Context, id int64, in ExamCombinationInput) (ExamCombinationResponse, error) {
	if in.EducationProgramID <= 0 || in.AdmissionYear < 2020 || len(in.Items) == 0 {
		return ExamCombinationResponse{}, fmt.Errorf("invalid exam combination")
	}
	items := make([]ExamCombinationItemInput, 0, len(in.Items))
	seen := map[int64]bool{}
	for _, item := range in.Items {
		if item.ExamSubjectID <= 0 || seen[item.ExamSubjectID] {
			return ExamCombinationResponse{}, fmt.Errorf("invalid exam combination items")
		}
		seen[item.ExamSubjectID] = true
		items = append(items, item)
	}
	var combination models.ExamCombination
	if id == 0 {
		combination = models.ExamCombination{EducationProgramID: in.EducationProgramID, AdmissionYear: in.AdmissionYear, Name: in.Name}
	} else {
		if err := s.db.WithContext(c).First(&combination, id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return ExamCombinationResponse{}, fmt.Errorf("%w: exam combination", ErrNotFound)
		} else if err != nil {
			return ExamCombinationResponse{}, err
		}
		combination.EducationProgramID = in.EducationProgramID
		combination.AdmissionYear = in.AdmissionYear
		combination.Name = in.Name
	}
	transaction := s.db.WithContext(c).Transaction(func(tx *gorm.DB) error {
		if combination.ID == 0 {
			if err := tx.Create(&combination).Error; err != nil {
				return err
			}
		} else if err := tx.Save(&combination).Error; err != nil {
			return err
		}
		if err := tx.Where("exam_combination_id = ?", combination.ID).Delete(&models.ExamCombinationItem{}).Error; err != nil {
			return err
		}
		for _, item := range items {
			var count int64
			if err := tx.Model(&models.ExamSubject{}).Where("id = ?", item.ExamSubjectID).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return fmt.Errorf("%w: exam subject", ErrNotFound)
			}
			if err := tx.Create(&models.ExamCombinationItem{ExamCombinationID: combination.ID, ExamSubjectID: item.ExamSubjectID, MinScore: item.MinScore}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if transaction != nil {
		return ExamCombinationResponse{}, transaction
	}
	in.Items = items
	return ExamCombinationResponse{ID: combination.ID, ExamCombinationInput: in}, nil
}
func (s *Store) SaveRoadmapTemplate(c context.Context, in RoadmapTemplateInput) error {
	if in.CareerDirectionID <= 0 || strings.TrimSpace(in.Name) == "" || len(in.Steps) == 0 {
		return fmt.Errorf("invalid roadmap template")
	}
	return s.db.WithContext(c).Transaction(func(tx *gorm.DB) error {
		var direction models.CareerDirection
		if err := tx.First(&direction, in.CareerDirectionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: career direction", ErrNotFound)
			}
			return err
		}
		var maxVersion int
		if err := tx.Model(&models.RoadmapTemplate{}).Where("career_direction_id = ?", in.CareerDirectionID).Select("COALESCE(MAX(version), 0)").Scan(&maxVersion).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.RoadmapTemplate{}).Where("career_direction_id = ? AND is_active = ?", in.CareerDirectionID, true).Update("is_active", false).Error; err != nil {
			return err
		}
		template := models.RoadmapTemplate{CareerDirectionID: in.CareerDirectionID, Name: strings.TrimSpace(in.Name), Version: maxVersion + 1, IsActive: true}
		if err := tx.Create(&template).Error; err != nil {
			return err
		}
		for index, step := range in.Steps {
			if strings.TrimSpace(step.StepType) == "" || strings.TrimSpace(step.Title) == "" {
				return fmt.Errorf("invalid roadmap step")
			}
			if err := tx.Create(&models.RoadmapTemplateStep{RoadmapTemplateID: template.ID, OrderNo: int16(index + 1), StepType: strings.TrimSpace(step.StepType), Title: strings.TrimSpace(step.Title), Description: strings.TrimSpace(step.Description)}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
func (s *Store) SaveAdmissionScore(c context.Context, in AdmissionScoreInput) error {
	if in.EducationProgramID <= 0 || in.AdmissionYear < 2020 {
		return fmt.Errorf("invalid admission score")
	}
	return s.db.WithContext(c).Save(&models.AdmissionScoreHistory{EducationProgramID: in.EducationProgramID, AdmissionYear: in.AdmissionYear, BudgetPassingScore: in.BudgetPassingScore, PaidPassingScore: in.PaidPassingScore}).Error
}
func (s *Store) SaveAdmissionRule(c context.Context, in AdmissionRuleInput) error {
	if in.AdmissionYear < 2020 || in.MaxUniversities <= 0 || in.MaxProgramsPerUniversity <= 0 {
		return fmt.Errorf("invalid admission rule")
	}
	return s.db.WithContext(c).Save(&models.AdmissionCampaignRule{AdmissionYear: in.AdmissionYear, MaxUniversities: in.MaxUniversities, MaxProgramsPerUniversity: in.MaxProgramsPerUniversity}).Error
}
func (s *Store) ReferenceData(c context.Context) (ReferenceDataResponse, error) {
	var rs []models.Region
	var ts []models.InterestTag
	var es []models.ExamSubject
	if e := s.db.WithContext(c).Order("name").Find(&rs).Error; e != nil {
		return ReferenceDataResponse{}, e
	}
	if e := s.db.WithContext(c).Order("name").Find(&ts).Error; e != nil {
		return ReferenceDataResponse{}, e
	}
	if e := s.db.WithContext(c).Order("name").Find(&es).Error; e != nil {
		return ReferenceDataResponse{}, e
	}
	o := ReferenceDataResponse{Regions: []NamedReference{}, InterestTags: []NamedReference{}, ExamSubjects: []NamedReference{}}
	for _, x := range rs {
		o.Regions = append(o.Regions, NamedReference{ID: x.ID, Name: x.Name})
	}
	for _, x := range ts {
		o.InterestTags = append(o.InterestTags, NamedReference{ID: x.ID, Name: x.Name})
	}
	for _, x := range es {
		o.ExamSubjects = append(o.ExamSubjects, NamedReference{ID: x.ID, Name: x.Name})
	}
	return o, nil
}

func (s *Store) ListEmployerApplications(c context.Context, companyID int64) ([]EmployerApplicationListItem, error) {
	type row struct {
		RoadmapID       int64
		UserID          int64
		Grade           int16
		UserRegion      string
		CareerDirection string
		OpportunityID   int64
		OpportunityName string
		SubmittedAt     time.Time
	}
	var rows []row
	e := s.db.WithContext(c).Table("roadmap_employer_applications AS a").Select(`a.roadmap_id, g.user_profile_id AS user_id, u.grade, r.name AS user_region, d.name AS career_direction, o.id AS opportunity_id, o.name AS opportunity_name, a.submitted_at`).Joins("JOIN roadmaps rm ON rm.id=a.roadmap_id").Joins("JOIN user_goals g ON g.id=rm.user_goal_id").Joins("JOIN user_profiles u ON u.id=g.user_profile_id").Joins("JOIN regions r ON r.id=u.region_id").Joins("JOIN career_directions d ON d.id=g.career_direction_id").Joins("JOIN company_opportunities o ON o.id=a.company_opportunity_id").Where("d.company_id=?", companyID).Scan(&rows).Error
	if e != nil {
		return nil, e
	}
	out := make([]EmployerApplicationListItem, 0, len(rows))
	for _, x := range rows {
		out = append(out, EmployerApplicationListItem{RoadmapID: x.RoadmapID, UserID: x.UserID, Grade: x.Grade, UserRegion: x.UserRegion, CareerDirection: x.CareerDirection, OpportunityID: x.OpportunityID, OpportunityName: x.OpportunityName, SubmittedAt: x.SubmittedAt, Status: "submitted", FeedbackHistory: []EmployerFeedbackView{}})
	}
	return out, nil
}
func (s *Store) SaveEmployerFeedback(c context.Context, companyID, roadmapID int64, in EmployerFeedbackInput) (EmployerFeedbackView, error) {
	var n int64
	e := s.db.WithContext(c).Table("roadmap_employer_applications AS a").Joins("JOIN roadmaps rm ON rm.id=a.roadmap_id").Joins("JOIN user_goals g ON g.id=rm.user_goal_id").Joins("JOIN career_directions d ON d.id=g.career_direction_id").Where("a.roadmap_id=? AND d.company_id=?", roadmapID, companyID).Count(&n).Error
	if e != nil {
		return EmployerFeedbackView{}, e
	}
	if n == 0 {
		return EmployerFeedbackView{}, fmt.Errorf("%w: application", ErrNotFound)
	}
	f := models.EmployerFeedback{RoadmapID: roadmapID, Status: in.Status, Message: strings.TrimSpace(in.Message), Contact: strings.TrimSpace(in.Contact), CreatedAt: time.Now().UTC()}
	if e = s.db.WithContext(c).Create(&f).Error; e != nil {
		return EmployerFeedbackView{}, e
	}
	return EmployerFeedbackView{ID: f.ID, Status: f.Status, Message: f.Message, Contact: f.Contact, CreatedAt: f.CreatedAt}, nil
}

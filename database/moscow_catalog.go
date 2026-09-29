package database

import (
	"fmt"
	"os"
	"strings"
	"time"

	"efr_bot/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MoscowCatalogSeedEnabled is intentionally separate from DemoSeedEnabled.
// A real catalog must never share the demo database/volume by accident.
func MoscowCatalogSeedEnabled() bool {
	return strings.EqualFold(os.Getenv("SEED_MOSCOW_CATALOG"), "true")
}

// SeedMoscowCatalog imports the reviewed Moscow-only catalog.  Every URL below
// is an official institution/company page; program-to-direction links and
// interest tags are editorial product mappings, not claims made by a university.
// Passing scores are intentionally absent until a university publishes a source
// that can be stored and checked for a concrete programme and campaign.
func SeedMoscowCatalog(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		regions, err := seedRegions(tx)
		if err != nil {
			return err
		}
		tags, err := seedTags(tx)
		if err != nil {
			return err
		}
		subjects, err := seedSubjects(tx)
		if err != nil {
			return err
		}
		if err := normalizeAdmissionMathSubject(tx, subjects); err != nil {
			return err
		}
		moscow, ok := regions["Москва"]
		if !ok {
			return fmt.Errorf("Moscow reference region is missing")
		}
		companies, directions, err := seedMoscowCompanies(tx, tags)
		if err != nil {
			return err
		}
		programs, err := seedMoscowEducation(tx, moscow.ID, subjects)
		if err != nil {
			return err
		}
		if err := seedMoscowDirectionPrograms(tx, directions, programs); err != nil {
			return err
		}
		if err := seedMoscowOpportunities(tx, moscow.ID, companies, directions); err != nil {
			return err
		}
		return seedMoscowRoadmapTemplates(tx, directions)
	})
}

type moscowDirection struct {
	key, company, name, description string
	tags                            map[string]float64
}

func seedMoscowCompanies(tx *gorm.DB, tags map[string]models.InterestTag) (map[string]models.Company, map[string]models.CareerDirection, error) {
	companyData := []struct{ key, name, description, url string }{
		{"yandex", "Яндекс", "Технологическая компания и разработчик цифровых сервисов.", "https://yandex.ru/company/"},
		{"vk", "VK", "Российская технологическая компания, развивающая интернет-сервисы и социальные платформы.", "https://vk.company/ru/"},
		{"sber", "Сбер", "Технологическая компания и экосистема финансовых и цифровых сервисов.", "https://www.sberbank.ru/"},
		{"mts", "МТС", "Цифровая экосистема в телекоме и ИТ.", "https://moskva.mts.ru/"},
		{"avito", "Авито", "Онлайн-платформа объявлений и технологическая компания.", "https://www.avito.ru/"},
		{"rosatom", "Госкорпорация «Росатом»", "Российская государственная корпорация в области атомной энергетики и технологий.", "https://www.rosatom.ru/"},
		{"rzd", "ОАО «РЖД»", "Российская железнодорожная компания и транспортный холдинг.", "https://www.rzd.ru/"},
	}
	companies := make(map[string]models.Company, len(companyData))
	for _, item := range companyData {
		value := models.Company{Name: item.name, Description: item.description, WebsiteURL: item.url, IsActive: true}
		if err := tx.Where("name = ?", value.Name).Assign(value).FirstOrCreate(&value).Error; err != nil {
			return nil, nil, err
		}
		companies[item.key] = value
	}
	directionData := []moscowDirection{
		{"yandex_dev", "yandex", "Разработчик программного обеспечения", "Разрабатывает и поддерживает цифровые сервисы.", map[string]float64{"Программирование": 1, "Математика": .8}},
		{"yandex_data", "yandex", "Аналитик данных", "Работает с данными и продуктовыми метриками.", map[string]float64{"Аналитика": 1, "Математика": .9}},
		{"vk_dev", "vk", "Разработчик программного обеспечения", "Разрабатывает веб- и платформенные сервисы.", map[string]float64{"Программирование": 1, "Математика": .7}},
		{"vk_product", "vk", "Менеджер продукта", "Развивает цифровой продукт на основе потребностей пользователей и данных.", map[string]float64{"Управление продуктом": 1, "Аналитика": .8, "Коммуникация": .7}},
		{"sber_data", "sber", "Аналитик данных", "Создаёт аналитические решения и модели для цифровых продуктов.", map[string]float64{"Аналитика": 1, "Математика": .9}},
		{"sber_dev", "sber", "Разработчик программного обеспечения", "Разрабатывает банковские и технологические сервисы.", map[string]float64{"Программирование": 1, "Математика": .8}},
		{"mts_dev", "mts", "Разработчик цифровых сервисов", "Разрабатывает цифровые продукты и сервисы экосистемы.", map[string]float64{"Программирование": 1, "Математика": .7}},
		{"mts_telecom", "mts", "Инженер телекоммуникационных систем", "Работает с сетями связи и телекоммуникационной инфраструктурой.", map[string]float64{"Инженерия": 1, "Физика": .7}},
		{"avito_data", "avito", "Аналитик данных", "Помогает принимать продуктовые решения на основе данных.", map[string]float64{"Аналитика": 1, "Математика": .9}},
		{"avito_dev", "avito", "Разработчик программного обеспечения", "Разрабатывает сервисы онлайн-платформы.", map[string]float64{"Программирование": 1, "Математика": .7}},
		{"rosatom_nuclear", "rosatom", "Инженер атомной энергетики", "Работает с инженерными задачами атомной энергетики.", map[string]float64{"Инженерия": 1, "Физика": 1, "Исследования": .6}},
		{"rosatom_it", "rosatom", "ИТ-специалист", "Создаёт и сопровождает цифровые решения для отрасли.", map[string]float64{"Программирование": 1, "Инженерия": .6}},
		{"rzd_transport", "rzd", "Инженер транспортных систем", "Работает с технологиями и инфраструктурой железнодорожного транспорта.", map[string]float64{"Инженерия": 1, "Логистика": .9}},
		{"rzd_it", "rzd", "Разработчик цифровых транспортных систем", "Создаёт цифровые решения для железнодорожного транспорта.", map[string]float64{"Программирование": 1, "Логистика": .8}},
	}
	directions := make(map[string]models.CareerDirection, len(directionData))
	for _, item := range directionData {
		company := companies[item.company]
		value := models.CareerDirection{CompanyID: company.ID, Name: item.name, Description: item.description, IsActive: true}
		if err := tx.Where("company_id = ? AND name = ?", company.ID, item.name).Assign(value).FirstOrCreate(&value).Error; err != nil {
			return nil, nil, err
		}
		for tag, weight := range item.tags {
			link := models.CareerDirectionInterestTag{CareerDirectionID: value.ID, InterestTagID: tags[tag].ID, Weight: weight}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "career_direction_id"}, {Name: "interest_tag_id"}}, DoUpdates: clause.AssignmentColumns([]string{"weight"})}).Create(&link).Error; err != nil {
				return nil, nil, err
			}
		}
		directions[item.key] = value
	}
	return companies, directions, nil
}

type moscowProgram struct {
	key, university, code, name, description, source string
	subjects                                         []string
}

func seedMoscowEducation(tx *gorm.DB, moscowID int64, subjects map[string]models.ExamSubject) (map[string]models.EducationProgram, error) {
	// Source pages are official admission portals.  Combinations are stored only
	// as subject sets; per-subject minima are nil until a programme-specific rule
	// is confirmed, so the bot never presents invented thresholds.
	unis := []struct{ name, description, url string }{
		{"Национальный исследовательский университет «Высшая школа экономики» (НИУ ВШЭ)", "Национальный исследовательский университет в Москве.", "https://www.hse.ru/"},
		{"Московский государственный технический университет им. Н. Э. Баумана (МГТУ им. Баумана)", "Технический университет в Москве.", "https://bmstu.ru/"},
		{"Национальный исследовательский технологический университет «МИСИС» (НИТУ МИСИС)", "Технологический университет в Москве.", "https://misis.ru/"},
		{"Национальный исследовательский ядерный университет «МИФИ» (НИЯУ МИФИ)", "Ядерный исследовательский университет в Москве.", "https://mephi.ru/"},
		{"Российский университет транспорта (РУТ (МИИТ))", "Транспортный университет в Москве.", "https://rut-miit.ru/"},
		{"Российский университет дружбы народов имени Патриса Лумумбы (РУДН)", "Университет в Москве.", "https://www.rudn.ru/"},
		{"Российский технологический университет МИРЭА (РТУ МИРЭА)", "Технологический университет в Москве.", "https://www.mirea.ru/"},
		{"Первый Московский государственный медицинский университет имени И. М. Сеченова (Сеченовский Университет)", "Медицинский университет в Москве.", "https://www.sechenov.ru/"},
	}
	universities := map[string]models.University{}
	for _, u := range unis {
		value := models.University{RegionID: moscowID, Name: u.name, Description: u.description, WebsiteURL: u.url, IsActive: true}
		if err := tx.Where("region_id = ? AND name = ?", moscowID, u.name).Assign(value).FirstOrCreate(&value).Error; err != nil {
			return nil, err
		}
		universities[u.name] = value
	}
	data := []moscowProgram{
		{"hse_pi", unis[0].name, "09.03.03", "Прикладная информатика", "ОП в области прикладной информатики.", "https://ba.hse.ru/", []string{"Русский язык", "Математика (профильная)", "Информатика"}},
		{"hse_bi", unis[0].name, "38.03.05", "Бизнес-информатика", "ОП на стыке ИТ и управления.", "https://ba.hse.ru/", []string{"Русский язык", "Математика (профильная)", "Обществознание"}},
		{"bmstu_it", unis[1].name, "09.03.01", "Информатика и вычислительная техника", "ИТ-инженерная ОП.", "https://bmstu.ru/abitur", []string{"Русский язык", "Математика (профильная)", "Информатика"}},
		{"bmstu_auto", unis[1].name, "15.03.04", "Автоматизация технологических процессов и производств", "Инженерная ОП автоматизации.", "https://bmstu.ru/abitur", []string{"Русский язык", "Математика (профильная)", "Физика"}},
		{"misis_it", unis[2].name, "09.03.01", "Информатика и вычислительная техника", "ИТ-ОП НИТУ МИСИС.", "https://misis.ru/applicants/", []string{"Русский язык", "Математика (профильная)", "Информатика"}},
		{"misis_materials", unis[2].name, "22.03.01", "Материаловедение и технологии материалов", "Инженерно-материаловедческая ОП.", "https://misis.ru/applicants/", []string{"Русский язык", "Математика (профильная)", "Физика"}},
		{"mephi_nuclear", unis[3].name, "14.03.02", "Ядерные физика и технологии", "ОП в сфере ядерных технологий.", "https://admission.mephi.ru/", []string{"Русский язык", "Математика (профильная)", "Физика"}},
		{"mephi_it", unis[3].name, "09.03.01", "Информатика и вычислительная техника", "ИТ-ОП НИЯУ МИФИ.", "https://admission.mephi.ru/", []string{"Русский язык", "Математика (профильная)", "Информатика"}},
		{"rut_transport", unis[4].name, "23.03.01", "Технология транспортных процессов", "Транспортно-логистическая ОП.", "https://rut-miit.ru/", []string{"Русский язык", "Математика (профильная)", "Физика"}},
		{"rut_it", unis[4].name, "09.03.01", "Информатика и вычислительная техника", "ИТ-ОП РУТ (МИИТ).", "https://rut-miit.ru/", []string{"Русский язык", "Математика (профильная)", "Информатика"}},
		{"rudn_se", unis[5].name, "09.03.04", "Программная инженерия", "ОП программной инженерии.", "https://www.rudn.ru/abiturient", []string{"Русский язык", "Математика (профильная)", "Информатика"}},
		{"rudn_management", unis[5].name, "38.03.02", "Менеджмент", "ОП менеджмента.", "https://www.rudn.ru/abiturient", []string{"Русский язык", "Математика (профильная)", "Обществознание"}},
		{"mirea_se", unis[6].name, "09.03.04", "Программная инженерия", "ОП программной инженерии.", "https://priem.mirea.ru/", []string{"Русский язык", "Математика (профильная)", "Информатика"}},
		{"mirea_telecom", unis[6].name, "11.03.02", "Инфокоммуникационные технологии и системы связи", "ОП в области сетей и связи.", "https://priem.mirea.ru/", []string{"Русский язык", "Математика (профильная)", "Физика"}},
		{"sechenov_biophysics", unis[7].name, "30.05.02", "Медицинская биофизика", "Медицинско-физическая специализация.", "https://www.sechenov.ru/univers/", []string{"Русский язык", "Химия", "Биология"}},
		{"sechenov_biology", unis[7].name, "06.03.01", "Биология", "Биологическая ОП.", "https://www.sechenov.ru/univers/", []string{"Русский язык", "Химия", "Биология"}},
		// These additions have programme-level official sources, rather than an
		// inferred relation from a university-wide catalogue page.
		{"hse_data", unis[0].name, "01.03.02", "Компьютерные науки и анализ данных", "ОП по прикладной математике и анализу данных.", "https://ba.hse.ru/minkrit", []string{"Русский язык", "Математика (профильная)", "Информатика"}},
		{"mephi_energy", unis[3].name, "14.03.01", "Ядерная энергетика и теплофизика", "ОП в области ядерной энергетики.", "https://admission.mephi.ru/admission/baccalaureate-and-specialty/exams/list", []string{"Русский язык", "Математика (профильная)", "Физика"}},
		{"rut_is", unis[4].name, "09.03.02", "Информационные системы и технологии", "Информационные системы и технологии на транспорте.", "https://www.rut-miit.ru/admissions/degrees", []string{"Русский язык", "Математика (профильная)", "Информатика"}},
		{"rudn_ai", unis[5].name, "02.03.02", "Фундаментальная информатика и информационные технологии", "Профиль: искусственный интеллект: разработка и обучение интеллектуальных систем.", "https://admission.rudn.ru/pk/2026/adm_rules/bs/pp_bsm_26.pdf", []string{"Русский язык", "Математика (профильная)", "Информатика"}},
		{"misis_pi", unis[2].name, "09.03.03", "Прикладная информатика", "ИТ-программа НИТУ МИСИС.", "https://jen.msk.misis.ru/applicants/admission/baccalaureate-and-specialty/list/perechen_vstupitel_nyhispytanii/", []string{"Русский язык", "Математика (профильная)", "Информатика"}},
		{"mirea_ai", unis[6].name, "09.03.03", "Прикладная информатика", "ОП в области прикладной информатики.", "https://priem.mirea.ru/", []string{"Русский язык", "Математика (профильная)", "Информатика"}},
		{"sechenov_medicine", unis[7].name, "31.05.01", "Лечебное дело", "Специалитет Сеченовского Университета.", "https://www.sechenov.ru/univers/", []string{"Русский язык", "Химия", "Биология"}},
	}
	checked := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	confirmedMinimums := map[string]map[string]int16{
		"hse_data":     {"Русский язык": 60, "Математика (профильная)": 70, "Информатика": 65},
		"mephi_energy": {"Русский язык": 70, "Математика (профильная)": 75, "Физика": 75},
		"rudn_ai":      {"Русский язык": 65, "Математика (профильная)": 65, "Информатика": 65},
		"misis_pi":     {"Математика (профильная)": 65},
	}
	confirmedPassingScores := map[string]struct {
		budget int16
		source string
	}{
		"bmstu_it":      {278, "https://home.science.iu5.bmstu.ru/ab"},
		"mephi_nuclear": {272, "https://admission.mephi.ru/admission/baccalaureate-and-specialty/exams/previous-years"},
		"mephi_energy":  {271, "https://admission.mephi.ru/admission/baccalaureate-and-specialty/exams/previous-years"},
	}
	result := map[string]models.EducationProgram{}
	for _, item := range data {
		uni := universities[item.university]
		p := models.EducationProgram{UniversityID: uni.ID, Code: item.code, Name: item.name, Description: item.description + " Официальный источник: " + item.source, SourceURL: item.source, SourceCheckedAt: &checked, IsActive: true}
		if err := tx.Where("university_id = ? AND code = ? AND name = ?", uni.ID, p.Code, p.Name).Assign(p).FirstOrCreate(&p).Error; err != nil {
			return nil, err
		}
		combo := models.ExamCombination{EducationProgramID: p.ID, AdmissionYear: 2026, SourceURL: item.source, SourceCheckedAt: &checked}
		if err := tx.Where("education_program_id = ? AND admission_year = ?", p.ID, 2026).Assign(combo).FirstOrCreate(&combo).Error; err != nil {
			return nil, err
		}
		for _, subject := range item.subjects {
			link := models.ExamCombinationItem{ExamCombinationID: combo.ID, ExamSubjectID: subjectsMap(subjects, subject)}
			if score, found := confirmedMinimums[item.key][subject]; found {
				link.MinScore = &score
			}
			if link.ExamSubjectID == 0 {
				return nil, fmt.Errorf("unknown subject %q", subject)
			}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "exam_combination_id"}, {Name: "exam_subject_id"}}, DoUpdates: clause.AssignmentColumns([]string{"min_score"})}).Create(&link).Error; err != nil {
				return nil, err
			}
		}
		if score, found := confirmedPassingScores[item.key]; found {
			value := models.AdmissionScoreHistory{EducationProgramID: p.ID, AdmissionYear: 2025, BudgetPassingScore: &score.budget, SourceURL: score.source, SourceCheckedAt: &checked}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "education_program_id"}, {Name: "admission_year"}}, DoUpdates: clause.AssignmentColumns([]string{"budget_passing_score", "paid_passing_score", "source_url", "source_checked_at"})}).Create(&value).Error; err != nil {
				return nil, err
			}
		}
		result[item.key] = p
	}
	return result, nil
}

func subjectsMap(subjects map[string]models.ExamSubject, name string) int64 { return subjects[name].ID }

func seedMoscowDirectionPrograms(tx *gorm.DB, directions map[string]models.CareerDirection, programs map[string]models.EducationProgram) error {
	links := map[string][]string{
		"yandex_dev": {"hse_pi", "bmstu_it", "mirea_ai"}, "yandex_data": {"hse_data", "hse_bi", "mephi_it"}, "vk_dev": {"hse_pi", "mirea_se", "rudn_se"}, "vk_product": {"hse_bi", "rudn_management"}, "sber_data": {"hse_data", "mephi_it", "rudn_ai"}, "sber_dev": {"bmstu_it", "misis_pi", "mirea_se"}, "mts_dev": {"bmstu_it", "mirea_se", "rut_it"}, "mts_telecom": {"mirea_telecom", "bmstu_auto"}, "avito_data": {"hse_data", "hse_bi", "rudn_management"}, "avito_dev": {"hse_pi", "mirea_se", "rudn_se"}, "rosatom_nuclear": {"mephi_nuclear", "mephi_energy", "bmstu_auto"}, "rosatom_it": {"mephi_it", "misis_pi"}, "rzd_transport": {"rut_transport", "bmstu_auto"}, "rzd_it": {"rut_it", "rut_is", "mirea_se"},
	}
	for direction, items := range links {
		for _, program := range items {
			row := models.CareerDirectionEducationProgram{CareerDirectionID: directions[direction].ID, EducationProgramID: programs[program].ID}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedMoscowOpportunities(tx *gorm.DB, moscowID int64, companies map[string]models.Company, directions map[string]models.CareerDirection) error {
	checked := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	data := []struct {
		company, direction, name, portal, listing string
		course                                    int16
		kind                                      string
	}{
		{"yandex", "yandex_dev", "Young&&Yandex: стажировка в разработке", "https://yandex.ru/jobs/", "https://yandex.ru/jobs/internship", 2, models.OpportunityTypeInternship}, {"yandex", "yandex_data", "Young&&Yandex: стажировка в аналитике данных", "https://yandex.ru/jobs/", "https://yandex.ru/jobs/internship", 2, models.OpportunityTypeInternship},
		{"vk", "vk_dev", "Карьера в VK", "https://vk.company/ru/", "", 2, models.OpportunityTypeInternship}, {"vk", "vk_product", "Карьера в VK", "https://vk.company/ru/", "", 2, models.OpportunityTypeInternship},
		{"sber", "sber_data", "Студенческие программы Сбера", "https://sberstudent.ru/", "", 2, models.OpportunityTypeInternship}, {"sber", "sber_dev", "Студенческие программы Сбера", "https://sberstudent.ru/", "", 2, models.OpportunityTypeInternship},
		{"mts", "mts_dev", "Карьера в МТС", "https://job.mts.ru/", "", 2, models.OpportunityTypeInternship}, {"mts", "mts_telecom", "Карьера в МТС", "https://job.mts.ru/", "", 2, models.OpportunityTypeInternship},
		{"avito", "avito_data", "Карьера в Авито", "https://career.avito.com/", "", 2, models.OpportunityTypeInternship}, {"avito", "avito_dev", "Карьера в Авито", "https://career.avito.com/", "", 2, models.OpportunityTypeInternship},
		{"rosatom", "rosatom_nuclear", "Карьера в Росатоме", "https://rosatom.ru/career/", "", 3, models.OpportunityTypeInternship}, {"rosatom", "rosatom_it", "Карьера в Росатоме", "https://rosatom.ru/career/", "", 2, models.OpportunityTypeInternship},
		{"rzd", "rzd_transport", "Карьерный портал РЖД", "https://team.rzd.ru/", "", 3, models.OpportunityTypePractice}, {"rzd", "rzd_it", "Карьерный портал РЖД", "https://team.rzd.ru/", "", 2, models.OpportunityTypeInternship},
	}
	for _, item := range data {
		c, d := companies[item.company], directions[item.direction]
		regionID := moscowID
		value := models.CompanyOpportunity{CompanyID: c.ID, CareerDirectionID: d.ID, Type: item.kind, Name: item.name, Description: "Официальный карьерный/студенческий портал; доступность конкретного набора проверяется по ссылке.", URL: item.portal, ActiveListingURL: item.listing, SourceCheckedAt: &checked, WorkFormat: "onsite", MinStudyYear: item.course, RegionID: &regionID, IsActive: true}
		if err := tx.Where("company_id = ? AND career_direction_id = ? AND name = ?", value.CompanyID, value.CareerDirectionID, value.Name).Assign(value).FirstOrCreate(&value).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedMoscowRoadmapTemplates(tx *gorm.DB, directions map[string]models.CareerDirection) error {
	steps := []struct{ typ, title string }{{models.RoadmapStepTypeChooseOrConfirmExams, "Подтвердить набор ЕГЭ"}, {models.RoadmapStepTypePrepareForExams, "Подготовиться к ЕГЭ"}, {models.RoadmapStepTypePassExams, "Сдать ЕГЭ"}, {models.RoadmapStepTypeChooseUniversity, "Выбрать вузы и образовательные программы"}, {models.RoadmapStepTypeSubmitAdmissionDocuments, "Подать документы"}, {models.RoadmapStepTypeConfirmEnrollment, "Подтвердить зачисление"}, {models.RoadmapStepTypeLearnAtUniversity, "Учиться в выбранном вузе"}, {models.RoadmapStepTypeEmployerExperience, "Получить практический опыт у работодателя"}, {models.RoadmapStepTypeApplyToEmployer, "Подать заявку в компанию"}}
	for _, direction := range directions {
		template := models.RoadmapTemplate{CareerDirectionID: direction.ID, Name: "Московский roadmap: " + direction.Name, Version: 1, IsActive: true}
		if err := tx.Where("career_direction_id = ? AND version = ?", direction.ID, 1).Assign(template).FirstOrCreate(&template).Error; err != nil {
			return err
		}
		if err := upsertMoscowTemplateSteps(tx, template.ID, steps); err != nil {
			return err
		}
	}
	return nil
}

func upsertMoscowTemplateSteps(tx *gorm.DB, templateID int64, steps []struct{ typ, title string }) error {
	for i, step := range steps {
		value := models.RoadmapTemplateStep{RoadmapTemplateID: templateID, OrderNo: int16(i + 1), StepType: step.typ, Title: step.title, Description: "Шаг roadmap для подтверждённого московского каталога."}
		if err := tx.Where("roadmap_template_id = ? AND step_type = ?", templateID, step.typ).Assign(value).FirstOrCreate(&value).Error; err != nil {
			return err
		}
	}
	return nil
}

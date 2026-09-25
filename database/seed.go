package database

import (
	"fmt"
	"os"
	"strings"

	"efr_bot/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DemoSeedEnabled включает заполнение тестовых данных только в demo-окружении.
func DemoSeedEnabled() bool {
	return strings.EqualFold(os.Getenv("APP_ENV"), "demo") || strings.EqualFold(os.Getenv("SEED_DEMO_DATA"), "true")
}

// SeedDemoData заполняет связный демонстрационный каталог для проверки сценариев MVP.
// Названия организаций и ОП взяты как ориентир из открытых источников. Связи,
// проходные баллы, возможности и требования намеренно тестовые и не заменяют
// официальные правила приема или вакансии.
func SeedDemoData(db *gorm.DB) error {
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

		companies, err := seedCompanies(tx)
		if err != nil {
			return err
		}
		directions, err := seedDirections(tx, companies, tags)
		if err != nil {
			return err
		}
		programs, err := seedEducation(tx, regions, subjects, directions)
		if err != nil {
			return err
		}
		if err := seedOpportunities(tx, regions, companies, directions); err != nil {
			return err
		}
		if err := seedRoadmapTemplates(tx, directions); err != nil {
			return err
		}
		if err := seedAdmissionRules(tx); err != nil {
			return err
		}
		_ = programs
		return nil
	})
}

func seedRegions(tx *gorm.DB) (map[string]models.Region, error) {
	return seedNamed(tx, []string{
		"Пермский край", "Свердловская область", "Москва", "Республика Татарстан",
		"Санкт-Петербург", "Новосибирская область",
	}, func(name string) models.Region { return models.Region{Name: name} }, func(v models.Region) string { return v.Name })
}

func seedTags(tx *gorm.DB) (map[string]models.InterestTag, error) {
	return seedNamed(tx, []string{
		"Программирование", "Аналитика", "Инженерия", "Исследования", "Коммуникация",
		"Математика", "Физика", "Английский язык", "Управление продуктом",
		"Техническая документация", "Производство", "Химия", "Биология", "Медицина",
		"Экология", "Логистика", "Дизайн",
	}, func(name string) models.InterestTag { return models.InterestTag{Name: name} }, func(v models.InterestTag) string { return v.Name })
}

func seedSubjects(tx *gorm.DB) (map[string]models.ExamSubject, error) {
	return seedNamed(tx, []string{
		"Русский язык", "Математика (профильная)", "Информатика", "Физика",
		"Обществознание", "Английский язык", "Химия", "Биология", "География",
		"История", "Литература",
	}, func(name string) models.ExamSubject { return models.ExamSubject{Name: name} }, func(v models.ExamSubject) string { return v.Name })
}

func seedNamed[T interface {
	models.Region | models.InterestTag | models.ExamSubject
}](tx *gorm.DB, names []string, makeValue func(string) T, nameOf func(T) string) (map[string]T, error) {
	result := make(map[string]T, len(names))
	for _, name := range names {
		value := makeValue(name)
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&value).Error; err != nil {
			return nil, err
		}
		if err := tx.Where("name = ?", name).First(&value).Error; err != nil {
			return nil, err
		}
		result[nameOf(value)] = value
	}
	return result, nil
}

func seedCompanies(tx *gorm.DB) (map[string]models.Company, error) {
	data := []models.Company{
		{Name: "Т1", Description: "Демонстрационная запись работодателя: данные не являются описанием актуальных вакансий."},
		{Name: "Яндекс", Description: "Демонстрационная запись работодателя: данные не являются описанием актуальных вакансий."},
		{Name: "ПАО «КАМАЗ»", Description: "Демонстрационная запись работодателя: данные не являются описанием актуальных вакансий."},
		{Name: "СИБУР", Description: "Демонстрационная запись работодателя: данные не являются описанием актуальных вакансий."},
		{Name: "Госкорпорация «Росатом»", Description: "Демонстрационная запись работодателя: данные не являются описанием актуальных вакансий."},
		{Name: "Группа компаний «МЕДСИ»", Description: "Демонстрационная запись работодателя: данные не являются описанием актуальных вакансий."},
		{Name: "ОАО «РЖД»", Description: "Демонстрационная запись работодателя: данные не являются описанием актуальных вакансий."},
	}
	result := make(map[string]models.Company, len(data))
	for _, value := range data {
		if err := tx.Where("name = ?", value.Name).Assign(value).FirstOrCreate(&value).Error; err != nil {
			return nil, err
		}
		result[value.Name] = value
	}
	return result, nil
}

type directionSeed struct {
	key, company, name string
	tags               map[string]float64
}

func seedDirections(tx *gorm.DB, companies map[string]models.Company, tags map[string]models.InterestTag) (map[string]models.CareerDirection, error) {
	data := []directionSeed{
		{"t1_dev", "Т1", "Разработчик программного обеспечения", map[string]float64{"Программирование": 1, "Математика": .7, "Английский язык": .3}},
		{"t1_analyst", "Т1", "Системный аналитик", map[string]float64{"Аналитика": 1, "Коммуникация": .8, "Математика": .5}},
		{"t1_product", "Т1", "Менеджер продукта", map[string]float64{"Управление продуктом": 1, "Аналитика": .8, "Коммуникация": .7}},
		{"t1_writer", "Т1", "Технический писатель", map[string]float64{"Техническая документация": 1, "Коммуникация": .9, "Английский язык": .5}},
		{"yandex_backend", "Яндекс", "Backend-разработчик", map[string]float64{"Программирование": 1, "Математика": .8, "Исследования": .4}},
		{"yandex_data", "Яндекс", "Аналитик данных", map[string]float64{"Аналитика": 1, "Математика": .9, "Исследования": .7}},
		{"yandex_product", "Яндекс", "Менеджер продукта", map[string]float64{"Управление продуктом": 1, "Аналитика": .8, "Коммуникация": .7}},
		{"yandex_writer", "Яндекс", "Технический писатель", map[string]float64{"Техническая документация": 1, "Коммуникация": .9, "Программирование": .4}},
		{"kamaz_auto", "ПАО «КАМАЗ»", "Инженер по автоматизации", map[string]float64{"Инженерия": 1, "Физика": .9, "Математика": .7}},
		{"kamaz_design", "ПАО «КАМАЗ»", "Инженер-конструктор", map[string]float64{"Инженерия": 1, "Дизайн": .7, "Физика": .7}},
		{"kamaz_process", "ПАО «КАМАЗ»", "Инженер-технолог производства", map[string]float64{"Производство": 1, "Инженерия": .8, "Химия": .4}},
		{"sibur_process", "СИБУР", "Инженер-химик-технолог", map[string]float64{"Химия": 1, "Производство": .9, "Инженерия": .6}},
		{"sibur_lab", "СИБУР", "Химик-лаборант", map[string]float64{"Химия": 1, "Исследования": .9, "Биология": .3}},
		{"sibur_data", "СИБУР", "Аналитик данных производства", map[string]float64{"Аналитика": 1, "Производство": .8, "Математика": .7}},
		{"rosatom_nuclear", "Госкорпорация «Росатом»", "Инженер атомной энергетики", map[string]float64{"Инженерия": 1, "Физика": 1, "Исследования": .6}},
		{"rosatom_safety", "Госкорпорация «Росатом»", "Инженер по радиационной безопасности", map[string]float64{"Физика": 1, "Экология": .8, "Инженерия": .7}},
		{"rosatom_writer", "Госкорпорация «Росатом»", "Специалист по техническим коммуникациям", map[string]float64{"Техническая документация": 1, "Коммуникация": .9, "Английский язык": .5}},
		{"medsi_doctor", "Группа компаний «МЕДСИ»", "Врач-лечебник", map[string]float64{"Медицина": 1, "Биология": .9, "Химия": .7}},
		{"medsi_lab", "Группа компаний «МЕДСИ»", "Специалист медицинской лаборатории", map[string]float64{"Медицина": 1, "Биология": .9, "Исследования": .7}},
		{"medsi_analytics", "Группа компаний «МЕДСИ»", "Аналитик медицинских данных", map[string]float64{"Аналитика": 1, "Медицина": .8, "Математика": .7}},
		{"rzd_transport", "ОАО «РЖД»", "Инженер транспортных систем", map[string]float64{"Инженерия": 1, "Логистика": .9, "Физика": .6}},
		{"rzd_logistics", "ОАО «РЖД»", "Логистический аналитик", map[string]float64{"Логистика": 1, "Аналитика": .9, "Математика": .6}},
		{"rzd_software", "ОАО «РЖД»", "Разработчик цифровых транспортных систем", map[string]float64{"Программирование": 1, "Логистика": .8, "Математика": .6}},
	}
	result := make(map[string]models.CareerDirection, len(data))
	for _, item := range data {
		company := companies[item.company]
		direction := models.CareerDirection{CompanyID: company.ID, Name: item.name, Description: "Демонстрационное карьерное направление для MVP."}
		if err := tx.Where("company_id = ? AND name = ?", company.ID, item.name).Assign(direction).FirstOrCreate(&direction).Error; err != nil {
			return nil, err
		}
		for tagName, weight := range item.tags {
			link := models.CareerDirectionInterestTag{CareerDirectionID: direction.ID, InterestTagID: tags[tagName].ID, Weight: weight}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "career_direction_id"}, {Name: "interest_tag_id"}}, DoUpdates: clause.AssignmentColumns([]string{"weight"})}).Create(&link).Error; err != nil {
				return nil, err
			}
		}
		result[item.key] = direction
	}
	return result, nil
}

type programSeed struct {
	key, region, university, code, name string
	directions                          []string
	combinations                        [][]string
	budget, paid                        int16
}

func seedEducation(tx *gorm.DB, regions map[string]models.Region, subjects map[string]models.ExamSubject, directions map[string]models.CareerDirection) (map[string]models.EducationProgram, error) {
	data := []programSeed{
		{"pstu_pe", "Пермский край", "Пермский национальный исследовательский политехнический университет (ПНИПУ)", "09.03.04", "Программная инженерия и искусственный интеллект", []string{"t1_dev", "yandex_backend"}, [][]string{{"Русский язык", "Математика (профильная)", "Информатика"}, {"Русский язык", "Математика (профильная)", "Физика"}}, 245, 175},
		{"psu_cs", "Пермский край", "Пермский государственный национальный исследовательский университет (ПГНИУ)", "09.03.01", "Информатика и вычислительная техника", []string{"t1_dev", "yandex_backend", "yandex_data"}, [][]string{{"Русский язык", "Математика (профильная)", "Информатика"}}, 235, 165},
		{"urfu_pe", "Свердловская область", "Уральский федеральный университет имени первого Президента России Б.Н. Ельцина (УрФУ)", "09.03.04", "Программная инженерия", []string{"t1_dev", "yandex_backend"}, [][]string{{"Русский язык", "Математика (профильная)", "Информатика"}, {"Русский язык", "Математика (профильная)", "Физика"}}, 264, 180},
		{"urfu_auto", "Свердловская область", "Уральский федеральный университет имени первого Президента России Б.Н. Ельцина (УрФУ)", "15.03.04", "Автоматизация технологических процессов и производств", []string{"kamaz_auto"}, [][]string{{"Русский язык", "Математика (профильная)", "Физика"}}, 210, 155},
		{"urfu_mech", "Свердловская область", "Уральский федеральный университет имени первого Президента России Б.Н. Ельцина (УрФУ)", "15.03.01", "Машиностроение", []string{"kamaz_design", "kamaz_process", "sibur_process"}, [][]string{{"Русский язык", "Математика (профильная)", "Физика"}}, 205, 150},
		{"hse_bi", "Москва", "Национальный исследовательский университет «Высшая школа экономики» (НИУ ВШЭ)", "38.03.05", "Бизнес-информатика", []string{"t1_analyst", "yandex_data"}, [][]string{{"Русский язык", "Математика (профильная)", "Обществознание"}, {"Русский язык", "Математика (профильная)", "Английский язык"}}, 285, 245},
		{"hse_management", "Москва", "Национальный исследовательский университет «Высшая школа экономики» (НИУ ВШЭ)", "38.03.02", "Менеджмент", []string{"t1_product", "yandex_product", "rzd_logistics"}, [][]string{{"Русский язык", "Математика (профильная)", "Обществознание"}, {"Русский язык", "Математика (профильная)", "Английский язык"}}, 275, 230},
		{"hse_media", "Москва", "Национальный исследовательский университет «Высшая школа экономики» (НИУ ВШЭ)", "42.03.01", "Реклама и связи с общественностью", []string{"t1_writer", "yandex_writer", "rosatom_writer"}, [][]string{{"Русский язык", "Литература", "Обществознание"}, {"Русский язык", "Английский язык", "Обществознание"}}, 260, 210},
		{"pstu_chem", "Пермский край", "Пермский национальный исследовательский политехнический университет (ПНИПУ)", "18.03.01", "Химическая технология", []string{"sibur_process", "sibur_lab", "kamaz_process"}, [][]string{{"Русский язык", "Математика (профильная)", "Химия"}, {"Русский язык", "Математика (профильная)", "Физика"}}, 198, 145},
		{"kfu_chem", "Республика Татарстан", "Казанский (Приволжский) федеральный университет (КФУ)", "18.03.01", "Химическая технология", []string{"sibur_process", "sibur_lab"}, [][]string{{"Русский язык", "Математика (профильная)", "Химия"}, {"Русский язык", "Математика (профильная)", "Физика"}}, 225, 165},
		{"kfu_ecology", "Республика Татарстан", "Казанский (Приволжский) федеральный университет (КФУ)", "05.03.06", "Экология и природопользование", []string{"rosatom_safety", "sibur_lab"}, [][]string{{"Русский язык", "География", "Математика (профильная)"}, {"Русский язык", "Химия", "Биология"}}, 190, 140},
		{"mephi_nuclear", "Москва", "Национальный исследовательский ядерный университет «МИФИ»", "14.03.02", "Ядерные физика и технологии", []string{"rosatom_nuclear", "rosatom_safety"}, [][]string{{"Русский язык", "Математика (профильная)", "Физика"}, {"Русский язык", "Математика (профильная)", "Информатика"}}, 270, 215},
		{"sechenov_medicine", "Москва", "Первый Московский государственный медицинский университет имени И. М. Сеченова", "31.05.01", "Лечебное дело", []string{"medsi_doctor"}, [][]string{{"Русский язык", "Химия", "Биология"}}, 290, 240},
		{"sechenov_biophysics", "Москва", "Первый Московский государственный медицинский университет имени И. М. Сеченова", "30.05.02", "Медицинская биофизика", []string{"medsi_lab", "medsi_analytics"}, [][]string{{"Русский язык", "Математика (профильная)", "Физика", "Биология"}, {"Русский язык", "Химия", "Биология"}}, 260, 210},
		{"miit_transport", "Москва", "Российский университет транспорта (РУТ (МИИТ))", "23.03.01", "Технология транспортных процессов", []string{"rzd_transport", "rzd_logistics"}, [][]string{{"Русский язык", "Математика (профильная)", "Физика"}, {"Русский язык", "Математика (профильная)", "Обществознание"}}, 205, 155},
		{"miit_software", "Москва", "Российский университет транспорта (РУТ (МИИТ))", "09.03.01", "Информатика и вычислительная техника", []string{"rzd_software", "sibur_data"}, [][]string{{"Русский язык", "Математика (профильная)", "Информатика"}}, 215, 165},
		{"spbpu_design", "Санкт-Петербург", "Санкт-Петербургский политехнический университет Петра Великого", "15.03.01", "Машиностроение", []string{"kamaz_design", "rzd_transport"}, [][]string{{"Русский язык", "Математика (профильная)", "Физика"}}, 230, 170},
		{"nsu_biotech", "Новосибирская область", "Новосибирский национальный исследовательский государственный университет (НГУ)", "19.03.01", "Биотехнология", []string{"medsi_lab", "sibur_lab"}, [][]string{{"Русский язык", "Математика (профильная)", "Биология"}, {"Русский язык", "Химия", "Биология"}}, 235, 175},
	}
	result := make(map[string]models.EducationProgram, len(data))
	for _, item := range data {
		region := regions[item.region]
		university := models.University{RegionID: region.ID, Name: item.university, Description: "Демонстрационная запись: условия и программы требуют проверки на официальном сайте вуза."}
		if err := tx.Where("region_id = ? AND name = ?", region.ID, item.university).Assign(university).FirstOrCreate(&university).Error; err != nil {
			return nil, err
		}
		program := models.EducationProgram{UniversityID: university.ID, Code: item.code, Name: item.name, Description: "Демонстрационная программа для проверки сценариев MVP."}
		if err := tx.Where("university_id = ? AND code = ?", university.ID, item.code).Assign(program).FirstOrCreate(&program).Error; err != nil {
			return nil, err
		}
		for _, directionKey := range item.directions {
			link := models.CareerDirectionEducationProgram{CareerDirectionID: directions[directionKey].ID, EducationProgramID: program.ID}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&link).Error; err != nil {
				return nil, err
			}
		}
		for _, year := range []int16{2026} {
			for index, names := range item.combinations {
				combinationName := fmt.Sprintf("Демо-набор %d: %s", index+1, strings.Join(names, " + "))
				combination := models.ExamCombination{EducationProgramID: program.ID, AdmissionYear: year, Name: &combinationName}
				if err := tx.Where("education_program_id = ? AND admission_year = ? AND name = ?", program.ID, year, combinationName).FirstOrCreate(&combination).Error; err != nil {
					return nil, err
				}
				for _, subjectName := range names {
					minimum := demoMinimumScore(subjectName)
					entry := models.ExamCombinationItem{ExamCombinationID: combination.ID, ExamSubjectID: subjects[subjectName].ID, MinScore: &minimum}
					if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "exam_combination_id"}, {Name: "exam_subject_id"}}, DoUpdates: clause.AssignmentColumns([]string{"min_score"})}).Create(&entry).Error; err != nil {
						return nil, err
					}
				}
			}
			budget, paid := item.budget, item.paid
			score := models.AdmissionScoreHistory{EducationProgramID: program.ID, AdmissionYear: year - 1, BudgetPassingScore: &budget, PaidPassingScore: &paid}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "education_program_id"}, {Name: "admission_year"}}, DoUpdates: clause.AssignmentColumns([]string{"budget_passing_score", "paid_passing_score"})}).Create(&score).Error; err != nil {
				return nil, err
			}
		}
		result[item.key] = program
	}
	return result, nil
}

func demoMinimumScore(subject string) int16 {
	if subject == "Информатика" {
		return 44
	}
	if subject == "Физика" {
		return 41
	}
	return 40
}

func seedAdmissionRules(tx *gorm.DB) error {
	for _, year := range []int16{2026} {
		rule := models.AdmissionCampaignRule{AdmissionYear: year, MaxUniversities: 5, MaxProgramsPerUniversity: 5}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "admission_year"}}, DoUpdates: clause.AssignmentColumns([]string{"max_universities", "max_programs_per_university"})}).Create(&rule).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedOpportunities(tx *gorm.DB, regions map[string]models.Region, companies map[string]models.Company, directions map[string]models.CareerDirection) error {
	type opportunitySeed struct {
		company, direction, name, kind string
		course                         int16
		region                         string
	}
	data := []opportunitySeed{
		{"Т1", "t1_dev", "Демо-стажировка для разработчика", models.OpportunityTypeInternship, 3, "Пермский край"},
		{"Т1", "t1_analyst", "Демо-практика системного аналитика", models.OpportunityTypePractice, 3, "Москва"},
		{"Т1", "t1_product", "Демо-проект по управлению продуктом", models.OpportunityTypeProject, 3, "Москва"},
		{"Т1", "t1_writer", "Демо-практика технического писателя", models.OpportunityTypePractice, 2, "Свердловская область"},
		{"Яндекс", "yandex_backend", "Демо-проект для backend-разработчика", models.OpportunityTypeProject, 3, "Свердловская область"},
		{"Яндекс", "yandex_data", "Демо-хакатон по анализу данных", models.OpportunityTypeHackathon, 2, "Москва"},
		{"Яндекс", "yandex_product", "Демо-стажировка менеджера продукта", models.OpportunityTypeInternship, 3, "Москва"},
		{"Яндекс", "yandex_writer", "Демо-школа технической документации", models.OpportunityTypeProject, 2, "Москва"},
		{"ПАО «КАМАЗ»", "kamaz_auto", "Демо-практика по автоматизации", models.OpportunityTypePractice, 3, "Пермский край"},
		{"ПАО «КАМАЗ»", "kamaz_design", "Демо-практика инженера-конструктора", models.OpportunityTypePractice, 3, "Республика Татарстан"},
		{"ПАО «КАМАЗ»", "kamaz_process", "Демо-проект по технологии производства", models.OpportunityTypeProject, 3, "Республика Татарстан"},
		{"СИБУР", "sibur_process", "Демо-стажировка инженера-химика", models.OpportunityTypeInternship, 3, "Республика Татарстан"},
		{"СИБУР", "sibur_lab", "Демо-практика в химической лаборатории", models.OpportunityTypePractice, 2, "Пермский край"},
		{"СИБУР", "sibur_data", "Демо-хакатон по данным производства", models.OpportunityTypeHackathon, 2, "Республика Татарстан"},
		{"Госкорпорация «Росатом»", "rosatom_nuclear", "Демо-стажировка в инженерном дивизионе", models.OpportunityTypeInternship, 3, "Москва"},
		{"Госкорпорация «Росатом»", "rosatom_safety", "Демо-проект по промышленной безопасности", models.OpportunityTypeProject, 3, "Свердловская область"},
		{"Госкорпорация «Росатом»", "rosatom_writer", "Демо-практика технических коммуникаций", models.OpportunityTypePractice, 2, "Москва"},
		{"Группа компаний «МЕДСИ»", "medsi_doctor", "Демо-практика в клиническом отделении", models.OpportunityTypePractice, 4, "Москва"},
		{"Группа компаний «МЕДСИ»", "medsi_lab", "Демо-стажировка в медицинской лаборатории", models.OpportunityTypeInternship, 3, "Москва"},
		{"Группа компаний «МЕДСИ»", "medsi_analytics", "Демо-проект по медицинской аналитике", models.OpportunityTypeProject, 3, "Москва"},
		{"ОАО «РЖД»", "rzd_transport", "Демо-практика транспортного инженера", models.OpportunityTypePractice, 3, "Москва"},
		{"ОАО «РЖД»", "rzd_logistics", "Демо-стажировка в логистической аналитике", models.OpportunityTypeInternship, 3, "Санкт-Петербург"},
		{"ОАО «РЖД»", "rzd_software", "Демо-хакатон цифровых транспортных систем", models.OpportunityTypeHackathon, 2, "Новосибирская область"},
	}
	for _, item := range data {
		regionID := regions[item.region].ID
		value := models.CompanyOpportunity{CompanyID: companies[item.company].ID, CareerDirectionID: directions[item.direction].ID, Type: item.kind, Name: item.name, Description: "Тестовая возможность для демонстрации roadmap; не является объявлением работодателя.", MinStudyYear: item.course, RegionID: &regionID, IsActive: true}
		if err := tx.Where("company_id = ? AND career_direction_id = ? AND name = ?", value.CompanyID, value.CareerDirectionID, value.Name).Assign(value).FirstOrCreate(&value).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedRoadmapTemplates(tx *gorm.DB, directions map[string]models.CareerDirection) error {
	for _, direction := range directions {
		template := models.RoadmapTemplate{CareerDirectionID: direction.ID, Name: "Демо-roadmap: " + direction.Name, Version: 1, IsActive: true}
		if err := tx.Where("career_direction_id = ? AND version = ?", direction.ID, 1).Assign(template).FirstOrCreate(&template).Error; err != nil {
			return err
		}
		steps := []struct{ typ, title string }{
			{models.RoadmapStepTypeChooseOrConfirmExams, "Подтвердить набор ЕГЭ"}, {models.RoadmapStepTypePrepareForExams, "Подготовиться к ЕГЭ"}, {models.RoadmapStepTypePassExams, "Сдать ЕГЭ"}, {models.RoadmapStepTypeChooseUniversity, "Выбрать вузы и образовательные программы"}, {models.RoadmapStepTypeSubmitAdmissionDocuments, "Подать документы"}, {models.RoadmapStepTypeConfirmEnrollment, "Подтвердить зачисление"}, {models.RoadmapStepTypeLearnAtUniversity, "Учиться в выбранном вузе"}, {models.RoadmapStepTypeEmployerExperience, "Получить практический опыт у работодателя"}, {models.RoadmapStepTypeApplyToEmployer, "Подать документы в компанию"},
		}
		if err := upsertRoadmapTemplateSteps(tx, template.ID, steps); err != nil {
			return err
		}
	}
	return nil
}

func upsertRoadmapTemplateSteps(tx *gorm.DB, templateID int64, steps []struct{ typ, title string }) error {
	var existing []models.RoadmapTemplateStep
	if err := tx.Where("roadmap_template_id = ?", templateID).Find(&existing).Error; err != nil {
		return err
	}
	byType := make(map[string]models.RoadmapTemplateStep, len(existing))
	for _, step := range existing {
		byType[step.StepType] = step
	}
	if len(existing) > 0 {
		if err := tx.Model(&models.RoadmapTemplateStep{}).Where("roadmap_template_id = ?", templateID).Update("order_no", gorm.Expr("order_no + 100")).Error; err != nil {
			return err
		}
	}
	for index, step := range steps {
		value, found := byType[step.typ]
		if !found {
			value = models.RoadmapTemplateStep{RoadmapTemplateID: templateID}
		}
		value.OrderNo = int16(index + 1)
		value.StepType = step.typ
		value.Title = step.title
		value.Description = "Демонстрационный шаг roadmap для MVP."
		if err := tx.Save(&value).Error; err != nil {
			return err
		}
	}
	return nil
}

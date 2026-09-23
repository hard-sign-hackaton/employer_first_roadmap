// Package dto содержит структуры обмена данными между обработчиками бота и сервисами.
// DTO не повторяют GORM-модели: они описывают только данные конкретного действия пользователя
// или ответ, который нужно показать в сценарии.
package dto

// IDNameResponse используется для компактного представления справочников и вариантов выбора.
type IDNameResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// RegionResponse — регион из справочника.
type RegionResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// ExamSubjectResponse — предмет ЕГЭ из справочника.
type ExamSubjectResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// ErrorResponse описывает ошибку, которую можно показать пользователю или обработать контроллером.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

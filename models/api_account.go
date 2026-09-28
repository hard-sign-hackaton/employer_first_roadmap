package models

import "time"

// APIAccount — учётная запись для внешнего API. Секрет токена не хранится:
// в базе находится только его SHA-256 хэш.
type APIAccount struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Login     string    `json:"login" gorm:"size:255;not null;uniqueIndex"`
	TokenHash string    `json:"-" gorm:"size:64;not null;uniqueIndex"`
	Role      string    `json:"role" gorm:"size:16;not null;check:role IN ('admin','employer')"`
	CompanyID *int64    `json:"company_id,omitempty" gorm:"index"`
	IsActive  bool      `json:"is_active" gorm:"not null;default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Company *Company `json:"company,omitempty" gorm:"foreignKey:CompanyID"`
}

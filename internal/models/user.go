package models

import "time"

// User — модель пользователя (по доке GORM: используем теги для полей)
type User struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TelegramID   int64     `gorm:"uniqueIndex;not null" json:"telegram_id"`
	FirstName    string    `gorm:"size:255" json:"first_name"`
	LastName     string    `gorm:"size:255" json:"last_name"`
	Username     string    `gorm:"size:255;index" json:"username"`
	LanguageCode string    `gorm:"size:10" json:"language_code"`
	IsPremium    bool      `gorm:"default:false" json:"is_premium"`
	PhotoURL     string    `gorm:"size:512" json:"photo_url"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Subscription *Subscription `gorm:"constraint:OnDelete:CASCADE" json:"subscription"`
}

package models

import "time"

// User — модель пользователя (по доке GORM: используем теги для полей)
type User struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"` // int64 для совместимости с Postgres BIGINT (см. доку GORM на типы)
	TelegramID   int64     `gorm:"uniqueIndex;not null"`     // Уникальный индекс, not null
	FirstName    string    `gorm:"size:255"`                 // Размер строки (max 255)
	LastName     string    `gorm:"size:255"`
	Username     string    `gorm:"size:255;index"` // Индекс для быстрого поиска
	LanguageCode string    `gorm:"size:10"`
	IsPremium    bool      `gorm:"default:false"` // Дефолт false
	PhotoURL     string    `gorm:"size:512"`
	CreatedAt    time.Time `gorm:"autoCreateTime"` // Автозаполнение
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`

	Subscription *Subscription `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"` // Связь (каскадное обновление/удаление, по доке GORM constraints)
}

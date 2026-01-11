package models

import "time"

// Subscription — модель подписки (по доке GORM: foreignKey по умолчанию по имени UserID)
type Subscription struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"` // int64
	UserID    int64     `gorm:"index;not null"`           // FK на User.ID (GORM сам поймёт)
	Plan      string    `gorm:"size:50;default:'free'"`   // Дефолт 'free'
	Active    bool      `gorm:"default:false"`
	StartDate time.Time `gorm:"index"` // Индекс для поиска по датам
	EndDate   time.Time `gorm:"index"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

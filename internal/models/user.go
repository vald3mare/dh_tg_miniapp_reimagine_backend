package models

import "time"

type User struct {
	ID           uint  `gorm:"primaryKey"`
	TelegramID   int64 `gorm:"uniqueIndex;not null"`
	FirstName    string
	LastName     string
	Username     string
	LanguageCode string
	IsPremium    bool
	PhotoURL     string
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`

	// Связь с подпиской (один-к-одному)
	Subscription *Subscription `gorm:"foreignKey:UserID"`
}

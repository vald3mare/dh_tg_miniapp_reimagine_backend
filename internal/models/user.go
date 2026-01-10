package models

import "time"

type User struct {
	ID           uint   `gorm:"primaryKey"`
	TelegramID   int64  `gorm:"uniqueIndex;not null"`
	FirstName    string `gorm:"size:255"`
	LastName     string `gorm:"size:255"`
	Username     string `gorm:"size:255"`
	LanguageCode string `gorm:"size:10"`
	IsPremium    bool
	PhotoURL     string    `gorm:"size:512"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`

	// Связь с подпиской (один-к-одному)
	Subscription *Subscription `gorm:"foreignKey:UserID"`
}

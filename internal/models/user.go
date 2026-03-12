package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model // Embed для ID, CreatedAt, UpdatedAt, DeletedAt
	TelegramID uint `gorm:"uniqueIndex;not null"`
	FirstName  string
	LastName   string
	Username   string
	IsPremium  bool
	PhotoURL   string

	Subscription *Subscription `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

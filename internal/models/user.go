package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model // Embed для ID, CreatedAt, UpdatedAt, DeletedAt
	TelegramID      uint    `gorm:"uniqueIndex;not null"`
	FirstName       string
	LastName        string
	Username        string
	IsPremium       bool
	PhotoURL        string
	Role            string  `gorm:"default:'customer'"`
	Rating          float64 `gorm:"default:0"`
	OrdersCompleted int     `gorm:"default:0"`

	Subscription    *Subscription    `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	UserAchievements []UserAchievement `gorm:"foreignKey:UserID"`
}

package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	TelegramID      uint    `gorm:"uniqueIndex;not null" json:"telegram_id"`
	FirstName       string  `json:"first_name"`
	LastName        string  `json:"last_name"`
	Username        string  `json:"username"`
	IsPremium       bool    `json:"is_premium"`
	PhotoURL        string  `json:"photo_url"`
	Role            string  `gorm:"default:'customer'" json:"role"`
	Rating          float64 `gorm:"default:0" json:"rating"`
	OrdersCompleted int     `gorm:"default:0" json:"orders_completed"`

	Subscription     *Subscription     `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	UserAchievements []UserAchievement `gorm:"foreignKey:UserID" json:"-"`
}

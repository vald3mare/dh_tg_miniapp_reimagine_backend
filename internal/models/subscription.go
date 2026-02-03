package models

import (
	"time"

	"gorm.io/gorm"
)

type Subscription struct {
	gorm.Model
	SubscriptionID uint
	UserID         *uint
	User           User   `gorm:"foreignKey:UserID"`
	Plan           string // {тариф1, тариф2 и тд...}
	Active         bool
	StartDate      time.Time
	EndDate        time.Time
	PaymentID      string `gorm:"index"` // ID платежа от ЮKassa
}

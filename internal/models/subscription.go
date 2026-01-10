package models

import "time"

type Subscription struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"uniqueIndex;not null"`
	Plan      string `gorm:"default:'free'"` // free, basic, premium
	Active    bool   `gorm:"default:false"`
	StartDate time.Time
	EndDate   time.Time
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

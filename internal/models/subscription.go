package models

import "time"

type Subscription struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"index"`
	Plan      string `gorm:"size:50;default:'free'"`
	Active    bool   `gorm:"default:false"`
	StartDate time.Time
	EndDate   time.Time `gorm:"index"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

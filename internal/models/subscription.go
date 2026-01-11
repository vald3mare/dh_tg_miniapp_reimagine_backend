package models

import "time"

type Subscription struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"index;not null" json:"user_id"`
	Plan      string    `gorm:"size:50;default:'free'" json:"plan"` // free, premium и т.д.
	Active    bool      `gorm:"default:false" json:"active"`
	StartDate time.Time `gorm:"index" json:"start_date"`
	EndDate   time.Time `gorm:"index" json:"end_date"`
	PaymentID string    `gorm:"size:255" json:"payment_id"` // ID платежа от ЮKassa
	CreatedAt time.Time `gorm:"autoCreateTime;<-:create" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

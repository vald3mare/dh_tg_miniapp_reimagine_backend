package models

import "gorm.io/gorm"

// Payment хранит запись о каждом созданном платеже.
// Статусы зеркалят статусы ЮKassa: pending, waiting_for_capture, succeeded, canceled.
type Payment struct {
	gorm.Model
	UserID      uint        `gorm:"not null;index" json:"user_id"`
	User        User        `gorm:"foreignKey:UserID" json:"-"`
	ItemID      uint        `gorm:"not null" json:"item_id"`
	Item        CatalogItem `gorm:"foreignKey:ItemID" json:"-"`
	YooKassaID  string      `gorm:"uniqueIndex;not null" json:"yookassa_id"`
	Amount      float64     `gorm:"not null" json:"amount"`
	Currency    string      `gorm:"default:'RUB'" json:"currency"`
	Status      string      `gorm:"default:'pending';index" json:"status"`
	Description string      `json:"description"`
}

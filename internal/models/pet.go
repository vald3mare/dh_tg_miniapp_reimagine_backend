package models

import "gorm.io/gorm"

type Pet struct {
	gorm.Model
	UserID uint   `gorm:"not null;index" json:"user_id"`
	Name   string `gorm:"not null" json:"name"`
	Emoji  string `gorm:"default:'🐾'" json:"emoji"`
	Breed  string `json:"breed"`
}

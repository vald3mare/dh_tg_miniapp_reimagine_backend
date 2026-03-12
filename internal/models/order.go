package models

import (
	"time"

	"gorm.io/gorm"
)

type Order struct {
	gorm.Model
	CustomerID      *uint      `gorm:"index"`
	ServiceType     string
	Description     string
	Status          string     `gorm:"default:'open';index"` // open, accepted, done, canceled
	Price           float64
	CustomerName    string
	CustomerContact string
	ScheduledAt     *time.Time
	ExecutorID      *uint      `gorm:"index"`
}

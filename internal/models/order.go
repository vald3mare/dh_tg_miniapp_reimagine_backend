package models

import (
	"time"

	"gorm.io/gorm"
)

type Order struct {
	gorm.Model
	CustomerID      *uint      `gorm:"index" json:"customer_id"`
	ServiceType     string     `json:"service_type"`
	Description     string     `json:"description"`
	Status          string     `gorm:"default:'open';index" json:"status"`
	Price           float64    `json:"price"`
	CustomerName    string     `json:"customer_name"`
	CustomerContact string     `json:"customer_contact"`
	ScheduledAt     *time.Time `json:"scheduled_at"`
	ExecutorID      *uint      `gorm:"index" json:"executor_id"`
}

package models

import "gorm.io/gorm"

type CatalogItem struct {
	gorm.Model          // ID, CreatedAt, UpdatedAt, DeletedAt
	Name        string  `json:"name" gorm:"type:varchar(255);not null"`
	Description string  `json:"description" gorm:"type:text"`
	Price       float64 `json:"price" gorm:"type:decimal(10,2)"`
	ImageURL    string  `json:"image_url" gorm:"type:varchar(512)"`
	Type        string  `json:"type" gorm:"type:varchar(50)"`
}

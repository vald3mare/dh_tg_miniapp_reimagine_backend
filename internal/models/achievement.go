package models

import "gorm.io/gorm"

type Achievement struct {
	gorm.Model
	Key           string `gorm:"uniqueIndex;not null"`
	Name          string
	Description   string
	IconEmoji     string
	ConditionType string // orders_completed, manual
	Threshold     int
}

type UserAchievement struct {
	gorm.Model
	UserID        uint        `gorm:"index;not null"`
	AchievementID uint        `gorm:"index;not null"`
	Achievement   Achievement `gorm:"foreignKey:AchievementID"`
}

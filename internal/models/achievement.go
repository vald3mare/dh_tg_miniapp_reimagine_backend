package models

import "gorm.io/gorm"

type Achievement struct {
	gorm.Model
	Key           string `gorm:"uniqueIndex;not null" json:"key"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	IconEmoji     string `json:"icon_emoji"`
	ConditionType string `json:"condition_type"`
	Threshold     int    `json:"threshold"`
}

type UserAchievement struct {
	gorm.Model
	UserID        uint        `gorm:"index;not null" json:"user_id"`
	AchievementID uint        `gorm:"index;not null" json:"achievement_id"`
	Achievement   Achievement `gorm:"foreignKey:AchievementID" json:"achievement"`
}

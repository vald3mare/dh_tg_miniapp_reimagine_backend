package models

import "gorm.io/gorm"

// ExecutorApplication — заявка на роль исполнителя, поданная через Telegram-бот.
// После прохождения анкеты и тестов бот отправляет данные в бэкенд,
// откуда администратор может одобрить или отклонить заявку.
type ExecutorApplication struct {
	gorm.Model
	TelegramID       uint   `gorm:"index"                      json:"telegram_id"`
	TelegramUsername string `                                  json:"telegram_username"`
	FullName         string `                                  json:"full_name"`
	FormDataJSON     string `gorm:"type:text"                  json:"form_data_json"` // CandidateForm в виде JSON
	Status           string `gorm:"default:'pending';index"    json:"status"`         // pending | approved | rejected
	ReviewNote       string `                                  json:"review_note,omitempty"`
	ReviewedBy       *uint  `                                  json:"reviewed_by,omitempty"` // ID администратора (из таблицы users)
}

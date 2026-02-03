package models

// type User struct {
// 	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
// 	TelegramID   int64     `gorm:"uniqueIndex:idx_users_telegram_id;not null" json:"telegram_id"` // явное имя индекса
// 	FirstName    string    `gorm:"size:255" json:"first_name"`
// 	LastName     string    `gorm:"size:255" json:"last_name"`
// 	Username     string    `gorm:"size:255;index" json:"username"`
// 	LanguageCode string    `gorm:"size:10" json:"language_code"`
// 	IsPremium    bool      `gorm:"default:false" json:"is_premium"`
// 	PhotoURL     string    `gorm:"size:512" json:"photo_url"`
// 	CreatedAt    time.Time `gorm:"autoCreateTime;<-:create" json:"created_at"` // <-:create — заполняется только при создании, не функция
// 	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`           // autoUpdateTime работает нормально

// 	Subscription *Subscription `gorm:"constraint:OnDelete:CASCADE" json:"subscription"`
// }

// type Subscription struct {
// 	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
// 	UserID    int64     `gorm:"index;not null" json:"user_id"`
// 	Plan      string    `gorm:"size:50;default:'free'" json:"plan"` // free, premium и т.д.
// 	Active    bool      `gorm:"default:false" json:"active"`
// 	StartDate time.Time `gorm:"index" json:"start_date"`
// 	EndDate   time.Time `gorm:"index" json:"end_date"`
// 	PaymentID string    `gorm:"size:255" json:"payment_id"` // ID платежа от ЮKassa
// 	CreatedAt time.Time `gorm:"autoCreateTime;<-:create" json:"created_at"`
// 	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
// }

// type Subscription struct {
// 	ID     uint
// 	UserID uint
// 	Plan   string // {тариф1, тариф2 и тд...}
// }

// type User struct {
// 	ID         uint
// 	TelegramID uint
// 	FirstName  string
// 	LastName   string
// 	Username   string
// 	IsPremium  bool
// 	PhotoURL   string
// 	CreatedAt  time.Time
// 	UpdatedAt  time.Time

// 	Subscription *Subscription
// }

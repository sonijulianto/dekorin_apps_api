package models

import "time"

// ClientErrorLog menyimpan log error dari aplikasi Flutter (Android APK & Web PWA)
// yang dikelompokkan berdasarkan fingerprint agar mudah dipantau dan diselesaikan.
type ClientErrorLog struct {
	ID              string     `gorm:"primaryKey;type:varchar(64)" json:"id"`
	Fingerprint     string     `gorm:"type:varchar(64);index" json:"fingerprint"`
	Level           string     `gorm:"type:varchar(20);index;default:'error'" json:"level"` // fatal, error, warning
	Platform        string     `gorm:"type:varchar(20);index" json:"platform"`             // android, web, ios
	AppVersion      string     `gorm:"type:varchar(50);index" json:"app_version"`
	Route           string     `gorm:"type:varchar(150);index" json:"route"`
	ErrorType       string     `gorm:"type:varchar(120)" json:"error_type"`
	ErrorMessage    string     `gorm:"type:text" json:"error_message"`
	StackTrace      string     `gorm:"type:text" json:"stack_trace"`
	Breadcrumbs     string     `gorm:"type:text" json:"breadcrumbs"` // JSON array string
	ApiContext      string     `gorm:"type:text" json:"api_context"` // JSON object string
	DeviceInfo      string     `gorm:"type:varchar(255)" json:"device_info"`
	UserID          string     `gorm:"type:varchar(64);index" json:"user_id"`
	CompanyID       string     `gorm:"type:varchar(64);index" json:"company_id"`
	UserRole        string     `gorm:"type:varchar(50)" json:"user_role"`
	IPAddr          string     `gorm:"type:varchar(50)" json:"ip_addr"`
	Status          string     `gorm:"type:varchar(20);index;default:'open'" json:"status"` // open, resolved
	OccurrenceCount int        `gorm:"type:int;default:1" json:"occurrence_count"`
	FirstSeenAt     time.Time  `gorm:"index" json:"first_seen_at"`
	LastSeenAt      time.Time  `gorm:"index" json:"last_seen_at"`
	ResolvedAt      *time.Time `json:"resolved_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

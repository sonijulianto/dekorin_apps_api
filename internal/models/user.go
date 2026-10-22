package models

import "time"

// Entitas User untuk tabel Database (GORM)
type User struct {
	ID        string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	CompanyID string    `gorm:"type:varchar(50);index;default:''" json:"company_id"` // Kosong untuk role 'superadmin'
	FullName  string    `gorm:"type:varchar(100);not null" json:"full_name"`
	Phone     string    `gorm:"type:varchar(25);index;default:''" json:"phone"`
	Email     string    `gorm:"type:varchar(100);default:''" json:"email"`
	Password  string    `gorm:"type:varchar(255);not null" json:"-"` // JSON di-ignore agar password tidak bocor
	Role      string    `gorm:"type:varchar(20);default:'crew';not null" json:"role"` // 'superadmin', 'owner', 'admin', 'crew'
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Company *Company `gorm:"foreignKey:CompanyID" json:"company,omitempty"`
}

// UserResponse adalah format balikan API yang aman dikirim ke Client
type UserResponse struct {
	ID             string `json:"id"`
	CompanyID      string `json:"company_id"`
	CompanyName    string `json:"company_name,omitempty"`
	CompanyAddress string `json:"company_address,omitempty"`
	CompanyPhone   string `json:"company_phone,omitempty"`
	FullName       string `json:"full_name"`
	Phone          string `json:"phone"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	CreatedAt      string `json:"created_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

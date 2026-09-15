package models

import "time"

// Entitas User untuk tabel Database (GORM)
type User struct {
	ID        string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	FullName  string    `gorm:"type:varchar(100);not null" json:"full_name"`
	Email     string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"type:varchar(255);not null" json:"-"` // JSON di-ignore agar password tidak bocor
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserResponse adalah format balikan API yang aman dikirim ke Client
type UserResponse struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

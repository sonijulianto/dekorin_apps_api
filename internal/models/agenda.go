package models

import "time"

// Package mewakili master data paket dekorasi
type Package struct {
	ID          string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	Name        string    `gorm:"type:varchar(150);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	BasePrice   float64   `gorm:"type:decimal(15,2);not null;default:0" json:"base_price"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Agenda mewakili item booking/jadwal acara
type Agenda struct {
	ID            string      `gorm:"primaryKey;type:varchar(50)" json:"id"`
	ClientName    string      `gorm:"type:varchar(150);not null" json:"client_name"`
	ClientPhone   string      `gorm:"type:varchar(50)" json:"client_phone"`
	BackdropTitle string      `gorm:"type:varchar(150);not null" json:"backdrop_title"`
	EventDateTime time.Time   `gorm:"not null" json:"event_date_time"`
	MapsURL       string      `gorm:"type:text" json:"maps_url"`
	PackageID     string      `gorm:"type:varchar(50);not null" json:"package_id"`
	PackageName   string      `gorm:"type:varchar(150)" json:"package_name"`
	Status        string      `gorm:"type:varchar(50);default:'upcoming'" json:"status"` // upcoming, completed, cancelled
	FormToken     string      `gorm:"type:varchar(50);uniqueIndex" json:"form_token"`
	FormStatus    string      `gorm:"type:varchar(50);default:'pending'" json:"form_status"` // pending, filled
	Notes         string      `gorm:"type:text" json:"notes"`
	ClientForm    *ClientForm `gorm:"foreignKey:AgendaID" json:"client_form,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// CreateAgendaRequest adalah payload untuk membuat agenda baru
type CreateAgendaRequest struct {
	ClientName    string `json:"client_name"`
	ClientPhone   string `json:"client_phone"`
	BackdropTitle string `json:"backdrop_title"`
	EventDateTime string `json:"event_date_time"` // Format ISO8601 atau "2006-01-02 15:04"
	MapsURL       string `json:"maps_url"`
	PackageID     string `json:"package_id"`
	Notes         string `json:"notes"`
}

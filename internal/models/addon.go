package models

import "time"

// Addon mewakili master data item tambahan dekorasi
type Addon struct {
	ID          string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	Name        string    `gorm:"type:varchar(150);not null" json:"name"`
	ImageURL    string    `gorm:"type:text" json:"image_url"`
	Price       float64   `gorm:"type:decimal(15,2);not null;default:0" json:"price"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateOrUpdateAddonRequest adalah payload request untuk membuat atau mengubah item tambahan
type CreateOrUpdateAddonRequest struct {
	Name        string  `json:"name"`
	ImageURL    string  `json:"image_url"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

// CreateOrUpdatePackageRequest adalah payload request untuk membuat atau mengubah paket dekorasi
type CreateOrUpdatePackageRequest struct {
	Name        string  `json:"name"`
	ImageURL    string  `json:"image_url"`
	BasePrice   float64 `json:"base_price"`
	Description string  `json:"description"`
}

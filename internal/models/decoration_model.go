package models

import "time"

// DecorationModel mewakili master data model dekorasi (backdrop / pelaminan)
type DecorationModel struct {
	ID          string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	CompanyID   string    `gorm:"type:varchar(50);index;default:'cmp_default'" json:"company_id"`
	Name        string    `gorm:"type:varchar(150);not null" json:"name"`
	ImageURL    string    `gorm:"type:text" json:"image_url"`
	Price       float64   `gorm:"type:decimal(15,2);not null;default:0" json:"price"`
	Width       string    `gorm:"type:varchar(50)" json:"width"` // Contoh: "3 Meter", "4 Meter", dsb.
	Description string    `gorm:"type:text" json:"description"`
	SortOrder   int       `gorm:"type:int;default:0" json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PackageDecorationModel menghubungkan Package dengan Model Dekorasi beserta harga tambahan
type PackageDecorationModel struct {
	ID                string           `gorm:"primaryKey;type:varchar(50)" json:"id"`
	CompanyID         string           `gorm:"type:varchar(50);index;default:'cmp_default'" json:"company_id"`
	PackageID         string           `gorm:"type:varchar(50);index;not null" json:"package_id"`
	DecorationModelID string           `gorm:"type:varchar(50);index;not null" json:"decoration_model_id"`
	AdditionalPrice   float64          `gorm:"type:decimal(15,2);default:0" json:"additional_price"`
	SortOrder         int              `gorm:"type:int;default:0" json:"sort_order"`
	DecorationModel   *DecorationModel `gorm:"foreignKey:DecorationModelID" json:"decoration_model,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

// PackageModelInput adalah payload saat memilih model dekorasi dalam paket
type PackageModelInput struct {
	ModelID         string  `json:"model_id"`
	AdditionalPrice float64 `json:"additional_price"`
}

// CreateOrUpdateDecorationModelRequest adalah payload request untuk membuat atau mengubah model dekorasi
type CreateOrUpdateDecorationModelRequest struct {
	Name        string  `json:"name"`
	ImageURL    string  `json:"image_url"`
	Price       float64 `json:"price"`
	Width       string  `json:"width"`
	Description string  `json:"description"`
}

// ReorderDecorationModelsRequest adalah payload untuk mengatur urutan model dekorasi
type ReorderDecorationModelsRequest struct {
	ModelIDs []string `json:"model_ids"`
}

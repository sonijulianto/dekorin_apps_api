package models

import "time"

// ClientForm menyimpan data isian detail dari client via Web Form
type ClientForm struct {
	ID                string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	AgendaID          string    `gorm:"type:varchar(50);index;not null" json:"agenda_id"`
	PackageID         string    `gorm:"type:varchar(50)" json:"package_id"`
	PackageName       string    `gorm:"type:varchar(150)" json:"package_name"`
	PackagePrice      float64   `gorm:"type:decimal(15,2);default:0" json:"package_price"`
	SelectedAddons    string    `gorm:"type:text" json:"selected_addons"` // Comma-separated atau JSON nama/ID addon
	AddonsPrice       float64   `gorm:"type:decimal(15,2);default:0" json:"addons_price"`
	TotalPrice        float64   `gorm:"type:decimal(15,2);default:0" json:"total_price"`
	EventAddress      string    `gorm:"type:text;not null" json:"event_address"`
	DecorationTheme   string    `gorm:"type:varchar(100)" json:"decoration_theme"`
	ColorPreference   string    `gorm:"type:varchar(100)" json:"color_preference"`
	SpecialRequests   string    `gorm:"type:text" json:"special_requests"`
	ReferencePhotoURL string    `gorm:"type:text" json:"reference_photo_url"`
	SubmittedAt       time.Time `json:"submitted_at"`
}

// SubmitClientFormRequest digunakan saat client mengisi form via Web Form
type SubmitClientFormRequest struct {
	PackageID         string   `json:"package_id"`
	AddonIDs          []string `json:"addon_ids"`
	EventAddress      string   `json:"event_address"`
	DecorationTheme   string   `json:"decoration_theme"`
	ColorPreference   string   `json:"color_preference"`
	SpecialRequests   string   `json:"special_requests"`
	ReferencePhotoURL string   `json:"reference_photo_url"`
}

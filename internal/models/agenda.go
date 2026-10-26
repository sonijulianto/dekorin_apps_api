package models

import "time"

// Package mewakili master data paket dekorasi
type Package struct {
	ID            string                   `gorm:"primaryKey;type:varchar(50)" json:"id"`
	CompanyID     string                   `gorm:"type:varchar(50);index;default:'cmp_default'" json:"company_id"`
	Name          string                   `gorm:"type:varchar(150);not null" json:"name"`
	ImageURL      string                   `gorm:"type:text" json:"image_url"`
	Description   string                   `gorm:"type:text" json:"description"`
	BasePrice     float64                  `gorm:"type:decimal(15,2);not null;default:0" json:"base_price"`
	SortOrder     int                      `gorm:"type:int;default:0" json:"sort_order"`
	PackageModels []PackageDecorationModel `gorm:"foreignKey:PackageID" json:"package_models,omitempty"`
	CreatedAt     time.Time                `json:"created_at"`
	UpdatedAt     time.Time                `json:"updated_at"`
}

// Agenda mewakili item booking/jadwal acara
type Agenda struct {
	ID                   string      `gorm:"primaryKey;type:varchar(50)" json:"id"`
	CompanyID            string      `gorm:"type:varchar(50);index;default:'cmp_default'" json:"company_id"`
	ClientName           string      `gorm:"type:varchar(150);not null" json:"client_name"`
	ClientPhone          string      `gorm:"type:varchar(50)" json:"client_phone"`
	BackdropTitle        string      `gorm:"type:varchar(150)" json:"backdrop_title"`
	EventDateTime        time.Time   `json:"event_date_time"`
	MapsURL              string      `gorm:"type:text" json:"maps_url"`
	PackageID            string      `gorm:"type:varchar(50)" json:"package_id"`
	PackageName          string      `gorm:"type:varchar(150)" json:"package_name"`
	PackagePrice         float64     `gorm:"type:decimal(15,2);default:0" json:"package_price"`
	ModelID              string      `gorm:"type:varchar(50)" json:"model_id"`
	ModelName            string      `gorm:"type:varchar(150)" json:"model_name"`
	ModelWidth           string      `gorm:"type:varchar(50)" json:"model_width"`
	ModelImageURL        string      `gorm:"type:text" json:"model_image_url"`
	ModelAdditionalPrice float64     `gorm:"type:decimal(15,2);default:0" json:"model_additional_price"`
	TransportFee         float64     `gorm:"type:decimal(15,2);default:0" json:"transport_fee"`
	Status               string      `gorm:"type:varchar(50);default:'upcoming'" json:"status"` // upcoming, completed, cancelled
	PaymentStatus        string      `gorm:"type:varchar(50);default:'unpaid'" json:"payment_status"` // unpaid, dp, paid
	FormToken            string      `gorm:"type:varchar(50);uniqueIndex" json:"form_token"`
	FormStatus           string      `gorm:"type:varchar(50);default:'pending'" json:"form_status"` // pending, filled
	DpAmount             float64     `gorm:"type:decimal(15,2);default:0" json:"dp_amount"`
	Notes                string      `gorm:"type:text" json:"notes"`
	ClientForm           *ClientForm `gorm:"foreignKey:AgendaID" json:"client_form,omitempty"`
	CreatedAt            time.Time   `json:"created_at"`
	UpdatedAt            time.Time   `json:"updated_at"`
}

// CalculateTotalBiaya menghitung total biaya keseluruhan agenda (paket + model + addon + transport)
func (a *Agenda) CalculateTotalBiaya() float64 {
	total := a.TransportFee + a.ModelAdditionalPrice
	if a.ClientForm != nil && a.ClientForm.TotalPrice > 0 {
		total += a.ClientForm.TotalPrice
	} else {
		total += a.PackagePrice
	}
	return total
}

// CreateAgendaRequest adalah payload untuk membuat agenda baru
type CreateAgendaRequest struct {
	ClientName           string  `json:"client_name"`
	ClientPhone          string  `json:"client_phone"`
	BackdropTitle        string  `json:"backdrop_title"`
	EventDateTime        string  `json:"event_date_time"` // Format ISO8601 atau "2006-01-02 15:04"
	MapsURL              string  `json:"maps_url"`
	PackageID            string  `json:"package_id"`
	PackagePrice         float64 `json:"package_price"`
	ModelID              string  `json:"model_id"`
	ModelName            string  `json:"model_name"`
	ModelWidth           string  `json:"model_width"`
	ModelImageURL        string  `json:"model_image_url"`
	ModelAdditionalPrice float64 `json:"model_additional_price"`
	TransportFee         float64 `json:"transport_fee"`
	Notes                string  `json:"notes"`
}

// UpdateDpRequest adalah payload untuk input/update pembayaran DP
type UpdateDpRequest struct {
	DpAmount float64 `json:"dp_amount"`
}

// UpdateTransportRequest adalah payload untuk input/update uang transport
type UpdateTransportRequest struct {
	TransportFee float64 `json:"transport_fee"`
}

// UpdatePaymentStatusRequest adalah payload untuk mengubah status pembayaran
type UpdatePaymentStatusRequest struct {
	PaymentStatus string `json:"payment_status"` // unpaid, dp, paid
}

// UpdateEventStatusRequest adalah payload untuk mengubah status acara
type UpdateEventStatusRequest struct {
	Status string `json:"status"` // upcoming, completed, cancelled
}

// ReorderPackagesRequest adalah payload untuk update urutan paket
type ReorderPackagesRequest struct {
	PackageIDs []string `json:"package_ids"`
}

// UpdatePackageRequest adalah payload untuk mengubah paket dekorasi dan harga paket
type UpdatePackageRequest struct {
	PackageID            string  `json:"package_id"`
	PackageName          string  `json:"package_name"`
	PackagePrice         float64 `json:"package_price"`
	ModelID              string  `json:"model_id"`
	ModelName            string  `json:"model_name"`
	ModelWidth           string  `json:"model_width"`
	ModelImageURL        string  `json:"model_image_url"`
	ModelAdditionalPrice float64 `json:"model_additional_price"`
}

// UpdateModelRequest adalah payload untuk mengubah model dekorasi agenda
type UpdateModelRequest struct {
	ModelID              string  `json:"model_id"`
	ModelName            string  `json:"model_name"`
	ModelWidth           string  `json:"model_width"`
	ModelImageURL        string  `json:"model_image_url"`
	ModelAdditionalPrice float64 `json:"model_additional_price"`
}

// UpdateNotesRequest adalah payload untuk mengubah catatan / request tambahan agenda
type UpdateNotesRequest struct {
	Notes string `json:"notes"`
}





package models

import "time"

// Property mewakili master inventaris barang dekorasi atau alat kerja lapangan
type Property struct {
	ID          string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	CompanyID   string    `gorm:"type:varchar(50);index;default:'cmp_default'" json:"company_id"`
	Name        string    `gorm:"type:varchar(150);not null" json:"name"` // Contoh: Lampu 50 Watt, Tiang Besi 3m, Lem Tembak
	Category    string    `gorm:"type:varchar(100);default:'Lainnya'" json:"category"` // Rangka, Lampu, Bunga, Kain, Alat Kerja, dll.
	Type        string    `gorm:"type:varchar(50);default:'property'" json:"type"` // "property" (dekorasi) atau "tool" (alat kerja)
	TrackStock  bool      `gorm:"default:true" json:"track_stock"` // true = cek bentrok tanggal, false = bebas bawa (misal alat)
	TotalStock  int       `gorm:"type:int;not null;default:1" json:"total_stock"` // Total unit fisik di gudang
	Unit        string    `gorm:"type:varchar(50);default:'pcs'" json:"unit"` // pcs, unit, set, roll, box
	Notes       string    `gorm:"type:text" json:"notes"`
	ImageURL    string    `gorm:"type:text" json:"image_url"`
	SortOrder   int       `gorm:"type:int;default:0" json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DecorationModelProperty menghubungkan Model Dekorasi dengan Properti/Alat bawaan (Template)
type DecorationModelProperty struct {
	ID                string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	CompanyID         string    `gorm:"type:varchar(50);index;default:'cmp_default'" json:"company_id"`
	DecorationModelID string    `gorm:"type:varchar(50);index;not null" json:"decoration_model_id"`
	PropertyID        string    `gorm:"type:varchar(50);index;not null" json:"property_id"`
	DefaultQty        int       `gorm:"type:int;not null;default:1" json:"default_qty"`
	Notes             string    `gorm:"type:varchar(255)" json:"notes"`
	Property          *Property `gorm:"foreignKey:PropertyID" json:"property,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// AgendaProperty adalah Pick List aktual per agenda (Paperless Checklist)
type AgendaProperty struct {
	ID         string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	CompanyID  string    `gorm:"type:varchar(50);index;default:'cmp_default'" json:"company_id"`
	AgendaID   string    `gorm:"type:varchar(50);index;not null" json:"agenda_id"`
	PropertyID string    `gorm:"type:varchar(50);index;not null" json:"property_id"`
	CustomName string    `gorm:"type:varchar(150)" json:"custom_name"` // Jika ada nama kustom / barang mendadak
	Qty        int       `gorm:"type:int;not null;default:1" json:"qty"`
	Notes      string    `gorm:"type:text" json:"notes"`
	IsPacked   bool      `gorm:"default:false" json:"is_packed"`   // Dicentang saat muat ke mobil (Berangkat)
	IsReturned bool      `gorm:"default:false" json:"is_returned"` // Dicentang saat bongkaran kembali ke gudang (Pulang)
	Property   *Property `gorm:"foreignKey:PropertyID" json:"property,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// PropertyAvailabilityResponse mewakili stok dan ketersediaan di tanggal tertentu
type PropertyAvailabilityResponse struct {
	Property
	BookedQty      int  `json:"booked_qty"`
	AvailableQty   int  `json:"available_qty"`
	IsAvailable    bool `json:"is_available"` // false jika TrackStock=true dan AvailableQty <= 0
}

// CreateOrUpdatePropertyRequest payload request untuk master properti
type CreateOrUpdatePropertyRequest struct {
	Name       string `json:"name"`
	Category   string `json:"category"`
	Type       string `json:"type"`        // "property" atau "tool"
	TrackStock *bool  `json:"track_stock"` // default true jika type=property, false jika type=tool
	TotalStock int    `json:"total_stock"`
	Unit       string `json:"unit"`
	Notes      string `json:"notes"`
	ImageURL   string `json:"image_url"`
}

// ModelPropertyInput input properti saat setup model dekorasi
type ModelPropertyInput struct {
	PropertyID string `json:"property_id"`
	DefaultQty int    `json:"default_qty"`
	Notes      string `json:"notes"`
}

// AgendaPropertyInput input pick list barang untuk agenda
type AgendaPropertyInput struct {
	ID         string `json:"id,omitempty"`
	PropertyID string `json:"property_id"`
	CustomName string `json:"custom_name,omitempty"`
	Qty        int    `json:"qty"`
	Notes      string `json:"notes,omitempty"`
	IsPacked   bool   `json:"is_packed"`
	IsReturned bool   `json:"is_returned"`
}

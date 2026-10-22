package models

import "time"

// Company mewakili profil badan usaha / vendor dekorasi
type Company struct {
	ID        string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	Name      string    `gorm:"type:varchar(150);not null" json:"name"`
	Address   string    `gorm:"type:text" json:"address"`
	Phone     string    `gorm:"type:varchar(50)" json:"phone"`
	LogoURL   string    `gorm:"type:text" json:"logo_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relasi ke pengguna dalam perusahaan ini
	Users []User `gorm:"foreignKey:CompanyID" json:"users,omitempty"`
}

// CompanyDetailResponse untuk data detail perusahaan di Web Superadmin
type CompanyDetailResponse struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Address       string    `json:"address"`
	Phone         string    `json:"phone"`
	LogoURL       string    `json:"logo_url"`
	TotalOwners   int64     `json:"total_owners"`
	TotalAdmins   int64     `json:"total_admins"`
	TotalCrew     int64     `json:"total_crew"`
	TotalAgendas  int64     `json:"total_agendas"`
	OwnersList    []string  `json:"owners_list"` // Nama & no HP pemilik
	CreatedAt     time.Time `json:"created_at"`
}

// CreateCompanyRequest adalah payload saat Superadmin mendaftarkan usaha dan owner baru
type CreateCompanyRequest struct {
	CompanyName    string `json:"company_name"`
	CompanyAddress string `json:"company_address"`
	CompanyPhone   string `json:"company_phone"`
	OwnerName      string `json:"owner_name"`
	OwnerPhone     string `json:"owner_phone"`
	OwnerEmail     string `json:"owner_email"`
	OwnerPassword  string `json:"owner_password"`
}

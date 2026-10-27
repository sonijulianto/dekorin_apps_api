package models

import "time"

// Company mewakili profil badan usaha / vendor dekorasi
type Company struct {
	ID            string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	Name          string    `gorm:"type:varchar(150);not null" json:"name"`
	Address       string    `gorm:"type:text" json:"address"`
	Phone         string    `gorm:"type:varchar(50)" json:"phone"`
	Email         string    `gorm:"type:varchar(100)" json:"email"`
	LogoURL       string    `gorm:"type:text" json:"logo_url"`
	Description   string    `gorm:"type:text" json:"description"`
	Instagram     string    `gorm:"type:varchar(100)" json:"instagram"`
	BankInfo      string    `gorm:"type:text" json:"bank_info"`
	UpdatedBy     string    `gorm:"type:varchar(50)" json:"updated_by"`
	UpdatedByName string    `gorm:"type:varchar(100)" json:"updated_by_name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Relasi ke pengguna dalam perusahaan ini
	Users []User `gorm:"foreignKey:CompanyID" json:"users,omitempty"`
}

// CompanyDetailResponse untuk data detail perusahaan di Web Superadmin & Mobile App
type CompanyDetailResponse struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Address       string    `json:"address"`
	Phone         string    `json:"phone"`
	Email         string    `json:"email"`
	LogoURL       string    `json:"logo_url"`
	Description   string    `json:"description"`
	Instagram     string    `json:"instagram"`
	BankInfo      string    `json:"bank_info"`
	UpdatedBy     string    `json:"updated_by"`
	UpdatedByName string    `json:"updated_by_name"`
	TotalOwners   int64     `json:"total_owners"`
	TotalAdmins   int64     `json:"total_admins"`
	TotalCrew     int64     `json:"total_crew"`
	TotalAgendas  int64     `json:"total_agendas"`
	OwnersList    []string  `json:"owners_list"` // Nama & no HP pemilik
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// UpdateCompanyRequest payload untuk update data profil badan usaha
type UpdateCompanyRequest struct {
	Name        string `json:"name"`
	Address     string `json:"address"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
	LogoURL     string `json:"logo_url"`
	Description string `json:"description"`
	Instagram   string `json:"instagram"`
	BankInfo    string `json:"bank_info"`
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

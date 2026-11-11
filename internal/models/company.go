package models

import (
	"strings"
	"time"
)

// NormalizeVendorCategory menormalisasi kode kategori usaha vendor agar konsisten untuk multi-kategori.
func NormalizeVendorCategory(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "decoration", "dekorasi", "decor":
		return "decoration"
	case "wo", "wedding_organizer", "planner":
		return "wo"
	case "mua", "makeup", "rias":
		return "mua"
	case "photo_video", "photography", "foto", "dokumentasi":
		return "photo_video"
	case "catering", "katering":
		return "catering"
	case "attire", "busana", "gaun":
		return "attire"
	case "entertainment", "mc", "musik", "band":
		return "entertainment"
	case "venue", "gedung":
		return "venue"
	default:
		return "other"
	}
}

// Company mewakili profil badan usaha / vendor pernikahan (dekorasi maupun kategori vendor lain)
type Company struct {
	ID              string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	Name            string    `gorm:"type:varchar(150);not null" json:"name"`
	Category        string    `gorm:"type:varchar(40);default:'decoration';index" json:"category"`
	City            string    `gorm:"type:varchar(100);default:'';index" json:"city"`
	IsPublicListed  bool      `gorm:"default:false;index" json:"is_public_listed"`
	Address         string    `gorm:"type:text" json:"address"`
	Phone           string    `gorm:"type:varchar(50)" json:"phone"`
	Email           string    `gorm:"type:varchar(100)" json:"email"`
	LogoURL         string    `gorm:"type:text" json:"logo_url"`
	Description     string    `gorm:"type:text" json:"description"`
	Instagram       string    `gorm:"type:varchar(100)" json:"instagram"`
	BankInfo        string    `gorm:"type:text" json:"bank_info"`
	InvoiceTemplate string    `gorm:"type:varchar(60);default:'template_modern_navy'" json:"invoice_template"`
	UpdatedBy       string    `gorm:"type:varchar(50)" json:"updated_by"`
	UpdatedByName   string    `gorm:"type:varchar(100)" json:"updated_by_name"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relasi ke pengguna dalam perusahaan ini
	Users []User `gorm:"foreignKey:CompanyID" json:"users,omitempty"`
}

// CompanyDetailResponse untuk data detail perusahaan di Web Superadmin & Mobile App
type CompanyDetailResponse struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Category        string    `json:"category"`
	City            string    `json:"city"`
	IsPublicListed  bool      `json:"is_public_listed"`
	Address         string    `json:"address"`
	Phone           string    `json:"phone"`
	Email           string    `json:"email"`
	LogoURL         string    `json:"logo_url"`
	Description     string    `json:"description"`
	Instagram       string    `json:"instagram"`
	BankInfo        string    `json:"bank_info"`
	InvoiceTemplate string    `json:"invoice_template"`
	UpdatedBy       string    `json:"updated_by"`
	UpdatedByName   string    `json:"updated_by_name"`
	TotalOwners     int64     `json:"total_owners"`
	TotalAdmins     int64     `json:"total_admins"`
	TotalCrew       int64     `json:"total_crew"`
	TotalAgendas    int64     `json:"total_agendas"`
	OwnersList      []string  `json:"owners_list"` // Nama & no HP pemilik
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// UpdateCompanyRequest payload untuk update data profil badan usaha
type UpdateCompanyRequest struct {
	Name            string `json:"name"`
	Category        string `json:"category"`
	City            string `json:"city"`
	IsPublicListed  *bool  `json:"is_public_listed"`
	Address         string `json:"address"`
	Phone           string `json:"phone"`
	Email           string `json:"email"`
	LogoURL         string `json:"logo_url"`
	Description     string `json:"description"`
	Instagram       string `json:"instagram"`
	BankInfo        string `json:"bank_info"`
	InvoiceTemplate string `json:"invoice_template"`
}

// PublicVendorSummary merepresentasikan ringkasan profil vendor di Direktori Pencarian Calon Pengantin
type PublicVendorSummary struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	City         string   `json:"city"`
	Address      string   `json:"address"`
	Phone        string   `json:"phone"`
	Instagram    string   `json:"instagram"`
	LogoURL      string   `json:"logo_url"`
	Description  string   `json:"description"`
	PackageCount int      `json:"package_count"`
	ModelCount   int      `json:"model_count"`
	MinPrice     float64  `json:"min_price"`
	SampleImages []string `json:"sample_images"`
}

// PublicVendorPackageItem ringkasan paket yang aman ditampilkan di direktori publik
type PublicVendorPackageItem struct {
	ID          string  `json:"id"`
	BrandName   string  `json:"brand_name"`
	Name        string  `json:"name"`
	ImageURL    string  `json:"image_url"`
	Description string  `json:"description"`
	BasePrice   float64 `json:"base_price"`
}

// PublicVendorModelItem ringkasan model/portofolio yang aman ditampilkan di direktori publik
type PublicVendorModelItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	ImageURL    string  `json:"image_url"`
	Width       string  `json:"width"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

// PublicVendorDetailResponse profil lengkap vendor publik beserta katalog paket & portofolio
type PublicVendorDetailResponse struct {
	PublicVendorSummary
	Email    string                    `json:"email"`
	Packages []PublicVendorPackageItem `json:"packages"`
	Models   []PublicVendorModelItem   `json:"models"`
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

// CompanyBrand mewakili sub-brand di bawah satu badan usaha (Company)
// Alamat dan Nomor Rekening tetap mengikuti Company induk, sedangkan Nama, Logo, WA, IG, dan Template Invoice dapat berbeda per brand.
type CompanyBrand struct {
	ID              string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	CompanyID       string    `gorm:"type:varchar(50);index;not null" json:"company_id"`
	Name            string    `gorm:"type:varchar(150);not null" json:"name"`
	LogoURL         string    `gorm:"type:text" json:"logo_url"`
	Description     string    `gorm:"type:text" json:"description"`
	Phone           string    `gorm:"type:varchar(50)" json:"phone"`
	Instagram       string    `gorm:"type:varchar(100)" json:"instagram"`
	InvoiceTemplate string    `gorm:"type:varchar(60);default:''" json:"invoice_template"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateOrUpdateBrandRequest adalah payload untuk membuat atau mengubah sub-brand
type CreateOrUpdateBrandRequest struct {
	Name            string `json:"name"`
	LogoURL         string `json:"logo_url"`
	Description     string `json:"description"`
	Phone           string `json:"phone"`
	Instagram       string `json:"instagram"`
	InvoiceTemplate string `json:"invoice_template"`
}

// UpdateInvoiceTemplateRequest adalah payload untuk mengubah pilihan template desain invoice per brand
type UpdateInvoiceTemplateRequest struct {
	BrandID         string `json:"brand_id"`
	InvoiceTemplate string `json:"invoice_template"`
}

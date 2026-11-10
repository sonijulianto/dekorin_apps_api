package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// FormFieldDefinition mendefinisikan satu baris isian pada Form Link Client yang dapat diatur tiap vendor/sub-brand
type FormFieldDefinition struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Type        string   `json:"type"` // text, textarea, number, date, time, datetime, select, radio, checkbox, image, maps, package_picker, model_picker, addon_picker
	Placeholder string   `json:"placeholder"`
	Options     []string `json:"options,omitempty"`
	IsSystem    bool     `json:"is_system"`
	IsActive    bool     `json:"is_active"`
	IsRequired  bool     `json:"is_required"`
	SortOrder   int      `json:"sort_order"`
}

// CompanyFormConfig menyimpan konfigurasi daftar isian form client per Perusahaan dan Sub-Brand
type CompanyFormConfig struct {
	ID         string                `gorm:"primaryKey;type:varchar(50)" json:"id"`
	CompanyID  string                `gorm:"type:varchar(50);index;not null" json:"company_id"`
	BrandID    string                `gorm:"type:varchar(50);index;default:''" json:"brand_id"`
	ShowPrice  bool                  `gorm:"default:true" json:"show_price"`
	FieldsJSON string                `gorm:"type:text;not null" json:"-"`
	Fields     []FormFieldDefinition `gorm:"-" json:"fields"`
	CreatedAt  time.Time             `json:"created_at"`
	UpdatedAt  time.Time             `json:"updated_at"`
}

// AfterFind otomatis mengurai FieldsJSON menjadi slice Fields
func (c *CompanyFormConfig) AfterFind(tx *gorm.DB) error {
	if c.FieldsJSON != "" {
		_ = json.Unmarshal([]byte(c.FieldsJSON), &c.Fields)
	}
	if c.Fields == nil {
		c.Fields = []FormFieldDefinition{}
	}
	return nil
}

// UpdateCompanyFormConfigRequest payload untuk menyimpan pengaturan form client
type UpdateCompanyFormConfigRequest struct {
	BrandID   string                `json:"brand_id"`
	ShowPrice *bool                 `json:"show_price"`
	Fields    []FormFieldDefinition `json:"fields"`
}

// CustomFormAnswer menyimpan satu jawaban field dari client beserta snapshot label & tipenya
type CustomFormAnswer struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// SelectedAddonInput merepresentasikan addon dan kuantitas yang dipilih client di Web Form
type SelectedAddonInput struct {
	AddonID  string `json:"addon_id"`
	Quantity int    `json:"quantity"`
}

// ClientForm menyimpan data isian detail dari client via Web Form
type ClientForm struct {
	ID                string             `gorm:"primaryKey;type:varchar(50)" json:"id"`
	AgendaID          string             `gorm:"type:varchar(50);index;not null" json:"agenda_id"`
	ClientName        string             `gorm:"type:varchar(150);default:''" json:"client_name"`
	ClientPhone       string             `gorm:"type:varchar(50);default:''" json:"client_phone"`
	BackdropTitle     string             `gorm:"type:varchar(150);default:''" json:"backdrop_title"`
	EventDateTime     string             `gorm:"type:varchar(100);default:''" json:"event_date_time"`
	MapsURL           string             `gorm:"type:text;default:''" json:"maps_url"`
	PackageID         string             `gorm:"type:varchar(50)" json:"package_id"`
	PackageName       string             `gorm:"type:varchar(150)" json:"package_name"`
	PackagePrice      float64            `gorm:"type:decimal(15,2);default:0" json:"package_price"`
	ModelID           string             `gorm:"type:varchar(50);default:''" json:"model_id"`
	ModelName         string             `gorm:"type:varchar(150);default:''" json:"model_name"`
	SelectedAddons    string             `gorm:"type:text" json:"selected_addons"` // Comma-separated atau JSON nama/ID addon
	AddonsPrice       float64            `gorm:"type:decimal(15,2);default:0" json:"addons_price"`
	TotalPrice        float64            `gorm:"type:decimal(15,2);default:0" json:"total_price"`
	EventAddress      string             `gorm:"type:text;default:''" json:"event_address"`
	DecorationTheme   string             `gorm:"type:varchar(100)" json:"decoration_theme"`
	ColorPreference   string             `gorm:"type:varchar(100)" json:"color_preference"`
	SpecialRequests   string             `gorm:"type:text" json:"special_requests"`
	ReferencePhotoURL string             `gorm:"type:text" json:"reference_photo_url"`
	CustomAnswersJSON string             `gorm:"type:text;default:'[]'" json:"-"`
	CustomAnswers     []CustomFormAnswer `gorm:"-" json:"custom_answers"`
	SubmittedAt       time.Time          `json:"submitted_at"`
}

// AfterFind otomatis mengurai CustomAnswersJSON menjadi slice CustomAnswers
func (cf *ClientForm) AfterFind(tx *gorm.DB) error {
	if cf.CustomAnswersJSON != "" {
		_ = json.Unmarshal([]byte(cf.CustomAnswersJSON), &cf.CustomAnswers)
	}
	if cf.CustomAnswers == nil {
		cf.CustomAnswers = []CustomFormAnswer{}
	}
	return nil
}

// SubmitClientFormRequest digunakan saat client mengisi form via Web Form
type SubmitClientFormRequest struct {
	ClientName        string               `json:"client_name"`
	ClientPhone       string               `json:"client_phone"`
	BackdropTitle     string               `json:"backdrop_title"`
	EventDateTime     string               `json:"event_date_time"`
	MapsURL           string               `json:"maps_url"`
	PackageID         string               `json:"package_id"`
	ModelID           string               `json:"model_id"`
	AddonIDs          []string             `json:"addon_ids"`
	AddonItems        []SelectedAddonInput `json:"addon_items"`
	EventAddress      string               `json:"event_address"`
	DecorationTheme   string               `json:"decoration_theme"`
	ColorPreference   string               `json:"color_preference"`
	SpecialRequests   string               `json:"special_requests"`
	Notes             string               `json:"notes"`
	ReferencePhotoURL string               `json:"reference_photo_url"`
	CustomAnswers     map[string]string    `json:"custom_answers"`
}

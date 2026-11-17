package models

import "time"

// UserDeviceToken menyimpan token FCM perangkat user untuk push notification (Android, iOS, Web/PWA).
type UserDeviceToken struct {
	ID        string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	UserID    string    `gorm:"type:varchar(50);index;not null" json:"user_id"`
	CompanyID string    `gorm:"type:varchar(50);index;default:''" json:"company_id"`
	Platform  string    `gorm:"type:varchar(20);default:'android'" json:"platform"` // android, web, ios
	Token     string    `gorm:"type:varchar(500);uniqueIndex;not null" json:"token"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AgendaCollaboration merepresentasikan undangan & relasi kolaborasi antar-vendor pada sebuah agenda.
// Lead vendor menentukan kategori, memilih vendor partner, serta menetapkan harga layanan & transport partner.
type AgendaCollaboration struct {
	ID                  string     `gorm:"primaryKey;type:varchar(50)" json:"id"`
	LeadAgendaID        string     `gorm:"type:varchar(50);index;not null" json:"lead_agenda_id"`
	LeadCompanyID       string     `gorm:"type:varchar(50);index;not null" json:"lead_company_id"`
	LeadCompanyName     string     `gorm:"type:varchar(150);default:''" json:"lead_company_name"`
	PartnerCompanyID    string     `gorm:"type:varchar(50);index;not null" json:"partner_company_id"`
	PartnerCompanyName  string     `gorm:"type:varchar(150);default:''" json:"partner_company_name"`
	PartnerCategory     string     `gorm:"type:varchar(40);index;default:'other'" json:"partner_category"`
	PartnerAgendaID     string     `gorm:"type:varchar(50);index;default:''" json:"partner_agenda_id"`
	PartnerPrice        float64    `gorm:"type:decimal(15,2);default:0" json:"partner_price"`
	PartnerTransportFee float64    `gorm:"type:decimal(15,2);default:0" json:"partner_transport_fee"`
	Notes               string     `gorm:"type:text;default:''" json:"notes"`
	Status              string     `gorm:"type:varchar(30);index;default:'pending'" json:"status"` // pending, accepted, rejected, cancelled
	RejectReason        string     `gorm:"type:text;default:''" json:"reject_reason"`
	CreatedByUserID     string     `gorm:"type:varchar(50);default:''" json:"created_by_user_id"`
	RespondedByUserID   string     `gorm:"type:varchar(50);default:''" json:"responded_by_user_id"`
	RespondedAt         *time.Time `json:"responded_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// TotalCost mengembalikan total biaya partner (harga layanan + transport).
func (c *AgendaCollaboration) TotalCost() float64 {
	return c.PartnerPrice + c.PartnerTransportFee
}

// AppNotification menyimpan riwayat notifikasi in-app untuk user.
type AppNotification struct {
	ID          string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	UserID      string    `gorm:"type:varchar(50);index;not null" json:"user_id"`
	CompanyID   string    `gorm:"type:varchar(50);index;default:''" json:"company_id"`
	Type        string    `gorm:"type:varchar(50);index;not null" json:"type"` // collaboration_invite, collaboration_accepted, collaboration_rejected, collaboration_updated
	Title       string    `gorm:"type:varchar(200);not null" json:"title"`
	Body        string    `gorm:"type:text;default:''" json:"body"`
	ReferenceID string    `gorm:"type:varchar(50);index;default:''" json:"reference_id"`
	IsRead      bool      `gorm:"default:false;index" json:"is_read"`
	CreatedAt   time.Time `json:"created_at"`
}

// RegisterDeviceTokenRequest payload registrasi token FCM perangkat.
type RegisterDeviceTokenRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

// CreateCollaborationRequest payload saat Lead Vendor mengundang vendor partner ke agenda.
type CreateCollaborationRequest struct {
	PartnerCompanyID    string  `json:"partner_company_id"`
	PartnerPrice        float64 `json:"partner_price"`
	PartnerTransportFee float64 `json:"partner_transport_fee"`
	Notes               string  `json:"notes"`
}

// RespondCollaborationRequest payload saat Partner menerima atau menolak undangan kolaborasi.
type RespondCollaborationRequest struct {
	Action       string `json:"action"` // accept | reject
	RejectReason string `json:"reject_reason"`
}

// CollaborationVendorOption adalah item vendor terdaftar untuk dipilih saat mengundang partner.
type CollaborationVendorOption struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	City     string `json:"city"`
	Phone    string `json:"phone"`
	LogoURL  string `json:"logo_url"`
}

// IncomingCollaborationItem merepresentasikan undangan kolaborasi masuk untuk vendor partner.
type IncomingCollaborationItem struct {
	AgendaCollaboration
	ClientName          string `json:"client_name"`
	ClientPhone         string `json:"client_phone"`
	EventType           string `json:"event_type"`
	EventLabel          string `json:"event_label"`
	EventDateTime       string `json:"event_date_time"`
	EventAddress        string `json:"event_address"`
	MapsURL             string `json:"maps_url"`
	HasScheduleConflict bool   `json:"has_schedule_conflict"`
}

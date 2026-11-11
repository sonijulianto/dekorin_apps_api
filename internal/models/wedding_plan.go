package models

import "time"

// WeddingPlan mewakili ruang rencana pernikahan milik akun Calon Pengantin (dapat diakses berdua oleh pasangan).
type WeddingPlan struct {
	ID              string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	Title           string    `gorm:"type:varchar(200);not null" json:"title"`
	GroomName       string    `gorm:"type:varchar(150);default:''" json:"groom_name"`
	BrideName       string    `gorm:"type:varchar(150);default:''" json:"bride_name"`
	WeddingDate     string    `gorm:"type:varchar(30);default:''" json:"wedding_date"` // Format YYYY-MM-DD
	AkadTime        string    `gorm:"type:varchar(60);default:''" json:"akad_time"`
	ResepsiTime     string    `gorm:"type:varchar(60);default:''" json:"resepsi_time"`
	City            string    `gorm:"type:varchar(100);default:''" json:"city"`
	VenueName       string    `gorm:"type:varchar(150);default:''" json:"venue_name"`
	VenueAddress    string    `gorm:"type:text;default:''" json:"venue_address"`
	MapsURL         string    `gorm:"type:text;default:''" json:"maps_url"`
	BudgetTarget    float64   `gorm:"type:decimal(15,2);default:0" json:"budget_target"`
	JoinCode        string    `gorm:"type:varchar(20);uniqueIndex;not null" json:"join_code"`
	Notes           string    `gorm:"type:text;default:''" json:"notes"`
	CreatedByUserID string    `gorm:"type:varchar(50);index;default:''" json:"created_by_user_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Members []WeddingPlanMember `gorm:"foreignKey:WeddingPlanID" json:"members,omitempty"`
}

// WeddingPlanMember menghubungkan user (calon pengantin / pasangan) ke sebuah WeddingPlan.
type WeddingPlanMember struct {
	ID            string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	WeddingPlanID string    `gorm:"type:varchar(50);index;not null" json:"wedding_plan_id"`
	UserID        string    `gorm:"type:varchar(50);index;not null" json:"user_id"`
	MemberRole    string    `gorm:"type:varchar(30);default:'partner'" json:"member_role"` // 'creator', 'partner'
	CreatedAt     time.Time `json:"created_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// WeddingPlanVendor mewakili satu vendor di rencana pernikahan pengantin,
// baik yang tersinkron langsung dari Agenda Dekorin maupun vendor eksternal yang dicatat manual.
type WeddingPlanVendor struct {
	ID              string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	WeddingPlanID   string    `gorm:"type:varchar(50);index;not null" json:"wedding_plan_id"`
	VendorCategory  string    `gorm:"type:varchar(40);default:'other';index" json:"vendor_category"`
	VendorName      string    `gorm:"type:varchar(150);not null" json:"vendor_name"`
	VendorPhone     string    `gorm:"type:varchar(50);default:''" json:"vendor_phone"`
	VendorInstagram string    `gorm:"type:varchar(100);default:''" json:"vendor_instagram"`
	CompanyID       string    `gorm:"type:varchar(50);index;default:''" json:"company_id"`
	AgendaID        string    `gorm:"type:varchar(50);index;default:''" json:"agenda_id"`
	PackageName     string    `gorm:"type:varchar(150);default:''" json:"package_name"`
	AgreedPrice     float64   `gorm:"type:decimal(15,2);default:0" json:"agreed_price"`
	DpPaid          float64   `gorm:"type:decimal(15,2);default:0" json:"dp_paid"`
	PaymentStatus   string    `gorm:"type:varchar(30);default:'unpaid'" json:"payment_status"` // 'unpaid', 'dp', 'paid', 'inquiry'
	DueDate         string    `gorm:"type:varchar(30);default:''" json:"due_date"`
	Notes           string    `gorm:"type:text;default:''" json:"notes"`
	SourceType      string    `gorm:"type:varchar(30);default:'manual'" json:"source_type"` // 'manual', 'dekorin_agenda', 'dekorin_directory'
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// WeddingVendorViewResponse adalah format API yang diperkaya dengan data live Agenda & Company (jika tertaut).
type WeddingVendorViewResponse struct {
	WeddingPlanVendor
	IsSyncedAgenda   bool    `json:"is_synced_agenda"`
	CompanyLogo      string  `json:"company_logo,omitempty"`
	AgendaStatus     string  `json:"agenda_status,omitempty"`
	FormToken        string  `json:"form_token,omitempty"`
	FormStatus       string  `json:"form_status,omitempty"`
	ModelName        string  `json:"model_name,omitempty"`
	SubtotalDecor    float64 `json:"subtotal_decor,omitempty"`
	DiscountAmount   float64 `json:"discount_amount,omitempty"`
	TransportFee     float64 `json:"transport_fee,omitempty"`
	RemainingBalance float64 `json:"remaining_balance"`
	EventDateTime    string  `json:"event_date_time,omitempty"`
}

// WeddingTask mewakili item checklist persiapan pernikahan pada WeddingPlan.
type WeddingTask struct {
	ID            string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	WeddingPlanID string    `gorm:"type:varchar(50);index;not null" json:"wedding_plan_id"`
	Title         string    `gorm:"type:varchar(200);not null" json:"title"`
	Category      string    `gorm:"type:varchar(60);default:'Lainnya'" json:"category"`
	Phase         string    `gorm:"type:varchar(40);default:'Persiapan'" json:"phase"` // 'H-6 Bulan', 'H-3 Bulan', 'H-1 Bulan', 'H-7', 'Hari-H'
	DueDate       string    `gorm:"type:varchar(30);default:''" json:"due_date"`
	IsCompleted   bool      `gorm:"default:false;index" json:"is_completed"`
	Notes         string    `gorm:"type:text;default:''" json:"notes"`
	SortOrder     int       `gorm:"type:int;default:0" json:"sort_order"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// WeddingPlanMemberSummary merangkum data anggota/pasangan yang tergabung di WeddingPlan.
type WeddingPlanMemberSummary struct {
	UserID     string `json:"user_id"`
	FullName   string `json:"full_name"`
	Phone      string `json:"phone"`
	MemberRole string `json:"member_role"`
}

// WeddingPlanSummaryStats merangkum statistik progres persiapan, budget, dan undangan untuk dashboard pengantin.
type WeddingPlanSummaryStats struct {
	DaysRemaining          *int    `json:"days_remaining"`
	TotalTasks             int     `json:"total_tasks"`
	CompletedTasks         int     `json:"completed_tasks"`
	TotalVendors           int     `json:"total_vendors"`
	SyncedVendors          int     `json:"synced_vendors"`
	TotalAgreedCost        float64 `json:"total_agreed_cost"`
	TotalDpPaid            float64 `json:"total_dp_paid"`
	RemainingVendorBalance float64 `json:"remaining_vendor_balance"`
	LinkedInvitationsCount int     `json:"linked_invitations_count"`
	TotalRsvpHadir         int     `json:"total_rsvp_hadir"`
	TotalPaxHadir          int     `json:"total_pax_hadir"`
}

// WeddingPlanDetailResponse adalah response utama endpoint GET /api/couple/plan.
type WeddingPlanDetailResponse struct {
	Plan    WeddingPlan                `json:"plan"`
	Members []WeddingPlanMemberSummary `json:"members"`
	Stats   WeddingPlanSummaryStats    `json:"stats"`
}

// UpdateWeddingPlanRequest payload untuk memperbarui data inti pernikahan.
type UpdateWeddingPlanRequest struct {
	Title        string  `json:"title"`
	GroomName    string  `json:"groom_name"`
	BrideName    string  `json:"bride_name"`
	WeddingDate  string  `json:"wedding_date"`
	AkadTime     string  `json:"akad_time"`
	ResepsiTime  string  `json:"resepsi_time"`
	City         string  `json:"city"`
	VenueName    string  `json:"venue_name"`
	VenueAddress string  `json:"venue_address"`
	MapsURL      string  `json:"maps_url"`
	BudgetTarget float64 `json:"budget_target"`
	Notes        string  `json:"notes"`
}

// JoinWeddingPlanRequest payload untuk bergabung ke WeddingPlan pasangan memakai JoinCode.
type JoinWeddingPlanRequest struct {
	JoinCode string `json:"join_code"`
}

// CreateOrUpdateWeddingVendorRequest payload untuk mencatat / mengubah vendor manual di WeddingPlan.
type CreateOrUpdateWeddingVendorRequest struct {
	VendorCategory  string  `json:"vendor_category"`
	VendorName      string  `json:"vendor_name"`
	VendorPhone     string  `json:"vendor_phone"`
	VendorInstagram string  `json:"vendor_instagram"`
	PackageName     string  `json:"package_name"`
	AgreedPrice     float64 `json:"agreed_price"`
	DpPaid          float64 `json:"dp_paid"`
	PaymentStatus   string  `json:"payment_status"`
	DueDate         string  `json:"due_date"`
	Notes           string  `json:"notes"`
}

// LinkDekorinAgendaRequest payload untuk menautkan agenda vendor Dekorin ke WeddingPlan via FormToken / link form.
type LinkDekorinAgendaRequest struct {
	TokenOrUrl string `json:"token_or_url"`
}

// AddPublicVendorRequest payload untuk menambahkan vendor dari Direktori Publik Dekorin ke WeddingPlan.
type AddPublicVendorRequest struct {
	CompanyID   string  `json:"company_id"`
	PackageName string  `json:"package_name"`
	AgreedPrice float64 `json:"agreed_price"`
	Notes       string  `json:"notes"`
}

// CreateOrUpdateWeddingTaskRequest payload untuk membuat / memperbarui item checklist pengantin.
type CreateOrUpdateWeddingTaskRequest struct {
	Title       string `json:"title"`
	Category    string `json:"category"`
	Phase       string `json:"phase"`
	DueDate     string `json:"due_date"`
	IsCompleted *bool  `json:"is_completed"`
	Notes       string `json:"notes"`
}

// ClaimInvitationRequest payload untuk mengklaim undangan digital yang dibuatkan vendor ke WeddingPlan.
type ClaimInvitationRequest struct {
	Slug          string `json:"slug"`
	Pin           string `json:"pin"`
	EditTokenOrUrl string `json:"edit_token_or_url"`
}

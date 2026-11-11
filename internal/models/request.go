package models

type LoginRequest struct {
	Phone    string `json:"phone"`
	Email    string `json:"email"` // opsional fallback jika user memasukkan email
	Password string `json:"password"`
}

type CreateUserRequest struct {
	CompanyID string `json:"company_id"` // Opsional jika dibuat oleh Superadmin
	FullName  string `json:"full_name"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Role      string `json:"role"` // 'owner', 'admin' atau 'crew'
}

type UpdateUserRequest struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Password string `json:"password"` // opsional jika ingin mengubah password
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// RegisterCoupleRequest adalah payload pendaftaran mandiri untuk akun Calon Pengantin
type RegisterCoupleRequest struct {
	FullName     string  `json:"full_name"`
	Phone        string  `json:"phone"`
	Email        string  `json:"email"`
	Password     string  `json:"password"`
	PartnerName  string  `json:"partner_name"`
	WeddingDate  string  `json:"wedding_date"`  // format YYYY-MM-DD (opsional)
	City         string  `json:"city"`          // opsional
	VenueName    string  `json:"venue_name"`    // opsional
	BudgetTarget float64 `json:"budget_target"` // opsional
	JoinCode     string  `json:"join_code"`     // opsional: jika bergabung ke WeddingPlan pasangan yang sudah ada
}


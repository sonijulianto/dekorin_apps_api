package models

import "time"

// CompanyRolePermission menyimpan daftar hak akses khusus per role di suatu company
type CompanyRolePermission struct {
	ID          string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	CompanyID   string    `gorm:"type:varchar(50);index;not null" json:"company_id"`
	Role        string    `gorm:"type:varchar(20);not null" json:"role"` // 'admin', 'crew'
	Permissions string    `gorm:"type:text;not null;default:'[]'" json:"permissions"` // JSON array string
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PermissionItem mendefinisikan informasi sebuah hak akses / fitur
type PermissionItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// PermissionCategory mengelompokkan permission berdasarkan modul
type PermissionCategory struct {
	Category    string           `json:"category"`
	Permissions []PermissionItem `json:"permissions"`
}

// RolePermissionsResponse format balikan API untuk matriks hak akses
type RolePermissionsResponse struct {
	Definitions     []PermissionCategory `json:"definitions"`
	RolePermissions map[string][]string  `json:"role_permissions"` // map role -> list of permission IDs
}

// UpdateRolePermissionRequest format body saat Owner mengubah hak akses role
type UpdateRolePermissionRequest struct {
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

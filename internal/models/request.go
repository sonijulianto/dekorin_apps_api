package models

type LoginRequest struct {
	Phone    string `json:"phone"`
	Email    string `json:"email"` // opsional fallback jika user memasukkan email
	Password string `json:"password"`
}

type CreateUserRequest struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"` // 'admin' atau 'crew'
}

type UpdateUserRequest struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Password string `json:"password"` // opsional jika ingin mengubah password
}

package models

import "time"

// Expense mewakili pencatatan pengeluaran operasional dekorasi
type Expense struct {
	ID        string    `gorm:"primaryKey;type:varchar(50)" json:"id"`
	Title     string    `gorm:"type:varchar(200);not null" json:"title"`
	Amount    float64   `gorm:"type:decimal(15,2);not null;default:0" json:"amount"`
	Category  string    `gorm:"type:varchar(100);default:'Lainnya'" json:"category"`
	Date      time.Time `gorm:"not null" json:"date"`
	Notes     string    `gorm:"type:text" json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateExpenseRequest payload untuk membuat pengeluaran baru
type CreateExpenseRequest struct {
	Title    string  `json:"title"`
	Amount   float64 `json:"amount"`
	Category string  `json:"category"`
	Date     string  `json:"date"` // Format "2006-01-02" atau ISO8601
	Notes    string  `json:"notes"`
}

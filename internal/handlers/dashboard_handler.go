package handlers

import (
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
)

// DashboardSummaryResponse struct untuk response /api/dashboard/summary
type DashboardSummaryResponse struct {
	TotalRevenue          float64          `json:"total_revenue"`
	TotalExpense          float64          `json:"total_expense"`
	TotalProfit           float64          `json:"total_profit"`
	UpcomingBookingsCount int64            `json:"upcoming_bookings_count"`
	UpcomingAgendas       []models.Agenda  `json:"upcoming_agendas"`
	RecentExpenses        []models.Expense `json:"recent_expenses"`
}

// GetDashboardSummary handles GET /api/dashboard/summary
func GetDashboardSummary(c *fiber.Ctx) error {
	// 1. Total Pendapatan: Diambil dari semua agenda yang payment_status = 'paid'
	var paidAgendas []models.Agenda
	if err := config.DB.Preload("ClientForm").Where("payment_status = ?", "paid").Find(&paidAgendas).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menghitung pendapatan: " + err.Error(),
		})
	}

	var totalRevenue float64 = 0
	for _, a := range paidAgendas {
		totalRevenue += a.CalculateTotalBiaya()
	}

	// 2. Total Pengeluaran: SUM(amount) dari tabel expenses
	var totalExpense float64 = 0
	row := config.DB.Model(&models.Expense{}).Select("COALESCE(SUM(amount), 0)").Row()
	_ = row.Scan(&totalExpense)

	// 3. Keuntungan = Pendapatan - Pengeluaran
	totalProfit := totalRevenue - totalExpense

	// 4. Total Booking Mendatang: Agenda dengan event_date_time >= awal hari ini (00:00:00) dan belum dibatalkan
	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var upcomingCount int64 = 0
	config.DB.Model(&models.Agenda{}).
		Where("event_date_time >= ? AND status != ?", startOfToday, "cancelled").
		Count(&upcomingCount)

	// Ambil daftar booking mendatang terdekat (maks 10)
	var upcomingAgendas []models.Agenda
	config.DB.Preload("ClientForm").
		Where("event_date_time >= ? AND status != ?", startOfToday, "cancelled").
		Order("event_date_time ASC").
		Limit(10).
		Find(&upcomingAgendas)

	// 5. Ambil 5 pengeluaran terbaru
	var recentExpenses []models.Expense
	config.DB.Order("date DESC, created_at DESC").Limit(5).Find(&recentExpenses)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil ringkasan dashboard",
		"data": DashboardSummaryResponse{
			TotalRevenue:          totalRevenue,
			TotalExpense:          totalExpense,
			TotalProfit:           totalProfit,
			UpcomingBookingsCount: upcomingCount,
			UpcomingAgendas:       upcomingAgendas,
			RecentExpenses:        recentExpenses,
		},
	})
}

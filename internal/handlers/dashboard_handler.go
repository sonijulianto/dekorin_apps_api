package handlers

import (
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/middleware"
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
	PeriodFilter          string           `json:"period_filter,omitempty"`
	Month                 int              `json:"month,omitempty"`
	Year                  int              `json:"year,omitempty"`
}

// GetDashboardSummary handles GET /api/dashboard/summary
// Mendukung filter periode per bulan (?filter=month&month=11&year=2026),
// per tahun (?filter=year&year=2026), dan semua (?filter=all).
// Otomatis terisolasi hanya untuk company_id user yang sedang login,
// atau filter query company_id jika diakses oleh superadmin.
func GetDashboardSummary(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	userRole, _ := c.Locals("user_role").(string)

	if strings.EqualFold(userRole, "superadmin") {
		if qComp := strings.TrimSpace(c.Query("company_id")); qComp != "" && qComp != "all" {
			companyID = qComp
		} else {
			companyID = ""
		}
	}

	period := strings.ToLower(strings.TrimSpace(c.Query("filter")))
	if period == "" {
		period = strings.ToLower(strings.TrimSpace(c.Query("period")))
	}

	reqMonth := c.QueryInt("month", 0)
	reqYear := c.QueryInt("year", 0)
	now := time.Now()

	var startDate, endDate time.Time
	var isFiltered bool

	if period == "month" || (period == "" && (reqMonth > 0 || reqYear > 0)) {
		month := reqMonth
		if month < 1 || month > 12 {
			month = int(now.Month())
		}
		year := reqYear
		if year <= 0 {
			year = now.Year()
		}
		startDate = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)
		isFiltered = true
		period = "month"
		reqMonth = month
		reqYear = year
	} else if period == "year" {
		year := reqYear
		if year <= 0 {
			year = now.Year()
		}
		startDate = time.Date(year, 1, 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(1, 0, 0).Add(-time.Nanosecond)
		isFiltered = true
		reqYear = year
	} else if period == "all" {
		isFiltered = false
	} else if startStr := strings.TrimSpace(c.Query("start_date")); startStr != "" {
		if tStart, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = tStart
			if endStr := strings.TrimSpace(c.Query("end_date")); endStr != "" {
				if tEnd, err := time.Parse("2006-01-02", endStr); err == nil {
					endDate = tEnd.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
				}
			}
			isFiltered = true
		}
	}

	// 1. Total Pendapatan: Diambil dari agenda perusahaan yang payment_status = 'paid'
	agendaQuery := config.DB.Preload("ClientForm").Where("payment_status = ?", "paid")
	if companyID != "" {
		agendaQuery = agendaQuery.Where("company_id = ?", companyID)
	}
	if isFiltered && !startDate.IsZero() && !endDate.IsZero() {
		agendaQuery = agendaQuery.Where("event_date_time >= ? AND event_date_time <= ?", startDate, endDate)
	}

	var paidAgendas []models.Agenda
	if err := agendaQuery.Find(&paidAgendas).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menghitung pendapatan: " + err.Error(),
		})
	}

	var totalRevenue float64 = 0
	for _, a := range paidAgendas {
		totalRevenue += a.CalculateTotalBiaya()
	}

	// 2. Total Pengeluaran: SUM(amount) dari tabel expenses perusahaan
	var totalExpense float64 = 0
	expenseModel := config.DB.Model(&models.Expense{})
	if companyID != "" {
		expenseModel = expenseModel.Where("company_id = ?", companyID)
	}
	if isFiltered && !startDate.IsZero() && !endDate.IsZero() {
		expenseModel = expenseModel.Where("date >= ? AND date <= ?", startDate, endDate)
	}
	row := expenseModel.Select("COALESCE(SUM(amount), 0)").Row()
	_ = row.Scan(&totalExpense)

	// 3. Keuntungan = Pendapatan - Pengeluaran
	totalProfit := totalRevenue - totalExpense

	// 4. Hitung Jumlah Event Agenda dalam periode / mendatang
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var agendaCount int64 = 0
	countQuery := config.DB.Model(&models.Agenda{}).Where("status != ?", "cancelled")
	if companyID != "" {
		countQuery = countQuery.Where("company_id = ?", companyID)
	}
	if isFiltered && !startDate.IsZero() && !endDate.IsZero() {
		countQuery = countQuery.Where("event_date_time >= ? AND event_date_time <= ?", startDate, endDate)
	} else {
		countQuery = countQuery.Where("event_date_time >= ?", startOfToday)
	}
	countQuery.Count(&agendaCount)

	// Ambil daftar booking mendatang terdekat (maks 10) untuk carousel
	var upcomingAgendas []models.Agenda
	upcomingListQuery := config.DB.Preload("ClientForm").
		Where("event_date_time >= ? AND status != ?", startOfToday, "cancelled")
	if companyID != "" {
		upcomingListQuery = upcomingListQuery.Where("company_id = ?", companyID)
	}
	upcomingListQuery.Order("event_date_time ASC").Limit(10).Find(&upcomingAgendas)

	// 5. Ambil pengeluaran terbaru (maks 10) untuk list pengeluaran terkini
	var recentExpenses []models.Expense
	recentExpenseQuery := config.DB.Order("date DESC, created_at DESC").Limit(10)
	if companyID != "" {
		recentExpenseQuery = recentExpenseQuery.Where("company_id = ?", companyID)
	}
	recentExpenseQuery.Find(&recentExpenses)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil ringkasan dashboard",
		"data": DashboardSummaryResponse{
			TotalRevenue:          totalRevenue,
			TotalExpense:          totalExpense,
			TotalProfit:           totalProfit,
			UpcomingBookingsCount: agendaCount,
			UpcomingAgendas:       upcomingAgendas,
			RecentExpenses:        recentExpenses,
			PeriodFilter:          period,
			Month:                 reqMonth,
			Year:                  reqYear,
		},
	})
}

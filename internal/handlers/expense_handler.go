package handlers

import (
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetExpenses handles GET /api/expenses
func GetExpenses(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	var expenses []models.Expense
	query := config.DB.Model(&models.Expense{})

	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	// Filter opsional bulan/tahun atau tanggal jika diperlukan
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr != "" {
		if tStart, err := time.Parse("2006-01-02", startDateStr); err == nil {
			query = query.Where("date >= ?", tStart)
		}
	}
	if endDateStr != "" {
		if tEnd, err := time.Parse("2006-01-02", endDateStr); err == nil {
			tEndInclusive := tEnd.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			query = query.Where("date <= ?", tEndInclusive)
		}
	}

	if err := query.Order("date DESC, created_at DESC").Find(&expenses).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil data pengeluaran: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil data pengeluaran",
		"data":    expenses,
	})
}

// CreateExpense handles POST /api/expenses
func CreateExpense(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	if companyID == "" {
		companyID = "cmp_default"
	}

	var req models.CreateExpenseRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Keterangan/judul pengeluaran wajib diisi",
		})
	}

	if req.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nominal pengeluaran harus lebih dari 0",
		})
	}

	expenseDate := time.Now()
	if strings.TrimSpace(req.Date) != "" {
		if parsed, err := time.Parse("2006-01-02", req.Date); err == nil {
			expenseDate = parsed
		} else if parsedIso, err := time.Parse(time.RFC3339, req.Date); err == nil {
			expenseDate = parsedIso
		}
	}

	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "Lainnya"
	}

	expense := models.Expense{
		ID:        "exp_" + uuid.New().String()[:8],
		CompanyID: companyID,
		Title:     req.Title,
		Amount:    req.Amount,
		Category:  category,
		Date:      expenseDate,
		Notes:     strings.TrimSpace(req.Notes),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := config.DB.Create(&expense).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan pengeluaran: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Pengeluaran berhasil dicatat",
		"data":    expense,
	})
}

// DeleteExpense handles DELETE /api/expenses/:id
func DeleteExpense(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID pengeluaran tidak boleh kosong",
		})
	}

	query := config.DB.Where("id = ?", id)
	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	result := query.Delete(&models.Expense{})
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menghapus pengeluaran: " + result.Error.Error(),
		})
	}

	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Pengeluaran tidak ditemukan",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Pengeluaran berhasil dihapus",
	})
}

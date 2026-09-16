package handlers

import (
	"fmt"
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetAgendas handles GET /api/agendas
// Optional query params:
// - start_date: YYYY-MM-DD
// - end_date: YYYY-MM-DD
// - status: upcoming, completed, cancelled, all
func GetAgendas(c *fiber.Ctx) error {
	var agendas []models.Agenda
	query := config.DB.Model(&models.Agenda{})

	// Filter rentang tanggal
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr != "" {
		if tStart, err := time.Parse("2006-01-02", startDateStr); err == nil {
			query = query.Where("event_date_time >= ?", tStart)
		}
	}

	if endDateStr != "" {
		if tEnd, err := time.Parse("2006-01-02", endDateStr); err == nil {
			// Tambahkan 1 hari dikurangi 1 detik agar mencakup hingga akhir hari
			tEndInclusive := tEnd.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			query = query.Where("event_date_time <= ?", tEndInclusive)
		}
	}

	// Filter status jika ada
	statusStr := c.Query("status")
	if statusStr != "" && statusStr != "all" {
		query = query.Where("status = ?", statusStr)
	}

	// Sort berdasarkan event_date_time terdekat (ascending)
	if err := query.Order("event_date_time ASC").Find(&agendas).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil data agenda: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil data agenda",
		"data":    agendas,
	})
}

// CreateAgenda handles POST /api/agendas
func CreateAgenda(c *fiber.Ctx) error {
	var req models.CreateAgendaRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	// Validasi input
	if strings.TrimSpace(req.ClientName) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nama client wajib diisi",
		})
	}
	if strings.TrimSpace(req.BackdropTitle) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nama di backdrop wajib diisi",
		})
	}
	if strings.TrimSpace(req.EventDateTime) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Tanggal dan jam acara wajib diisi",
		})
	}
	if strings.TrimSpace(req.PackageID) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Paket dekorasi wajib dipilih",
		})
	}

	// Parsing tanggal dan waktu acara
	var eventTime time.Time
	var parseErr error

	timeFormats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04",
	}

	for _, format := range timeFormats {
		eventTime, parseErr = time.Parse(format, req.EventDateTime)
		if parseErr == nil {
			break
		}
	}

	if parseErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format tanggal dan waktu tidak valid. Gunakan format ISO8601 atau YYYY-MM-DD HH:mm",
		})
	}

	// Ambil nama paket dari master Paket
	var pkg models.Package
	packageName := ""
	if err := config.DB.Where("id = ?", req.PackageID).First(&pkg).Error; err == nil {
		packageName = pkg.Name
	} else {
		packageName = "Paket Khusus"
	}

	// Buat Agenda ID baru
	agendaID := fmt.Sprintf("agd_%s", uuid.New().String()[:8])

	agenda := models.Agenda{
		ID:            agendaID,
		ClientName:    strings.TrimSpace(req.ClientName),
		BackdropTitle: strings.TrimSpace(req.BackdropTitle),
		EventDateTime: eventTime,
		MapsURL:       strings.TrimSpace(req.MapsURL),
		PackageID:     req.PackageID,
		PackageName:   packageName,
		Status:        "upcoming",
		Notes:         strings.TrimSpace(req.Notes),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := config.DB.Create(&agenda).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan agenda: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Agenda berhasil dibuat",
		"data":    agenda,
	})
}

// GetPackages handles GET /api/packages
func GetPackages(c *fiber.Ctx) error {
	var packages []models.Package
	if err := config.DB.Order("base_price ASC").Find(&packages).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil data paket",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil master paket",
		"data":    packages,
	})
}

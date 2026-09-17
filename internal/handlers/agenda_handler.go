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
func GetAgendas(c *fiber.Ctx) error {
	var agendas []models.Agenda
	query := config.DB.Model(&models.Agenda{}).Preload("ClientForm")

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

	// Pastikan semua agenda punya FormToken & FormStatus default
	for i := range agendas {
		if agendas[i].FormToken == "" {
			agendas[i].FormToken = uuid.New().String()[:8]
			if agendas[i].FormStatus == "" {
				agendas[i].FormStatus = "pending"
			}
			config.DB.Model(&models.Agenda{}).Where("id = ?", agendas[i].ID).Updates(map[string]interface{}{
				"form_token":  agendas[i].FormToken,
				"form_status": agendas[i].FormStatus,
			})
		}
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

	// Validasi input (hanya nama client yang strictly required di HTTP API)
	if strings.TrimSpace(req.ClientName) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nama client wajib diisi",
		})
	}

	backdropTitle := strings.TrimSpace(req.BackdropTitle)
	if backdropTitle == "" {
		backdropTitle = fmt.Sprintf("Acara %s", strings.TrimSpace(req.ClientName))
	}

	// Parsing tanggal dan waktu acara (jika ada)
	var eventTime time.Time
	if strings.TrimSpace(req.EventDateTime) != "" {
		timeFormats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
			"2006-01-02 15:04",
			"2006-01-02T15:04",
		}
		for _, format := range timeFormats {
			if t, err := time.Parse(format, req.EventDateTime); err == nil {
				eventTime = t
				break
			}
		}
	}
	if eventTime.IsZero() {
		eventTime = time.Now().Add(24 * time.Hour) // Default besok jika tidak diisi
	}

	// Ambil nama paket dari master Paket (jika ada)
	packageName := "Belum Dipilih"
	if strings.TrimSpace(req.PackageID) != "" {
		var pkg models.Package
		if err := config.DB.Where("id = ?", req.PackageID).First(&pkg).Error; err == nil {
			packageName = pkg.Name
		} else {
			packageName = "Paket Khusus"
		}
	}

	// Buat Agenda ID & Form Token baru
	agendaID := fmt.Sprintf("agd_%s", uuid.New().String()[:8])
	formToken := uuid.New().String()[:8]

	agenda := models.Agenda{
		ID:            agendaID,
		ClientName:    strings.TrimSpace(req.ClientName),
		ClientPhone:   strings.TrimSpace(req.ClientPhone),
		BackdropTitle: backdropTitle,
		EventDateTime: eventTime,
		MapsURL:       strings.TrimSpace(req.MapsURL),
		PackageID:     req.PackageID,
		PackageName:   packageName,
		Status:        "upcoming",
		FormToken:     formToken,
		FormStatus:    "pending",
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

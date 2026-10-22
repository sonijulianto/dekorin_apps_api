package handlers

import (
	"fmt"
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetAgendas handles GET /api/agendas
func GetAgendas(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	var agendas []models.Agenda
	query := config.DB.Model(&models.Agenda{}).Preload("ClientForm")

	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

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

	// Pastikan semua agenda punya FormToken & FormStatus default, serta backfill package_price jika 0
	for i := range agendas {
		if agendas[i].PackagePrice == 0 && agendas[i].PackageID != "" {
			var pkg models.Package
			if err := config.DB.Where("id = ?", agendas[i].PackageID).First(&pkg).Error; err == nil {
				agendas[i].PackagePrice = pkg.BasePrice
				config.DB.Model(&models.Agenda{}).Where("id = ?", agendas[i].ID).Update("package_price", pkg.BasePrice)
			}
		}

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

	// Ambil nama & harga paket dari master Paket (jika ada)
	packageName := "Belum Dipilih"
	packagePrice := req.PackagePrice
	if strings.TrimSpace(req.PackageID) != "" {
		var pkg models.Package
		if err := config.DB.Where("id = ?", req.PackageID).First(&pkg).Error; err == nil {
			packageName = pkg.Name
			if packagePrice == 0 {
				packagePrice = pkg.BasePrice
			}
		} else {
			packageName = "Paket Khusus"
		}
	}

	// Buat Agenda ID & Form Token baru
	agendaID := fmt.Sprintf("agd_%s", uuid.New().String()[:8])
	formToken := uuid.New().String()[:8]
	companyID := middleware.GetCompanyID(c)
	if companyID == "" {
		companyID = "cmp_default"
	}

	agenda := models.Agenda{
		ID:            agendaID,
		CompanyID:     companyID,
		ClientName:    strings.TrimSpace(req.ClientName),
		ClientPhone:   strings.TrimSpace(req.ClientPhone),
		BackdropTitle: backdropTitle,
		EventDateTime: eventTime,
		MapsURL:       strings.TrimSpace(req.MapsURL),
		PackageID:     req.PackageID,
		PackageName:   packageName,
		PackagePrice:  packagePrice,
		TransportFee:  req.TransportFee,
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
	companyID := middleware.GetCompanyID(c)
	var packages []models.Package
	query := config.DB.Model(&models.Package{})
	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.Order("sort_order ASC, created_at ASC").Find(&packages).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil data paket",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil master paket",
		"data":    packages,
	})
}

// ReorderPackages handles PUT /api/packages/reorder
func ReorderPackages(c *fiber.Ctx) error {
	var req models.ReorderPackagesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	for idx, id := range req.PackageIDs {
		config.DB.Model(&models.Package{}).Where("id = ?", id).Update("sort_order", idx)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Urutan paket dekorasi berhasil diperbarui",
	})
}

// UpdateAgendaDp handles PUT /api/agendas/:id/dp
func UpdateAgendaDp(c *fiber.Ctx) error {
	id := c.Params("id")
	var req models.UpdateDpRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	if req.DpAmount < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nominal DP tidak boleh negatif",
		})
	}

	var agenda models.Agenda
	if err := config.DB.Preload("ClientForm").Where("id = ?", id).First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Agenda tidak ditemukan",
		})
	}

	agenda.DpAmount = req.DpAmount
	totalBiaya := agenda.CalculateTotalBiaya()
	if totalBiaya > 0 && agenda.DpAmount >= totalBiaya {
		agenda.PaymentStatus = "paid"
	} else if agenda.DpAmount > 0 {
		agenda.PaymentStatus = "dp"
	}

	if err := config.DB.Save(&agenda).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengupdate DP agenda: " + err.Error(),
		})
	}

	// Reload with ClientForm
	config.DB.Model(&models.Agenda{}).Preload("ClientForm").Where("id = ?", id).First(&agenda)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Pembayaran DP berhasil diperbarui",
		"data":    agenda,
	})
}

// UpdateAgendaPaymentStatus handles PUT /api/agendas/:id/payment-status
func UpdateAgendaPaymentStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	var req models.UpdatePaymentStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	validStatuses := map[string]bool{
		"unpaid": true,
		"dp":     true,
		"paid":   true,
	}
	req.PaymentStatus = strings.ToLower(strings.TrimSpace(req.PaymentStatus))
	if !validStatuses[req.PaymentStatus] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Status pembayaran tidak valid. Pilihan: unpaid, dp, paid",
		})
	}

	var agenda models.Agenda
	if err := config.DB.Preload("ClientForm").Where("id = ?", id).First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Agenda tidak ditemukan",
		})
	}

	agenda.PaymentStatus = req.PaymentStatus
	// Jika status diubah menjadi paid dan dp_amount belum menyamai total, opsional update dp_amount
	if req.PaymentStatus == "paid" {
		totalBiaya := agenda.CalculateTotalBiaya()
		if totalBiaya > 0 && agenda.DpAmount < totalBiaya {
			agenda.DpAmount = totalBiaya
		}
	}

	if err := config.DB.Save(&agenda).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memperbarui status pembayaran: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Status pembayaran berhasil diperbarui",
		"data":    agenda,
	})
}

// UpdateAgendaStatus handles PUT /api/agendas/:id/status
func UpdateAgendaStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	var req models.UpdateEventStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	validStatuses := map[string]bool{
		"upcoming":  true,
		"completed": true,
		"cancelled": true,
	}
	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	if !validStatuses[req.Status] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Status acara tidak valid. Pilihan: upcoming, completed, cancelled",
		})
	}

	var agenda models.Agenda
	if err := config.DB.Preload("ClientForm").Where("id = ?", id).First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Agenda tidak ditemukan",
		})
	}

	agenda.Status = req.Status
	if err := config.DB.Save(&agenda).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memperbarui status acara: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Status acara berhasil diperbarui",
		"data":    agenda,
	})
}

// UpdateAgendaTransport handles PUT /api/agendas/:id/transport
func UpdateAgendaTransport(c *fiber.Ctx) error {
	id := c.Params("id")
	var req models.UpdateTransportRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	if req.TransportFee < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nominal uang transport tidak boleh negatif",
		})
	}

	var agenda models.Agenda
	if err := config.DB.Where("id = ?", id).First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Agenda tidak ditemukan",
		})
	}

	agenda.TransportFee = req.TransportFee
	if err := config.DB.Save(&agenda).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengupdate uang transport agenda: " + err.Error(),
		})
	}

	// Reload with ClientForm
	config.DB.Model(&models.Agenda{}).Preload("ClientForm").Where("id = ?", id).First(&agenda)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Uang transport berhasil diperbarui",
		"data":    agenda,
	})
}

// UpdateAgendaPackage handles PUT /api/agendas/:id/package
func UpdateAgendaPackage(c *fiber.Ctx) error {
	id := c.Params("id")
	var req models.UpdatePackageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	if req.PackagePrice < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Harga paket tidak boleh negatif",
		})
	}

	var agenda models.Agenda
	if err := config.DB.Where("id = ?", id).First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Agenda tidak ditemukan",
		})
	}

	packageName := strings.TrimSpace(req.PackageName)
	packagePrice := req.PackagePrice

	if strings.TrimSpace(req.PackageID) != "" {
		var pkg models.Package
		if err := config.DB.Where("id = ?", req.PackageID).First(&pkg).Error; err == nil {
			if packageName == "" {
				packageName = pkg.Name
			}
			if packagePrice <= 0 {
				packagePrice = pkg.BasePrice
			}
		}
	}

	if packageName == "" {
		packageName = "Belum Dipilih"
	}

	agenda.PackageID = req.PackageID
	agenda.PackageName = packageName
	agenda.PackagePrice = packagePrice

	if err := config.DB.Save(&agenda).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengupdate paket agenda: " + err.Error(),
		})
	}

	// Update ClientForm jika ada agar sinkron
	var existingForm models.ClientForm
	if err := config.DB.Where("agenda_id = ?", id).First(&existingForm).Error; err == nil {
		existingForm.PackageID = req.PackageID
		existingForm.PackageName = packageName
		existingForm.PackagePrice = packagePrice
		existingForm.TotalPrice = packagePrice + existingForm.AddonsPrice
		config.DB.Save(&existingForm)
	}

	// Reload with ClientForm
	config.DB.Model(&models.Agenda{}).Preload("ClientForm").Where("id = ?", id).First(&agenda)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Paket agenda berhasil diperbarui",
		"data":    agenda,
	})
}


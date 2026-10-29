package handlers

import (
	"fmt"
	"net/url"
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
	userRole, _ := c.Locals("user_role").(string)
	var agendas []models.Agenda
	query := config.DB.Model(&models.Agenda{}).Preload("ClientForm")

	if strings.EqualFold(userRole, "superadmin") {
		if qComp := c.Query("company_id"); qComp != "" && qComp != "all" {
			query = query.Where("company_id = ?", qComp)
		}
	} else if companyID != "" {
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

		// Sinkronkan informasi Brand jika agenda memiliki brand_id
		if agendas[i].BrandID != "" && agendas[i].BrandName == "" {
			var brand models.CompanyBrand
			if err := config.DB.Where("id = ?", agendas[i].BrandID).First(&brand).Error; err == nil {
				agendas[i].BrandName = brand.Name
				agendas[i].BrandLogo = brand.LogoURL
				agendas[i].BrandPhone = brand.Phone
				agendas[i].BrandInstagram = brand.Instagram
				agendas[i].BrandDescription = brand.Description
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

		if agendas[i].EventAddress == "" && agendas[i].ClientForm != nil && agendas[i].ClientForm.EventAddress != "" {
			agendas[i].EventAddress = agendas[i].ClientForm.EventAddress
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
	brandID := strings.TrimSpace(req.BrandID)
	if strings.TrimSpace(req.PackageID) != "" {
		var pkg models.Package
		if err := config.DB.Where("id = ?", req.PackageID).First(&pkg).Error; err == nil {
			packageName = pkg.Name
			if packagePrice == 0 {
				packagePrice = pkg.BasePrice
			}
			if brandID == "" && pkg.BrandID != "" {
				brandID = pkg.BrandID
			}
		} else {
			packageName = "Paket Khusus"
		}
	}

	// Cek Model Dekorasi jika dipilih
	modelID := strings.TrimSpace(req.ModelID)
	modelName := strings.TrimSpace(req.ModelName)
	modelWidth := strings.TrimSpace(req.ModelWidth)
	modelImageURL := strings.TrimSpace(req.ModelImageURL)
	modelAdditionalPrice := req.ModelAdditionalPrice

	if modelID != "" && (modelName == "" || modelWidth == "" || modelImageURL == "") {
		var decorModel models.DecorationModel
		if err := config.DB.Where("id = ?", modelID).First(&decorModel).Error; err == nil {
			if modelName == "" {
				modelName = decorModel.Name
			}
			if modelWidth == "" {
				modelWidth = decorModel.Width
			}
			if modelImageURL == "" {
				modelImageURL = decorModel.ImageURL
			}
			if brandID == "" && decorModel.BrandID != "" {
				brandID = decorModel.BrandID
			}
		}
	}

	// Buat Agenda ID & Form Token baru
	agendaID := fmt.Sprintf("agd_%s", uuid.New().String()[:8])
	formToken := uuid.New().String()[:8]
	companyID := middleware.GetCompanyID(c)
	userRole, _ := c.Locals("user_role").(string)
	if strings.EqualFold(userRole, "superadmin") {
		if qComp := c.Query("company_id"); qComp != "" && qComp != "all" {
			companyID = qComp
		}
	}
	if companyID == "" {
		companyID = "cmp_default"
	}

	// Ambil data sub-brand jika dipilih
	brandName := ""
	brandLogo := ""
	brandPhone := ""
	brandInstagram := ""
	brandDescription := ""
	if brandID != "" {
		var brand models.CompanyBrand
		if err := config.DB.Where("id = ? AND company_id = ?", brandID, companyID).First(&brand).Error; err == nil {
			brandName = brand.Name
			brandLogo = brand.LogoURL
			brandPhone = brand.Phone
			brandInstagram = brand.Instagram
			brandDescription = brand.Description
		} else {
			brandID = ""
		}
	}

	mapsURL := strings.TrimSpace(req.MapsURL)
	if mapsURL != "" && !strings.HasPrefix(mapsURL, "http://") && !strings.HasPrefix(mapsURL, "https://") {
		mapsURL = fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%s", url.QueryEscape(mapsURL))
	}

	agenda := models.Agenda{
		ID:                   agendaID,
		CompanyID:            companyID,
		BrandID:              brandID,
		BrandName:            brandName,
		BrandLogo:            brandLogo,
		BrandPhone:           brandPhone,
		BrandInstagram:       brandInstagram,
		BrandDescription:     brandDescription,
		ClientName:           strings.TrimSpace(req.ClientName),
		ClientPhone:          strings.TrimSpace(req.ClientPhone),
		BackdropTitle:        backdropTitle,
		EventDateTime:        eventTime,
		EventAddress:         strings.TrimSpace(req.EventAddress),
		MapsURL:              mapsURL,
		PackageID:            req.PackageID,
		PackageName:          packageName,
		PackagePrice:         packagePrice,
		ModelID:              modelID,
		ModelName:            modelName,
		ModelWidth:           modelWidth,
		ModelImageURL:        modelImageURL,
		ModelAdditionalPrice: modelAdditionalPrice,
		TransportFee:         req.TransportFee,
		Status:               "upcoming",
		FormToken:            formToken,
		FormStatus:           "pending",
		Notes:                strings.TrimSpace(req.Notes),
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if err := config.DB.Create(&agenda).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan agenda: " + err.Error(),
		})
	}

	// Simpan atau Auto-populate Pick List Agenda
	if len(req.AgendaProperties) > 0 {
		for _, ap := range req.AgendaProperties {
			qty := ap.Qty
			if qty <= 0 {
				qty = 1
			}
			item := models.AgendaProperty{
				ID:         fmt.Sprintf("ap_%s", uuid.New().String()[:8]),
				CompanyID:  companyID,
				AgendaID:   agenda.ID,
				PropertyID: ap.PropertyID,
				CustomName: ap.CustomName,
				Qty:        qty,
				Notes:      ap.Notes,
				IsPacked:   ap.IsPacked,
				IsReturned: ap.IsReturned,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			config.DB.Create(&item)
		}
	} else if modelID != "" {
		var modelProps []models.DecorationModelProperty
		config.DB.Where("decoration_model_id = ?", modelID).Find(&modelProps)
		for _, mp := range modelProps {
			item := models.AgendaProperty{
				ID:         fmt.Sprintf("ap_%s", uuid.New().String()[:8]),
				CompanyID:  companyID,
				AgendaID:   agenda.ID,
				PropertyID: mp.PropertyID,
				Qty:        mp.DefaultQty,
				Notes:      mp.Notes,
				IsPacked:   false,
				IsReturned: false,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			config.DB.Create(&item)
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Agenda berhasil dibuat",
		"data":    agenda,
	})
}

// GetPackages handles GET /api/packages
func GetPackages(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	userRole, _ := c.Locals("user_role").(string)
	var packages []models.Package
	query := config.DB.Model(&models.Package{}).
		Preload("PackageModels").
		Preload("PackageModels.DecorationModel")
	if strings.EqualFold(userRole, "superadmin") {
		if qComp := c.Query("company_id"); qComp != "" && qComp != "all" {
			query = query.Where("company_id = ?", qComp)
		}
	} else if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	// Opsional filter berdasarkan brand_id
	if qBrand := c.Query("brand_id"); qBrand != "" && qBrand != "all" {
		if qBrand == "main" || qBrand == "default" {
			query = query.Where("brand_id = '' OR brand_id IS NULL")
		} else {
			query = query.Where("brand_id = ?", qBrand)
		}
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

	if strings.ToLower(agenda.Status) == "completed" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Agenda sudah selesai, uang transport tidak dapat diubah",
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

	if strings.ToLower(agenda.Status) == "completed" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Agenda sudah selesai, paket dekorasi tidak dapat diubah",
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

	// Update model info jika disediakan
	if req.ModelID != "" {
		agenda.ModelID = req.ModelID
		agenda.ModelName = req.ModelName
		agenda.ModelWidth = req.ModelWidth
		agenda.ModelImageURL = req.ModelImageURL
		agenda.ModelAdditionalPrice = req.ModelAdditionalPrice

		if (agenda.ModelName == "" || agenda.ModelWidth == "" || agenda.ModelImageURL == "") && agenda.ModelID != "" {
			var decorModel models.DecorationModel
			if err := config.DB.Where("id = ?", agenda.ModelID).First(&decorModel).Error; err == nil {
				if agenda.ModelName == "" {
					agenda.ModelName = decorModel.Name
				}
				if agenda.ModelWidth == "" {
					agenda.ModelWidth = decorModel.Width
				}
				if agenda.ModelImageURL == "" {
					agenda.ModelImageURL = decorModel.ImageURL
				}
			}
		}
	}

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

// UpdateAgendaModel handles PUT /api/agendas/:id/model
func UpdateAgendaModel(c *fiber.Ctx) error {
	id := c.Params("id")
	var req models.UpdateModelRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	companyID := middleware.GetCompanyID(c)
	userRole, _ := c.Locals("user_role").(string)

	var agenda models.Agenda
	query := config.DB.Where("id = ?", id)
	if userRole != "superadmin" && companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	if err := query.First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Agenda tidak ditemukan",
		})
	}

	// Cek apakah agenda sudah selesai atau dibatalkan
	if strings.ToLower(agenda.Status) == "completed" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Agenda sudah selesai, model dekorasi tidak dapat diubah",
		})
	}

	modelID := strings.TrimSpace(req.ModelID)
	modelName := strings.TrimSpace(req.ModelName)
	modelWidth := strings.TrimSpace(req.ModelWidth)
	modelImageURL := strings.TrimSpace(req.ModelImageURL)
	modelAdditionalPrice := req.ModelAdditionalPrice

	if modelID != "" {
		var decorModel models.DecorationModel
		if err := config.DB.Where("id = ?", modelID).First(&decorModel).Error; err == nil {
			if modelName == "" {
				modelName = decorModel.Name
			}
			if modelWidth == "" {
				modelWidth = decorModel.Width
			}
			if modelImageURL == "" {
				modelImageURL = decorModel.ImageURL
			}
		}

		// Jika ada paket terkait, cek harga tambahan dari relasi package_models
		if agenda.PackageID != "" && modelAdditionalPrice <= 0 {
			var pkgModel models.PackageDecorationModel
			if err := config.DB.Where("package_id = ? AND decoration_model_id = ?", agenda.PackageID, modelID).First(&pkgModel).Error; err == nil {
				modelAdditionalPrice = pkgModel.AdditionalPrice
			}
		}
	} else {
		// Kosongkan model jika tidak ada modelID
		modelName = ""
		modelWidth = ""
		modelImageURL = ""
		modelAdditionalPrice = 0
	}

	agenda.ModelID = modelID
	agenda.ModelName = modelName
	agenda.ModelWidth = modelWidth
	agenda.ModelImageURL = modelImageURL
	agenda.ModelAdditionalPrice = modelAdditionalPrice
	agenda.UpdatedAt = time.Now()

	if err := config.DB.Save(&agenda).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengupdate model dekorasi agenda: " + err.Error(),
		})
	}

	// Reload with ClientForm
	config.DB.Model(&models.Agenda{}).Preload("ClientForm").Where("id = ?", id).First(&agenda)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Model dekorasi berhasil diperbarui",
		"data":    agenda,
	})
}

// UpdateAgendaNotes handles PUT /api/agendas/:id/notes
func UpdateAgendaNotes(c *fiber.Ctx) error {
	id := c.Params("id")
	var req models.UpdateNotesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	companyID := middleware.GetCompanyID(c)
	userRole, _ := c.Locals("user_role").(string)

	var agenda models.Agenda
	query := config.DB.Where("id = ?", id)
	if userRole != "superadmin" && companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	if err := query.First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Agenda tidak ditemukan",
		})
	}

	// Cek apakah agenda sudah selesai
	if strings.ToLower(agenda.Status) == "completed" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Agenda sudah selesai, catatan / request tambahan tidak dapat diubah",
		})
	}

	newNotes := strings.TrimSpace(req.Notes)
	agenda.Notes = newNotes
	agenda.UpdatedAt = time.Now()

	if err := config.DB.Save(&agenda).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengupdate catatan agenda: " + err.Error(),
		})
	}

	// Update ClientForm jika ada agar tetap sinkron
	var clientForm models.ClientForm
	if err := config.DB.Where("agenda_id = ?", id).First(&clientForm).Error; err == nil {
		clientForm.SpecialRequests = newNotes
		config.DB.Save(&clientForm)
	}

	// Reload with ClientForm
	config.DB.Model(&models.Agenda{}).Preload("ClientForm").Where("id = ?", id).First(&agenda)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Catatan agenda berhasil diperbarui",
		"data":    agenda,
	})
}

// UpdateAgendaBrand handles PUT /api/agendas/:id/brand
func UpdateAgendaBrand(c *fiber.Ctx) error {
	id := c.Params("id")
	var req models.UpdateAgendaBrandRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	companyID := middleware.GetCompanyID(c)
	userRole, _ := c.Locals("user_role").(string)

	var agenda models.Agenda
	query := config.DB.Where("id = ?", id)
	if !strings.EqualFold(userRole, "superadmin") && companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	if err := query.First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Agenda tidak ditemukan",
		})
	}

	brandID := strings.TrimSpace(req.BrandID)
	if brandID == "" {
		agenda.BrandID = ""
		agenda.BrandName = ""
		agenda.BrandLogo = ""
		agenda.BrandPhone = ""
		agenda.BrandInstagram = ""
		agenda.BrandDescription = ""
	} else {
		var brand models.CompanyBrand
		if err := config.DB.Where("id = ? AND company_id = ?", brandID, agenda.CompanyID).First(&brand).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Brand tidak ditemukan",
			})
		}
		agenda.BrandID = brand.ID
		agenda.BrandName = brand.Name
		agenda.BrandLogo = brand.LogoURL
		agenda.BrandPhone = brand.Phone
		agenda.BrandInstagram = brand.Instagram
		agenda.BrandDescription = brand.Description
	}

	agenda.UpdatedAt = time.Now()
	if err := config.DB.Save(&agenda).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memperbarui brand agenda: " + err.Error(),
		})
	}

	config.DB.Model(&models.Agenda{}).Preload("ClientForm").Where("id = ?", id).First(&agenda)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Brand agenda berhasil diperbarui",
		"data":    agenda,
	})
}

// DeleteAgenda handles DELETE /api/agendas/:id
func DeleteAgenda(c *fiber.Ctx) error {
	id := c.Params("id")
	companyID := middleware.GetCompanyID(c)
	userRole, _ := c.Locals("user_role").(string)

	query := config.DB.Where("id = ?", id)
	if !strings.EqualFold(userRole, "superadmin") && companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	var agenda models.Agenda
	if err := query.First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Agenda tidak ditemukan",
		})
	}

	// Hapus client_form terkait jika ada
	config.DB.Where("agenda_id = ?", id).Delete(&models.ClientForm{})

	if err := config.DB.Delete(&agenda).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menghapus agenda: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Agenda berhasil dihapus",
	})
}

// UpdateAgendaSchedule handles PUT /api/agendas/:id/schedule (Pembaruan tanggal dan jam acara selama aktif)
func UpdateAgendaSchedule(c *fiber.Ctx) error {
	id := c.Params("id")
	companyID := middleware.GetCompanyID(c)
	userRole, _ := c.Locals("user_role").(string)

	var req models.UpdateAgendaScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	query := config.DB.Preload("ClientForm").Where("id = ?", id)
	if !strings.EqualFold(userRole, "superadmin") && companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	var agenda models.Agenda
	if err := query.First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Agenda tidak ditemukan",
		})
	}

	if strings.ToLower(agenda.Status) == "completed" || strings.ToLower(agenda.Status) == "cancelled" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Agenda sudah selesai atau dibatalkan, jadwal acara tidak dapat diubah.",
		})
	}

	timeFormats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04",
	}
	parsed := false
	for _, format := range timeFormats {
		if t, err := time.Parse(format, strings.TrimSpace(req.EventDateTime)); err == nil {
			agenda.EventDateTime = t
			parsed = true
			break
		}
	}

	if !parsed {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format tanggal dan waktu tidak valid",
		})
	}

	agenda.UpdatedAt = time.Now()
	if err := config.DB.Save(&agenda).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengupdate jadwal agenda: " + err.Error(),
		})
	}

	config.DB.Model(&models.Agenda{}).Preload("ClientForm").Where("id = ?", id).First(&agenda)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Jadwal acara berhasil diperbarui",
		"data":    agenda,
	})
}

// UpdateAgendaLocation handles PUT /api/agendas/:id/location (Pembaruan alamat & Maps lokasi selama aktif)
func UpdateAgendaLocation(c *fiber.Ctx) error {
	id := c.Params("id")
	companyID := middleware.GetCompanyID(c)
	userRole, _ := c.Locals("user_role").(string)

	var req models.UpdateAgendaLocationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	query := config.DB.Preload("ClientForm").Where("id = ?", id)
	if !strings.EqualFold(userRole, "superadmin") && companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	var agenda models.Agenda
	if err := query.First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Agenda tidak ditemukan",
		})
	}

	if strings.ToLower(agenda.Status) == "completed" || strings.ToLower(agenda.Status) == "cancelled" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Agenda sudah selesai atau dibatalkan, alamat acara tidak dapat diubah.",
		})
	}

	agenda.EventAddress = strings.TrimSpace(req.EventAddress)

	mapsURL := strings.TrimSpace(req.MapsURL)
	if mapsURL != "" && !strings.HasPrefix(mapsURL, "http://") && !strings.HasPrefix(mapsURL, "https://") {
		mapsURL = fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%s", url.QueryEscape(mapsURL))
	}
	agenda.MapsURL = mapsURL
	agenda.UpdatedAt = time.Now()

	if err := config.DB.Save(&agenda).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengupdate lokasi agenda: " + err.Error(),
		})
	}

	// Sinkronkan ke ClientForm jika ada
	if agenda.ClientForm != nil {
		config.DB.Model(&models.ClientForm{}).Where("id = ?", agenda.ClientForm.ID).Update("event_address", agenda.EventAddress)
		agenda.ClientForm.EventAddress = agenda.EventAddress
	} else if agenda.EventAddress != "" {
		config.DB.Model(&models.ClientForm{}).Where("agenda_id = ?", agenda.ID).Update("event_address", agenda.EventAddress)
	}

	config.DB.Model(&models.Agenda{}).Preload("ClientForm").Where("id = ?", id).First(&agenda)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Alamat dan lokasi acara berhasil diperbarui",
		"data":    agenda,
	})
}

// UpdateAgenda handles PUT /api/agendas/:id (Pembaruan lengkap agenda acara selama agenda masih aktif)
func UpdateAgenda(c *fiber.Ctx) error {
	id := c.Params("id")
	companyID := middleware.GetCompanyID(c)
	userRole, _ := c.Locals("user_role").(string)

	var req models.UpdateAgendaRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	query := config.DB.Preload("ClientForm").Where("id = ?", id)
	if !strings.EqualFold(userRole, "superadmin") && companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	var agenda models.Agenda
	if err := query.First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Agenda tidak ditemukan",
		})
	}

	// Aturan: kalau agenda sudah selesai atau dibatalkan, data acara tidak dapat diubah
	isCompleted := strings.ToLower(agenda.Status) == "completed"
	isCancelled := strings.ToLower(agenda.Status) == "cancelled"

	if isCompleted || isCancelled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Agenda sudah selesai atau dibatalkan, rincian acara tidak dapat diubah lagi.",
		})
	}

	// Validasi nama klien
	if strings.TrimSpace(req.ClientName) != "" {
		agenda.ClientName = strings.TrimSpace(req.ClientName)
	}
	if req.ClientPhone != "" {
		agenda.ClientPhone = strings.TrimSpace(req.ClientPhone)
	}
	if req.BackdropTitle != "" {
		agenda.BackdropTitle = strings.TrimSpace(req.BackdropTitle)
	}

	// Parsing tanggal acara baru jika diberikan
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
				agenda.EventDateTime = t
				break
			}
		}
	}

	// Event Address
	if req.EventAddress != "" {
		agenda.EventAddress = strings.TrimSpace(req.EventAddress)
		if agenda.ClientForm != nil {
			config.DB.Model(&models.ClientForm{}).Where("id = ?", agenda.ClientForm.ID).Update("event_address", agenda.EventAddress)
			agenda.ClientForm.EventAddress = agenda.EventAddress
		}
	}

	// Maps URL
	if req.MapsURL != "" {
		mapsURL := strings.TrimSpace(req.MapsURL)
		if mapsURL != "" && !strings.HasPrefix(mapsURL, "http://") && !strings.HasPrefix(mapsURL, "https://") {
			mapsURL = fmt.Sprintf("https://www.google.com/maps/search/?api=1&query=%s", url.QueryEscape(mapsURL))
		}
		agenda.MapsURL = mapsURL
	}

	// Update Brand jika dipilih
	brandID := strings.TrimSpace(req.BrandID)
	if brandID != "" && brandID != agenda.BrandID {
		var brand models.CompanyBrand
		targetCompanyID := agenda.CompanyID
		if err := config.DB.Where("id = ? AND company_id = ?", brandID, targetCompanyID).First(&brand).Error; err == nil {
			agenda.BrandID = brand.ID
			agenda.BrandName = brand.Name
			agenda.BrandLogo = brand.LogoURL
			agenda.BrandPhone = brand.Phone
			agenda.BrandInstagram = brand.Instagram
			agenda.BrandDescription = brand.Description
		}
	} else if brandID == "" && req.BrandID != "" {
		agenda.BrandID = ""
		agenda.BrandName = ""
		agenda.BrandLogo = ""
		agenda.BrandPhone = ""
		agenda.BrandInstagram = ""
		agenda.BrandDescription = ""
	}

	// Paket Dekorasi
	if strings.TrimSpace(req.PackageID) != "" {
		agenda.PackageID = strings.TrimSpace(req.PackageID)
		if strings.TrimSpace(req.PackageName) != "" {
			agenda.PackageName = strings.TrimSpace(req.PackageName)
		} else {
			var pkg models.Package
			if err := config.DB.Where("id = ?", agenda.PackageID).First(&pkg).Error; err == nil {
				agenda.PackageName = pkg.Name
				if req.PackagePrice == 0 {
					agenda.PackagePrice = pkg.BasePrice
				}
			}
		}
	} else if strings.TrimSpace(req.PackageName) != "" {
		agenda.PackageName = strings.TrimSpace(req.PackageName)
	}
	if req.PackagePrice >= 0 {
		agenda.PackagePrice = req.PackagePrice
	}

	// Model Dekorasi
	modelID := strings.TrimSpace(req.ModelID)
	if modelID != "" {
		agenda.ModelID = modelID
		agenda.ModelName = strings.TrimSpace(req.ModelName)
		agenda.ModelWidth = strings.TrimSpace(req.ModelWidth)
		agenda.ModelImageURL = strings.TrimSpace(req.ModelImageURL)
		agenda.ModelAdditionalPrice = req.ModelAdditionalPrice

		if agenda.ModelName == "" || agenda.ModelImageURL == "" {
			var decorModel models.DecorationModel
			if err := config.DB.Where("id = ?", modelID).First(&decorModel).Error; err == nil {
				if agenda.ModelName == "" {
					agenda.ModelName = decorModel.Name
				}
				if agenda.ModelWidth == "" {
					agenda.ModelWidth = decorModel.Width
				}
				if agenda.ModelImageURL == "" {
					agenda.ModelImageURL = decorModel.ImageURL
				}
			}
		}
	} else if req.ModelID != "" {
		agenda.ModelID = ""
		agenda.ModelName = ""
		agenda.ModelWidth = ""
		agenda.ModelImageURL = ""
		agenda.ModelAdditionalPrice = 0
	}

	// Transport fee
	agenda.TransportFee = req.TransportFee

	// DP
	if req.DpAmount >= 0 {
		agenda.DpAmount = req.DpAmount
	}

	// Payment status
	if req.PaymentStatus != "" {
		pStatus := strings.ToLower(strings.TrimSpace(req.PaymentStatus))
		if pStatus == "paid" || pStatus == "dp" || pStatus == "unpaid" {
			agenda.PaymentStatus = pStatus
		}
	}

	// Status acara
	if req.Status != "" {
		status := strings.ToLower(strings.TrimSpace(req.Status))
		if status == "upcoming" || status == "completed" || status == "cancelled" {
			agenda.Status = status
		}
	}

	// Notes
	agenda.Notes = strings.TrimSpace(req.Notes)
	agenda.UpdatedAt = time.Now()

	if err := config.DB.Save(&agenda).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengupdate agenda: " + err.Error(),
		})
	}

	config.DB.Model(&models.Agenda{}).Preload("ClientForm").Where("id = ?", id).First(&agenda)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Agenda acara berhasil diperbarui",
		"data":    agenda,
	})
}



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

// GetProperties handles GET /api/properties
func GetProperties(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	userRole, _ := c.Locals("user_role").(string)

	query := config.DB.Model(&models.Property{})

	if strings.EqualFold(userRole, "superadmin") {
		if qComp := c.Query("company_id"); qComp != "" && qComp != "all" {
			query = query.Where("company_id = ?", qComp)
		}
	} else if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	// Filter berdasarkan tipe: "property" atau "tool"
	typeFilter := strings.TrimSpace(c.Query("type"))
	if typeFilter != "" && typeFilter != "all" {
		query = query.Where("type = ?", typeFilter)
	}

	// Filter kategori jika ada
	categoryFilter := strings.TrimSpace(c.Query("category"))
	if categoryFilter != "" && categoryFilter != "all" {
		query = query.Where("category = ?", categoryFilter)
	}

	// Pencarian nama
	search := strings.TrimSpace(c.Query("search"))
	if search != "" {
		query = query.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(search)+"%")
	}

	var properties []models.Property
	if err := query.Order("sort_order ASC, name ASC").Find(&properties).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data properti: " + err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil data master properti & alat",
		"data":    properties,
	})
}

// CreateProperty handles POST /api/properties
func CreateProperty(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	if companyID == "" {
		companyID = "cmp_default"
	}

	var req models.CreateOrUpdatePropertyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama properti/alat wajib diisi"})
	}

	propType := strings.ToLower(strings.TrimSpace(req.Type))
	if propType != "tool" {
		propType = "property"
	}

	trackStock := true
	if propType == "tool" {
		trackStock = false
	}
	if req.TrackStock != nil {
		trackStock = *req.TrackStock
	}

	totalStock := req.TotalStock
	if totalStock < 0 {
		totalStock = 0
	}
	if totalStock == 0 && propType == "property" {
		totalStock = 1
	}

	unit := strings.TrimSpace(req.Unit)
	if unit == "" {
		unit = "pcs"
	}

	category := strings.TrimSpace(req.Category)
	if category == "" {
		if propType == "tool" {
			category = "Alat Kerja"
		} else {
			category = "Lainnya"
		}
	}

	prefix := "prop"
	if propType == "tool" {
		prefix = "tool"
	}

	prop := models.Property{
		ID:         fmt.Sprintf("%s_%s", prefix, uuid.New().String()[:8]),
		CompanyID:  companyID,
		Name:       name,
		Category:   category,
		Type:       propType,
		TrackStock: trackStock,
		TotalStock: totalStock,
		Unit:       unit,
		Notes:      strings.TrimSpace(req.Notes),
		ImageURL:   strings.TrimSpace(req.ImageURL),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := config.DB.Create(&prop).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan properti: " + err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Master properti berhasil ditambahkan",
		"data":    prop,
	})
}

// UpdateProperty handles PUT /api/properties/:id
func UpdateProperty(c *fiber.Ctx) error {
	id := c.Params("id")
	companyID := middleware.GetCompanyID(c)

	var prop models.Property
	query := config.DB.Where("id = ?", id)
	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.First(&prop).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data properti tidak ditemukan"})
	}

	var req models.CreateOrUpdatePropertyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if strings.TrimSpace(req.Name) != "" {
		prop.Name = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.Category) != "" {
		prop.Category = strings.TrimSpace(req.Category)
	}
	if strings.TrimSpace(req.Type) != "" {
		prop.Type = strings.ToLower(strings.TrimSpace(req.Type))
	}
	if req.TrackStock != nil {
		prop.TrackStock = *req.TrackStock
	}
	if req.TotalStock >= 0 {
		prop.TotalStock = req.TotalStock
	}
	if strings.TrimSpace(req.Unit) != "" {
		prop.Unit = strings.TrimSpace(req.Unit)
	}
	prop.Notes = strings.TrimSpace(req.Notes)
	prop.ImageURL = strings.TrimSpace(req.ImageURL)
	prop.UpdatedAt = time.Now()

	if err := config.DB.Save(&prop).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memperbarui properti: " + err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Data properti berhasil diperbarui",
		"data":    prop,
	})
}

// DeleteProperty handles DELETE /api/properties/:id
func DeleteProperty(c *fiber.Ctx) error {
	id := c.Params("id")
	companyID := middleware.GetCompanyID(c)

	var prop models.Property
	query := config.DB.Where("id = ?", id)
	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.First(&prop).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Data properti tidak ditemukan"})
	}

	// Hapus relasi template di model & pick list jika ada
	config.DB.Where("property_id = ?", id).Delete(&models.DecorationModelProperty{})
	config.DB.Where("property_id = ?", id).Delete(&models.AgendaProperty{})

	if err := config.DB.Delete(&prop).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus properti: " + err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Master properti berhasil dihapus"})
}

// GetPropertiesAvailability handles GET /api/properties/availability
// Query: date=2026-10-28 & exclude_agenda_id=...
func GetPropertiesAvailability(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	dateStr := strings.TrimSpace(c.Query("date"))
	excludeAgendaID := strings.TrimSpace(c.Query("exclude_agenda_id"))

	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	targetDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format parameter date tidak valid (harus YYYY-MM-DD)"})
	}

	// Range awal dan akhir hari target (00:00:00 s/d 23:59:59)
	startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, time.Local)
	endOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 23, 59, 59, 999999999, time.Local)

	// Ambil semua master properti untuk company ini
	var properties []models.Property
	propQuery := config.DB.Model(&models.Property{})
	if companyID != "" {
		propQuery = propQuery.Where("company_id = ?", companyID)
	}
	if err := propQuery.Order("type ASC, sort_order ASC, name ASC").Find(&properties).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data properti: " + err.Error()})
	}

	// Cari total booked Qty per properti pada tanggal tersebut
	// Hanya menghitung agenda aktif (status != 'cancelled')
	type BookedResult struct {
		PropertyID string `gorm:"column:property_id"`
		TotalQty   int    `gorm:"column:total_qty"`
	}

	var bookedList []BookedResult
	agendaSubQuery := config.DB.Table("agendas").
		Select("id").
		Where("event_date_time >= ? AND event_date_time <= ? AND status != 'cancelled'", startOfDay, endOfDay)

	if companyID != "" {
		agendaSubQuery = agendaSubQuery.Where("company_id = ?", companyID)
	}
	if excludeAgendaID != "" {
		agendaSubQuery = agendaSubQuery.Where("id != ?", excludeAgendaID)
	}

	config.DB.Table("agenda_properties").
		Select("property_id, SUM(qty) as total_qty").
		Where("agenda_id IN (?)", agendaSubQuery).
		Group("property_id").
		Scan(&bookedList)

	bookedMap := make(map[string]int)
	for _, b := range bookedList {
		bookedMap[b.PropertyID] = b.TotalQty
	}

	// Bentuk response availability
	results := make([]models.PropertyAvailabilityResponse, 0, len(properties))
	for _, p := range properties {
		booked := 0
		if p.TrackStock {
			booked = bookedMap[p.ID]
		}

		available := p.TotalStock - booked
		if !p.TrackStock {
			available = p.TotalStock
			if available < 1 {
				available = 999 // Alat kerja selalu tersedia
			}
		}

		isAvailable := true
		if p.TrackStock && available <= 0 {
			isAvailable = false
		}

		results = append(results, models.PropertyAvailabilityResponse{
			Property:     p,
			BookedQty:    booked,
			AvailableQty: available,
			IsAvailable:  isAvailable,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengecek ketersediaan properti",
		"date":    dateStr,
		"data":    results,
	})
}

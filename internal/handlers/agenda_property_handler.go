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

// GetAgendaProperties handles GET /api/agendas/:id/properties
func GetAgendaProperties(c *fiber.Ctx) error {
	agendaID := c.Params("id")
	companyID := middleware.GetCompanyID(c)

	var agenda models.Agenda
	query := config.DB.Where("id = ?", agendaID)
	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Agenda tidak ditemukan"})
	}

	var items []models.AgendaProperty
	config.DB.Preload("Property").
		Where("agenda_id = ?", agendaID).
		Order("created_at ASC").
		Find(&items)

	// Jika belum ada pick list dan agenda punya ModelID, auto-populate dari template model
	if len(items) == 0 && agenda.ModelID != "" {
		var modelProps []models.DecorationModelProperty
		config.DB.Preload("Property").
			Where("decoration_model_id = ?", agenda.ModelID).
			Find(&modelProps)

		if len(modelProps) > 0 {
			for _, mp := range modelProps {
				newItem := models.AgendaProperty{
					ID:         fmt.Sprintf("ap_%s", uuid.New().String()[:8]),
					CompanyID:  agenda.CompanyID,
					AgendaID:   agenda.ID,
					PropertyID: mp.PropertyID,
					Qty:        mp.DefaultQty,
					Notes:      mp.Notes,
					IsPacked:   false,
					IsReturned: false,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}
				config.DB.Create(&newItem)
			}
			// Refresh items
			config.DB.Preload("Property").
				Where("agenda_id = ?", agendaID).
				Order("created_at ASC").
				Find(&items)
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":   "Berhasil mengambil pick list agenda",
		"agenda_id": agendaID,
		"data":      items,
	})
}

// PopulateAgendaPropertiesRequest payload
type PopulateAgendaPropertiesRequest struct {
	ModelID string `json:"model_id"`
	Mode    string `json:"mode"` // "replace" (hapus lama) atau "merge" (gabungkan)
}

// PopulateAgendaProperties handles POST /api/agendas/:id/properties/populate
func PopulateAgendaProperties(c *fiber.Ctx) error {
	agendaID := c.Params("id")
	companyID := middleware.GetCompanyID(c)

	var agenda models.Agenda
	query := config.DB.Where("id = ?", agendaID)
	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Agenda tidak ditemukan"})
	}

	var req PopulateAgendaPropertiesRequest
	if err := c.BodyParser(&req); err != nil {
		req.ModelID = agenda.ModelID
		req.Mode = "replace"
	}

	targetModelID := strings.TrimSpace(req.ModelID)
	if targetModelID == "" {
		targetModelID = agenda.ModelID
	}
	if targetModelID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Model dekorasi belum dipilih"})
	}

	// Ambil template properti dari model
	var modelProps []models.DecorationModelProperty
	if err := config.DB.Preload("Property").
		Where("decoration_model_id = ?", targetModelID).
		Find(&modelProps).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil template properti model: " + err.Error()})
	}

	if strings.ToLower(req.Mode) == "replace" {
		config.DB.Where("agenda_id = ?", agendaID).Delete(&models.AgendaProperty{})
	}

	var warnings []string
	for _, mp := range modelProps {
		// Validasi stok jika properti berstatus TrackStock
		if mp.Property != nil && mp.Property.TrackStock {
			// Cek ketersediaan di tanggal agenda
			startOfDay := time.Date(agenda.EventDateTime.Year(), agenda.EventDateTime.Month(), agenda.EventDateTime.Day(), 0, 0, 0, 0, time.Local)
			endOfDay := time.Date(agenda.EventDateTime.Year(), agenda.EventDateTime.Month(), agenda.EventDateTime.Day(), 23, 59, 59, 999999999, time.Local)

			var bookedQty int64
			config.DB.Table("agenda_properties").
				Joins("JOIN agendas ON agendas.id = agenda_properties.agenda_id").
				Where("agenda_properties.property_id = ? AND agendas.id != ? AND agendas.event_date_time >= ? AND agendas.event_date_time <= ? AND agendas.status != 'cancelled'",
					mp.PropertyID, agendaID, startOfDay, endOfDay).
				Select("COALESCE(SUM(agenda_properties.qty), 0)").
				Scan(&bookedQty)

			available := mp.Property.TotalStock - int(bookedQty)
			if available < mp.DefaultQty {
				warnings = append(warnings, fmt.Sprintf("Stok '%s' kurang (tersedia %d, butuh %d)", mp.Property.Name, available, mp.DefaultQty))
			}
		}

		// Masukkan ke agenda_properties
		item := models.AgendaProperty{
			ID:         fmt.Sprintf("ap_%s", uuid.New().String()[:8]),
			CompanyID:  agenda.CompanyID,
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

	var updatedItems []models.AgendaProperty
	config.DB.Preload("Property").Where("agenda_id = ?", agendaID).Find(&updatedItems)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":  "Pick list berhasil diperbarui dari template model",
		"warnings": warnings,
		"data":     updatedItems,
	})
}

// AddAgendaProperty handles POST /api/agendas/:id/properties
func AddAgendaProperty(c *fiber.Ctx) error {
	agendaID := c.Params("id")
	companyID := middleware.GetCompanyID(c)

	var agenda models.Agenda
	query := config.DB.Where("id = ?", agendaID)
	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}
	if err := query.First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Agenda tidak ditemukan"})
	}

	var req models.AgendaPropertyInput
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Qty <= 0 {
		req.Qty = 1
	}

	item := models.AgendaProperty{
		ID:         fmt.Sprintf("ap_%s", uuid.New().String()[:8]),
		CompanyID:  agenda.CompanyID,
		AgendaID:   agenda.ID,
		PropertyID: req.PropertyID,
		CustomName: strings.TrimSpace(req.CustomName),
		Qty:        req.Qty,
		Notes:      strings.TrimSpace(req.Notes),
		IsPacked:   req.IsPacked,
		IsReturned: req.IsReturned,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := config.DB.Create(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menambahkan item: " + err.Error()})
	}

	config.DB.Preload("Property").Where("id = ?", item.ID).First(&item)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Item berhasil ditambahkan ke pick list",
		"data":    item,
	})
}

// ToggleAgendaPropertyPacked handles PATCH /api/agendas/:id/properties/:item_id/pack
func ToggleAgendaPropertyPacked(c *fiber.Ctx) error {
	agendaID := c.Params("id")
	itemID := c.Params("item_id")

	var item models.AgendaProperty
	if err := config.DB.Where("id = ? AND agenda_id = ?", itemID, agendaID).First(&item).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Item pick list tidak ditemukan"})
	}

	item.IsPacked = !item.IsPacked
	item.UpdatedAt = time.Now()
	config.DB.Save(&item)

	config.DB.Preload("Property").Where("id = ?", item.ID).First(&item)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Status muat (berangkat) berhasil diperbarui",
		"data":    item,
	})
}

// ToggleAgendaPropertyReturned handles PATCH /api/agendas/:id/properties/:item_id/return
func ToggleAgendaPropertyReturned(c *fiber.Ctx) error {
	agendaID := c.Params("id")
	itemID := c.Params("item_id")

	var item models.AgendaProperty
	if err := config.DB.Where("id = ? AND agenda_id = ?", itemID, agendaID).First(&item).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Item pick list tidak ditemukan"})
	}

	item.IsReturned = !item.IsReturned
	item.UpdatedAt = time.Now()
	config.DB.Save(&item)

	config.DB.Preload("Property").Where("id = ?", item.ID).First(&item)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Status bongkar (pulang) berhasil diperbarui",
		"data":    item,
	})
}

// UpdateAgendaPropertyItem handles PUT /api/agendas/:id/properties/:item_id
func UpdateAgendaPropertyItem(c *fiber.Ctx) error {
	agendaID := c.Params("id")
	itemID := c.Params("item_id")

	var item models.AgendaProperty
	if err := config.DB.Where("id = ? AND agenda_id = ?", itemID, agendaID).First(&item).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Item pick list tidak ditemukan"})
	}

	var req models.AgendaPropertyInput
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Qty > 0 {
		item.Qty = req.Qty
	}
	if req.Notes != "" {
		item.Notes = req.Notes
	}
	if req.CustomName != "" {
		item.CustomName = req.CustomName
	}
	item.IsPacked = req.IsPacked
	item.IsReturned = req.IsReturned
	item.UpdatedAt = time.Now()

	config.DB.Save(&item)
	config.DB.Preload("Property").Where("id = ?", item.ID).First(&item)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Item pick list berhasil diperbarui",
		"data":    item,
	})
}

// DeleteAgendaPropertyItem handles DELETE /api/agendas/:id/properties/:item_id
func DeleteAgendaPropertyItem(c *fiber.Ctx) error {
	agendaID := c.Params("id")
	itemID := c.Params("item_id")

	if err := config.DB.Where("id = ? AND agenda_id = ?", itemID, agendaID).Delete(&models.AgendaProperty{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus item: " + err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Item pick list berhasil dihapus"})
}

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

// ── Master Packages (Paket Dekorasi) ──────────────────────────

// CreatePackage menangani POST /api/packages
func CreatePackage(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	if companyID == "" {
		companyID = "cmp_default"
	}

	var req models.CreateOrUpdatePackageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if strings.TrimSpace(req.Name) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama paket wajib diisi"})
	}

	pkg := models.Package{
		ID:          fmt.Sprintf("pkg_%s", uuid.New().String()[:8]),
		CompanyID:   companyID,
		Name:        strings.TrimSpace(req.Name),
		ImageURL:    strings.TrimSpace(req.ImageURL),
		BasePrice:   req.BasePrice,
		Description: strings.TrimSpace(req.Description),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := config.DB.Create(&pkg).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan paket: " + err.Error()})
	}

	// Simpan relasi model dekorasi dalam paket jika ada
	if len(req.PackageModels) > 0 {
		for idx, pm := range req.PackageModels {
			if strings.TrimSpace(pm.ModelID) != "" {
				pkgModel := models.PackageDecorationModel{
					ID:                fmt.Sprintf("pm_%s", uuid.New().String()[:8]),
					CompanyID:         companyID,
					PackageID:         pkg.ID,
					DecorationModelID: strings.TrimSpace(pm.ModelID),
					AdditionalPrice:   pm.AdditionalPrice,
					SortOrder:         idx + 1,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}
				config.DB.Create(&pkgModel)
			}
		}
	}

	// Preload relasi model dekorasi untuk response
	config.DB.Preload("PackageModels.DecorationModel").Where("id = ?", pkg.ID).First(&pkg)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Paket dekorasi berhasil dibuat",
		"data":    pkg,
	})
}

// UpdatePackage menangani PUT /api/packages/:id
func UpdatePackage(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	id := c.Params("id")
	var pkg models.Package

	query := config.DB.Where("id = ?", id)
	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	if err := query.First(&pkg).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Paket tidak ditemukan"})
	}

	var req models.CreateOrUpdatePackageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if strings.TrimSpace(req.Name) != "" {
		pkg.Name = strings.TrimSpace(req.Name)
	}
	pkg.ImageURL = strings.TrimSpace(req.ImageURL)
	pkg.BasePrice = req.BasePrice
	pkg.Description = strings.TrimSpace(req.Description)
	pkg.UpdatedAt = time.Now()

	if err := config.DB.Save(&pkg).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal meng-update paket"})
	}

	// Update relasi model dekorasi jika package_models dikirim
	if req.PackageModels != nil {
		config.DB.Where("package_id = ?", pkg.ID).Delete(&models.PackageDecorationModel{})
		for idx, pm := range req.PackageModels {
			if strings.TrimSpace(pm.ModelID) != "" {
				pkgModel := models.PackageDecorationModel{
					ID:                fmt.Sprintf("pm_%s", uuid.New().String()[:8]),
					CompanyID:         pkg.CompanyID,
					PackageID:         pkg.ID,
					DecorationModelID: strings.TrimSpace(pm.ModelID),
					AdditionalPrice:   pm.AdditionalPrice,
					SortOrder:         idx + 1,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}
				config.DB.Create(&pkgModel)
			}
		}
	}

	// Preload relasi model dekorasi untuk response
	config.DB.Preload("PackageModels.DecorationModel").Where("id = ?", pkg.ID).First(&pkg)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Paket dekorasi berhasil diperbarui",
		"data":    pkg,
	})
}

// DeletePackage menangani DELETE /api/packages/:id
func DeletePackage(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	id := c.Params("id")

	query := config.DB.Where("id = ?", id)
	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	// Hapus relasi package_models terlebih dahulu
	config.DB.Where("package_id = ?", id).Delete(&models.PackageDecorationModel{})

	if err := query.Delete(&models.Package{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus paket"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Paket dekorasi berhasil dihapus"})
}

// ── Master Addons (Item Tambahan) ─────────────────────────────

// GetAddons menangani GET /api/addons
func GetAddons(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	var addons []models.Addon

	query := config.DB.Model(&models.Addon{})
	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	if err := query.Order("sort_order ASC, created_at ASC").Find(&addons).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data addon"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil data tambahan (addon)",
		"data":    addons,
	})
}

// ReorderAddons menangani PUT /api/addons/reorder
func ReorderAddons(c *fiber.Ctx) error {
	var req models.ReorderAddonsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	for idx, id := range req.AddonIDs {
		config.DB.Model(&models.Addon{}).Where("id = ?", id).Update("sort_order", idx)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Urutan item tambahan berhasil diperbarui",
	})
}

// CreateAddon menangani POST /api/addons
func CreateAddon(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	if companyID == "" {
		companyID = "cmp_default"
	}

	var req models.CreateOrUpdateAddonRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if strings.TrimSpace(req.Name) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama item tambahan wajib diisi"})
	}

	addon := models.Addon{
		ID:          fmt.Sprintf("adn_%s", uuid.New().String()[:8]),
		CompanyID:   companyID,
		Name:        strings.TrimSpace(req.Name),
		ImageURL:    strings.TrimSpace(req.ImageURL),
		Price:       req.Price,
		Description: strings.TrimSpace(req.Description),
	}

	if err := config.DB.Create(&addon).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan item tambahan: " + err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Item tambahan berhasil dibuat",
		"data":    addon,
	})
}

// UpdateAddon menangani PUT /api/addons/:id
func UpdateAddon(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	id := c.Params("id")
	var addon models.Addon

	query := config.DB.Where("id = ?", id)
	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	if err := query.First(&addon).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Item tambahan tidak ditemukan"})
	}

	var req models.CreateOrUpdateAddonRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if strings.TrimSpace(req.Name) != "" {
		addon.Name = strings.TrimSpace(req.Name)
	}
	addon.ImageURL = strings.TrimSpace(req.ImageURL)
	addon.Price = req.Price
	addon.Description = strings.TrimSpace(req.Description)

	if err := config.DB.Save(&addon).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal meng-update item tambahan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Item tambahan berhasil diperbarui",
		"data":    addon,
	})
}

// DeleteAddon menangani DELETE /api/addons/:id
func DeleteAddon(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	id := c.Params("id")

	query := config.DB.Where("id = ?", id)
	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	if err := query.Delete(&models.Addon{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus item tambahan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Item tambahan berhasil dihapus"})
}

// ── Master Decoration Models (Model Dekorasi) ─────────────────

// GetDecorationModels menangani GET /api/decoration-models
func GetDecorationModels(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	var modelsList []models.DecorationModel

	query := config.DB.Model(&models.DecorationModel{})
	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	if err := query.Order("sort_order ASC, created_at ASC").Find(&modelsList).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data model dekorasi"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil data model dekorasi",
		"data":    modelsList,
	})
}

// ReorderDecorationModels menangani PUT /api/decoration-models/reorder
func ReorderDecorationModels(c *fiber.Ctx) error {
	var req models.ReorderDecorationModelsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	for idx, id := range req.ModelIDs {
		config.DB.Model(&models.DecorationModel{}).Where("id = ?", id).Update("sort_order", idx+1)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Urutan model dekorasi berhasil diperbarui",
	})
}

// CreateDecorationModel menangani POST /api/decoration-models
func CreateDecorationModel(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	if companyID == "" {
		companyID = "cmp_default"
	}

	var req models.CreateOrUpdateDecorationModelRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if strings.TrimSpace(req.Name) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama model dekorasi wajib diisi"})
	}

	modelItem := models.DecorationModel{
		ID:          fmt.Sprintf("mdl_%s", uuid.New().String()[:8]),
		CompanyID:   companyID,
		Name:        strings.TrimSpace(req.Name),
		ImageURL:    strings.TrimSpace(req.ImageURL),
		Price:       req.Price,
		Width:       strings.TrimSpace(req.Width),
		Description: strings.TrimSpace(req.Description),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := config.DB.Create(&modelItem).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan model dekorasi: " + err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Model dekorasi berhasil dibuat",
		"data":    modelItem,
	})
}

// UpdateDecorationModel menangani PUT /api/decoration-models/:id
func UpdateDecorationModel(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	id := c.Params("id")
	var modelItem models.DecorationModel

	query := config.DB.Where("id = ?", id)
	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	if err := query.First(&modelItem).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Model dekorasi tidak ditemukan"})
	}

	var req models.CreateOrUpdateDecorationModelRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if strings.TrimSpace(req.Name) != "" {
		modelItem.Name = strings.TrimSpace(req.Name)
	}
	modelItem.ImageURL = strings.TrimSpace(req.ImageURL)
	modelItem.Price = req.Price
	modelItem.Width = strings.TrimSpace(req.Width)
	modelItem.Description = strings.TrimSpace(req.Description)
	modelItem.UpdatedAt = time.Now()

	if err := config.DB.Save(&modelItem).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal meng-update model dekorasi"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Model dekorasi berhasil diperbarui",
		"data":    modelItem,
	})
}

// DeleteDecorationModel menangani DELETE /api/decoration-models/:id
func DeleteDecorationModel(c *fiber.Ctx) error {
	companyID := middleware.GetCompanyID(c)
	id := c.Params("id")

	query := config.DB.Where("id = ?", id)
	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}

	// Hapus relasi di package_models terlebih dahulu
	config.DB.Where("decoration_model_id = ?", id).Delete(&models.PackageDecorationModel{})

	if err := query.Delete(&models.DecorationModel{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus model dekorasi"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Model dekorasi berhasil dihapus"})
}

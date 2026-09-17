package handlers

import (
	"fmt"
	"strings"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ── Master Packages (Paket Dekorasi) ──────────────────────────

// CreatePackage menangani POST /api/packages
func CreatePackage(c *fiber.Ctx) error {
	var req models.CreateOrUpdatePackageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if strings.TrimSpace(req.Name) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama paket wajib diisi"})
	}

	pkg := models.Package{
		ID:          fmt.Sprintf("pkg_%s", uuid.New().String()[:8]),
		Name:        strings.TrimSpace(req.Name),
		ImageURL:    strings.TrimSpace(req.ImageURL),
		BasePrice:   req.BasePrice,
		Description: strings.TrimSpace(req.Description),
	}

	if err := config.DB.Create(&pkg).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan paket: " + err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Paket dekorasi berhasil dibuat",
		"data":    pkg,
	})
}

// UpdatePackage menangani PUT /api/packages/:id
func UpdatePackage(c *fiber.Ctx) error {
	id := c.Params("id")
	var pkg models.Package

	if err := config.DB.Where("id = ?", id).First(&pkg).Error; err != nil {
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

	if err := config.DB.Save(&pkg).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal meng-update paket"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Paket dekorasi berhasil diperbarui",
		"data":    pkg,
	})
}

// DeletePackage menangani DELETE /api/packages/:id
func DeletePackage(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := config.DB.Where("id = ?", id).Delete(&models.Package{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus paket"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Paket dekorasi berhasil dihapus"})
}

// ── Master Addons (Item Tambahan) ─────────────────────────────

// GetAddons menangani GET /api/addons
func GetAddons(c *fiber.Ctx) error {
	var addons []models.Addon
	if err := config.DB.Order("price ASC").Find(&addons).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data addon"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil data tambahan (addon)",
		"data":    addons,
	})
}

// CreateAddon menangani POST /api/addons
func CreateAddon(c *fiber.Ctx) error {
	var req models.CreateOrUpdateAddonRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if strings.TrimSpace(req.Name) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama item tambahan wajib diisi"})
	}

	addon := models.Addon{
		ID:          fmt.Sprintf("adn_%s", uuid.New().String()[:8]),
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
	id := c.Params("id")
	var addon models.Addon

	if err := config.DB.Where("id = ?", id).First(&addon).Error; err != nil {
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
	id := c.Params("id")
	if err := config.DB.Where("id = ?", id).Delete(&models.Addon{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menghapus item tambahan"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Item tambahan berhasil dihapus"})
}

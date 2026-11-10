package handlers

import (
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
)

const (
	DefaultInvoiceTemplate        = "template_modern_navy"
	InvoiceTemplateModernNavy     = "template_modern_navy"
	InvoiceTemplateMinimalistGrey = "template_minimalist_grey"
)

// NormalizeInvoiceTemplate memvalidasi ID template invoice yang didukung sistem.
func NormalizeInvoiceTemplate(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	switch trimmed {
	case InvoiceTemplateModernNavy, InvoiceTemplateMinimalistGrey:
		return trimmed, true
	default:
		return DefaultInvoiceTemplate, false
	}
}

// EffectiveInvoiceTemplate mengembalikan template invoice valid atau default jika kosong/tidak dikenal.
func EffectiveInvoiceTemplate(raw string) string {
	norm, ok := NormalizeInvoiceTemplate(raw)
	if !ok {
		return DefaultInvoiceTemplate
	}
	return norm
}

// resolveBrandInvoiceTemplate menentukan template efektif sub-brand dan apakah nilainya mewarisi Brand Utama.
func resolveBrandInvoiceTemplate(companyTemplate string, brandTemplate string) (string, bool) {
	effectiveCompany := EffectiveInvoiceTemplate(companyTemplate)
	trimmedBrand := strings.TrimSpace(brandTemplate)
	if trimmedBrand == "" {
		return effectiveCompany, true
	}
	if norm, ok := NormalizeInvoiceTemplate(trimmedBrand); ok {
		return norm, false
	}
	return effectiveCompany, true
}

// buildBrandTemplatesSummary membangun peta template efektif per brand serta daftar brand yang masih mewarisi Brand Utama.
func buildBrandTemplatesSummary(companyTemplate string, brands []models.CompanyBrand) (map[string]string, []string) {
	effectiveCompany := EffectiveInvoiceTemplate(companyTemplate)
	brandTemplates := map[string]string{
		"": effectiveCompany,
	}
	inheritedBrands := make([]string, 0)

	for _, b := range brands {
		eff, isInherited := resolveBrandInvoiceTemplate(effectiveCompany, b.InvoiceTemplate)
		brandTemplates[b.ID] = eff
		if isInherited {
			inheritedBrands = append(inheritedBrands, b.ID)
		}
	}

	return brandTemplates, inheritedBrands
}

func resolveInvoiceTemplateCompanyID(c *fiber.Ctx) string {
	companyID := middleware.GetCompanyID(c)
	userRole, _ := c.Locals("user_role").(string)
	if strings.EqualFold(userRole, "superadmin") {
		if qCid := strings.TrimSpace(c.Query("company_id")); qCid != "" {
			companyID = qCid
		}
		if companyID == "" {
			var firstCompany models.Company
			if err := config.DB.First(&firstCompany).Error; err == nil {
				companyID = firstCompany.ID
			}
		}
	}
	return companyID
}

// GetCompanyInvoiceTemplate handles GET /api/company/invoice-template?brand_id=...
func GetCompanyInvoiceTemplate(c *fiber.Ctx) error {
	companyID := resolveInvoiceTemplateCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Pengguna tidak terikat dengan badan usaha manapun",
		})
	}

	var comp models.Company
	if err := config.DB.Where("id = ?", companyID).First(&comp).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Data profil usaha tidak ditemukan",
		})
	}

	var brands []models.CompanyBrand
	config.DB.Where("company_id = ?", companyID).Order("created_at ASC").Find(&brands)

	brandID := strings.TrimSpace(c.Query("brand_id"))
	effectiveCompany := EffectiveInvoiceTemplate(comp.InvoiceTemplate)
	effectiveTemplate := effectiveCompany
	isInherited := false

	if brandID != "" {
		found := false
		for _, b := range brands {
			if b.ID == brandID {
				effectiveTemplate, isInherited = resolveBrandInvoiceTemplate(effectiveCompany, b.InvoiceTemplate)
				found = true
				break
			}
		}
		if !found {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Brand tidak ditemukan pada perusahaan ini",
			})
		}
	}

	brandTemplates, inheritedBrands := buildBrandTemplatesSummary(effectiveCompany, brands)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil pengaturan template invoice",
		"data": fiber.Map{
			"brand_id":                 brandID,
			"invoice_template":         effectiveTemplate,
			"company_invoice_template": effectiveCompany,
			"is_inherited":             isInherited,
			"brand_templates":          brandTemplates,
			"inherited_brands":         inheritedBrands,
		},
	})
}

// UpdateCompanyInvoiceTemplate handles PUT /api/company/invoice-template (Owner & Admin)
func UpdateCompanyInvoiceTemplate(c *fiber.Ctx) error {
	companyID := resolveInvoiceTemplateCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Pengguna tidak terikat dengan badan usaha manapun",
		})
	}

	var req models.UpdateInvoiceTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	validTemplate, ok := NormalizeInvoiceTemplate(req.InvoiceTemplate)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Pilihan template invoice tidak valid",
		})
	}

	var comp models.Company
	if err := config.DB.Where("id = ?", companyID).First(&comp).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Data profil usaha tidak ditemukan",
		})
	}

	brandID := strings.TrimSpace(req.BrandID)
	now := time.Now()
	userID, _ := c.Locals("user_id").(string)

	if brandID == "" {
		comp.InvoiceTemplate = validTemplate
		comp.UpdatedBy = userID
		var updatingUser models.User
		if err := config.DB.Where("id = ?", userID).First(&updatingUser).Error; err == nil {
			comp.UpdatedByName = updatingUser.FullName
		}
		comp.UpdatedAt = now
		if err := config.DB.Save(&comp).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal menyimpan template invoice: " + err.Error(),
			})
		}
	} else {
		var brand models.CompanyBrand
		if err := config.DB.Where("id = ? AND company_id = ?", brandID, companyID).First(&brand).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Brand tidak ditemukan pada perusahaan ini",
			})
		}
		brand.InvoiceTemplate = validTemplate
		brand.UpdatedAt = now
		if err := config.DB.Save(&brand).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal menyimpan template invoice brand: " + err.Error(),
			})
		}
	}

	var brands []models.CompanyBrand
	config.DB.Where("company_id = ?", companyID).Order("created_at ASC").Find(&brands)

	effectiveCompany := EffectiveInvoiceTemplate(comp.InvoiceTemplate)
	brandTemplates, inheritedBrands := buildBrandTemplatesSummary(effectiveCompany, brands)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Template desain invoice berhasil disimpan",
		"data": fiber.Map{
			"brand_id":                 brandID,
			"invoice_template":         validTemplate,
			"company_invoice_template": effectiveCompany,
			"is_inherited":             false,
			"brand_templates":          brandTemplates,
			"inherited_brands":         inheritedBrands,
		},
	})
}

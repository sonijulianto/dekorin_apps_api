package handlers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetDefaultFormFields mengembalikan daftar field bawaan standar untuk Form Link Client
func GetDefaultFormFields() []models.FormFieldDefinition {
	return []models.FormFieldDefinition{
		{
			Key:         "client_name",
			Label:       "Nama Lengkap Pemesan / Pengantin",
			Type:        "text",
			Placeholder: "Contoh: Rian & Nisa / Ibu Rina",
			IsSystem:    true,
			IsActive:    true,
			IsRequired:  true,
			SortOrder:   1,
		},
		{
			Key:         "client_phone",
			Label:       "Nomor WhatsApp / HP",
			Type:        "text",
			Placeholder: "Contoh: 081234567890",
			IsSystem:    true,
			IsActive:    true,
			IsRequired:  true,
			SortOrder:   2,
		},
		{
			Key:         "backdrop_title",
			Label:       "Tulisan / Nama di Backdrop",
			Type:        "text",
			Placeholder: "Contoh: The Wedding of Rian & Nisa",
			IsSystem:    true,
			IsActive:    true,
			IsRequired:  false,
			SortOrder:   3,
		},
		{
			Key:         "event_date_time",
			Label:       "Tanggal & Jam Acara",
			Type:        "datetime",
			Placeholder: "Pilih tanggal dan jam acara berlangsung",
			IsSystem:    true,
			IsActive:    true,
			IsRequired:  true,
			SortOrder:   4,
		},
		{
			Key:         "event_address",
			Label:       "Alamat Lengkap Lokasi / Gedung Acara",
			Type:        "textarea",
			Placeholder: "Contoh: Gedung Harmoni, Jl. Melati No. 12, Jakarta",
			IsSystem:    true,
			IsActive:    true,
			IsRequired:  true,
			SortOrder:   5,
		},
		{
			Key:         "maps_url",
			Label:       "Link Google Maps Lokasi",
			Type:        "maps",
			Placeholder: "Tempel link Google Maps atau nama gedung",
			IsSystem:    true,
			IsActive:    true,
			IsRequired:  false,
			SortOrder:   6,
		},
		{
			Key:         "package_id",
			Label:       "Pilihan Paket Dekorasi",
			Type:        "package_picker",
			Placeholder: "Pilih paket dekorasi yang diinginkan",
			IsSystem:    true,
			IsActive:    true,
			IsRequired:  false,
			SortOrder:   7,
		},
		{
			Key:         "model_id",
			Label:       "Pilihan Model Dekorasi",
			Type:        "model_picker",
			Placeholder: "Pilih model backdrop / pelaminan",
			IsSystem:    true,
			IsActive:    true,
			IsRequired:  false,
			SortOrder:   8,
		},
		{
			Key:         "addons",
			Label:       "Item Tambahan (Addons)",
			Type:        "addon_picker",
			Placeholder: "Pilih item tambahan opsional jika diperlukan",
			IsSystem:    true,
			IsActive:    true,
			IsRequired:  false,
			SortOrder:   9,
		},
		{
			Key:         "decoration_theme",
			Label:       "Tema Dekorasi",
			Type:        "text",
			Placeholder: "Contoh: Rustic Modern / Klasik Jawa / Minimalis",
			IsSystem:    true,
			IsActive:    false,
			IsRequired:  false,
			SortOrder:   10,
		},
		{
			Key:         "color_preference",
			Label:       "Preferensi Warna Bunga & Kain",
			Type:        "text",
			Placeholder: "Contoh: Sage Green, Cream & Putih",
			IsSystem:    true,
			IsActive:    false,
			IsRequired:  false,
			SortOrder:   11,
		},
		{
			Key:         "reference_photo_url",
			Label:       "Foto Referensi / Denah Lokasi",
			Type:        "image",
			Placeholder: "Unggah foto inspirasi atau kondisi ruangan",
			IsSystem:    true,
			IsActive:    true,
			IsRequired:  false,
			SortOrder:   12,
		},
		{
			Key:         "notes",
			Label:       "Catatan & Request Khusus",
			Type:        "textarea",
			Placeholder: "Contoh: Akses loading barang malam H-1, tema warna bunga, dll.",
			IsSystem:    true,
			IsActive:    true,
			IsRequired:  false,
			SortOrder:   13,
		},
	}
}

func mergeSystemFieldsIfMissing(existing []models.FormFieldDefinition) []models.FormFieldDefinition {
	defaults := GetDefaultFormFields()
	existingMap := make(map[string]bool)
	for _, f := range existing {
		existingMap[f.Key] = true
	}

	result := make([]models.FormFieldDefinition, len(existing))
	copy(result, existing)

	maxOrder := len(result)
	for _, def := range defaults {
		if !existingMap[def.Key] {
			maxOrder++
			def.IsActive = false
			def.IsRequired = false
			def.SortOrder = maxOrder
			result = append(result, def)
		}
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].SortOrder < result[j].SortOrder
	})
	for i := range result {
		result[i].SortOrder = i + 1
	}
	return result
}

// GetEffectiveFormConfig mengambil konfigurasi form untuk Company & Sub-Brand tertentu.
// Mengembalikan (config, isInheritedFromParentOrDefault)
func GetEffectiveFormConfig(companyID string, brandID string) (models.CompanyFormConfig, bool) {
	brandID = strings.TrimSpace(brandID)

	if brandID != "" {
		var brandCfg models.CompanyFormConfig
		if err := config.DB.Where("company_id = ? AND brand_id = ?", companyID, brandID).First(&brandCfg).Error; err == nil {
			brandCfg.Fields = mergeSystemFieldsIfMissing(brandCfg.Fields)
			return brandCfg, false
		}
	}

	// Fallback ke pengaturan Brand Utama perusahaan (brand_id = '')
	var defaultCompanyCfg models.CompanyFormConfig
	if err := config.DB.Where("company_id = ? AND (brand_id = '' OR brand_id IS NULL)", companyID).First(&defaultCompanyCfg).Error; err == nil {
		defaultCompanyCfg.Fields = mergeSystemFieldsIfMissing(defaultCompanyCfg.Fields)
		if brandID != "" {
			defaultCompanyCfg.BrandID = brandID
			return defaultCompanyCfg, true
		}
		return defaultCompanyCfg, false
	}

	// Fallback ke template standar sistem
	return models.CompanyFormConfig{
		CompanyID: companyID,
		BrandID:   brandID,
		ShowPrice: true,
		Fields:    GetDefaultFormFields(),
	}, true
}

func resolveFormConfigCompanyID(c *fiber.Ctx) string {
	companyID := middleware.GetCompanyID(c)
	userRole, _ := c.Locals("user_role").(string)
	if strings.EqualFold(userRole, "superadmin") {
		if qCid := strings.TrimSpace(c.Query("company_id")); qCid != "" {
			companyID = qCid
		}
	}
	if companyID == "" {
		var firstCompany models.Company
		if err := config.DB.First(&firstCompany).Error; err == nil {
			companyID = firstCompany.ID
		}
	}
	return companyID
}

// GetCompanyFormConfig handles GET /api/company/form-config?brand_id=...
func GetCompanyFormConfig(c *fiber.Ctx) error {
	companyID := resolveFormConfigCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Company ID tidak ditemukan",
		})
	}

	brandID := strings.TrimSpace(c.Query("brand_id"))
	cfg, isInherited := GetEffectiveFormConfig(companyID, brandID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":      "Berhasil mengambil pengaturan form client",
		"data":         cfg,
		"is_inherited": isInherited,
	})
}

// UpdateCompanyFormConfig handles PUT /api/company/form-config
func UpdateCompanyFormConfig(c *fiber.Ctx) error {
	companyID := resolveFormConfigCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Company ID tidak ditemukan",
		})
	}

	var req models.UpdateCompanyFormConfigRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format data tidak valid",
		})
	}

	brandID := strings.TrimSpace(req.BrandID)

	// Validasi & sanitasi daftar field
	systemKeys := map[string]bool{}
	for _, df := range GetDefaultFormFields() {
		systemKeys[df.Key] = true
	}

	allowedTypes := map[string]bool{
		"text": true, "textarea": true, "number": true, "date": true,
		"time": true, "datetime": true, "select": true, "radio": true,
		"checkbox": true, "image": true, "maps": true,
		"package_picker": true, "model_picker": true, "addon_picker": true,
	}

	sanitizedFields := make([]models.FormFieldDefinition, 0, len(req.Fields))
	seenKeys := map[string]bool{}

	for idx, f := range req.Fields {
		label := strings.TrimSpace(f.Label)
		if label == "" {
			continue
		}

		key := strings.TrimSpace(f.Key)
		if key == "" {
			key = fmt.Sprintf("custom_%s", strings.ReplaceAll(uuid.New().String()[:8], "-", ""))
		}
		if seenKeys[key] {
			key = fmt.Sprintf("%s_%d", key, idx+1)
		}
		seenKeys[key] = true

		fieldType := strings.ToLower(strings.TrimSpace(f.Type))
		if !allowedTypes[fieldType] {
			fieldType = "text"
		}

		isActive := f.IsActive
		isRequired := f.IsRequired
		if !isActive {
			isRequired = false
		}

		var cleanOpts []string
		for _, opt := range f.Options {
			trimmed := strings.TrimSpace(opt)
			if trimmed != "" {
				cleanOpts = append(cleanOpts, trimmed)
			}
		}

		sanitizedFields = append(sanitizedFields, models.FormFieldDefinition{
			Key:         key,
			Label:       label,
			Type:        fieldType,
			Placeholder: strings.TrimSpace(f.Placeholder),
			Options:     cleanOpts,
			IsSystem:    systemKeys[key] || f.IsSystem,
			IsActive:    isActive,
			IsRequired:  isRequired,
			SortOrder:   len(sanitizedFields) + 1,
		})
	}

	if len(sanitizedFields) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Minimal harus ada 1 field isian dalam form",
		})
	}

	fieldsBytes, err := json.Marshal(sanitizedFields)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memproses format konfigurasi form",
		})
	}

	showPrice := true
	if req.ShowPrice != nil {
		showPrice = *req.ShowPrice
	}

	var existing models.CompanyFormConfig
	err = config.DB.Where("company_id = ? AND brand_id = ?", companyID, brandID).First(&existing).Error
	if err != nil {
		existing = models.CompanyFormConfig{
			ID:         fmt.Sprintf("fcfg_%s", uuid.New().String()[:8]),
			CompanyID:  companyID,
			BrandID:    brandID,
			ShowPrice:  showPrice,
			FieldsJSON: string(fieldsBytes),
			Fields:     sanitizedFields,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if err := config.DB.Create(&existing).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal menyimpan pengaturan form client",
			})
		}
	} else {
		existing.ShowPrice = showPrice
		existing.FieldsJSON = string(fieldsBytes)
		existing.Fields = sanitizedFields
		existing.UpdatedAt = time.Now()
		if err := config.DB.Save(&existing).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal memperbarui pengaturan form client",
			})
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Pengaturan form client berhasil disimpan",
		"data":    existing,
	})
}

// ResetCompanyFormConfig handles POST /api/company/form-config/reset?brand_id=...
func ResetCompanyFormConfig(c *fiber.Ctx) error {
	companyID := resolveFormConfigCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Company ID tidak ditemukan",
		})
	}

	brandID := strings.TrimSpace(c.Query("brand_id"))
	if err := config.DB.Where("company_id = ? AND brand_id = ?", companyID, brandID).Delete(&models.CompanyFormConfig{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mereset pengaturan form client",
		})
	}

	cfg, isInherited := GetEffectiveFormConfig(companyID, brandID)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":      "Pengaturan form client dikembalikan ke setelan default",
		"data":         cfg,
		"is_inherited": isInherited,
	})
}

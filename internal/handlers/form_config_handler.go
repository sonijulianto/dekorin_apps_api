package handlers

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var nonAlphaNumUnderscoreRe = regexp.MustCompile(`[^a-z0-9_]+`)

var builtInEventTypeOrder = []string{"", "wedding", "lamaran", "birthday", "aqiqah"}

var builtInEventTypeLabels = map[string]string{
	"":         "Umum (Default)",
	"wedding":  "Wedding",
	"lamaran":  "Lamaran",
	"birthday": "Ulang Tahun",
	"aqiqah":   "Aqiqah",
}

// NormalizeEventType menormalkan slug jenis acara agar konsisten lintas platform.
func NormalizeEventType(raw string) string {
	cleaned := strings.ToLower(strings.TrimSpace(raw))
	switch cleaned {
	case "", "umum", "default", "general":
		return ""
	case "wedding", "pernikahan", "akad", "resepsi":
		return "wedding"
	case "lamaran", "engagement", "tunangan":
		return "lamaran"
	case "birthday", "ulang_tahun", "ulang tahun", "ultah", "bday":
		return "birthday"
	case "aqiqah", "akikah":
		return "aqiqah"
	default:
		cleaned = strings.ReplaceAll(cleaned, "-", "_")
		cleaned = strings.ReplaceAll(cleaned, " ", "_")
		cleaned = nonAlphaNumUnderscoreRe.ReplaceAllString(cleaned, "")
		cleaned = strings.Trim(cleaned, "_")
		if len(cleaned) > 45 {
			cleaned = strings.TrimRight(cleaned[:45], "_")
		}
		return cleaned
	}
}

// IsBuiltInEventType mengembalikan true jika event_type termasuk preset bawaan sistem.
func IsBuiltInEventType(eventType string) bool {
	_, ok := builtInEventTypeLabels[NormalizeEventType(eventType)]
	return ok
}

// SlugifyCustomEventType membuat slug unik untuk jenis acara custom dari nama labelnya.
func SlugifyCustomEventType(label string) string {
	base := NormalizeEventType(label)
	if base == "" {
		base = "event"
	}
	if !strings.HasPrefix(base, "custom_") {
		base = "custom_" + base
	}
	if len(base) > 45 {
		base = strings.TrimRight(base[:45], "_")
	}
	return base
}

// GetEventTypeLabel mengembalikan label ramah pengguna untuk suatu event_type.
func GetEventTypeLabel(eventType string, fallbackLabel string) string {
	norm := NormalizeEventType(eventType)
	if lbl, ok := builtInEventTypeLabels[norm]; ok {
		return lbl
	}
	if strings.TrimSpace(fallbackLabel) != "" {
		return strings.TrimSpace(fallbackLabel)
	}
	trimmed := strings.TrimPrefix(norm, "custom_")
	parts := strings.Split(trimmed, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	res := strings.TrimSpace(strings.Join(parts, " "))
	if res == "" {
		return "Jenis Acara Khusus"
	}
	return res
}

// ResolveAgendaEventTypeAndLabel menormalkan event_type dan mencari label resminya (termasuk template custom di DB).
func ResolveAgendaEventTypeAndLabel(companyID string, brandID string, rawEventType string, rawEventLabel string) (string, string) {
	norm := NormalizeEventType(rawEventType)
	if norm == "" {
		return "", ""
	}
	if lbl, ok := builtInEventTypeLabels[norm]; ok {
		return norm, lbl
	}
	if strings.TrimSpace(rawEventLabel) != "" {
		return norm, strings.TrimSpace(rawEventLabel)
	}
	if companyID != "" {
		if brandID != "" {
			if cfg, ok := findFormConfigRecord(companyID, brandID, norm); ok && strings.TrimSpace(cfg.EventLabel) != "" {
				return norm, strings.TrimSpace(cfg.EventLabel)
			}
		}
		if cfg, ok := findFormConfigRecord(companyID, "", norm); ok && strings.TrimSpace(cfg.EventLabel) != "" {
			return norm, strings.TrimSpace(cfg.EventLabel)
		}
	}
	return norm, GetEventTypeLabel(norm, "")
}

func whereEventType(query *gorm.DB, eventType string) *gorm.DB {
	norm := NormalizeEventType(eventType)
	if norm == "" {
		return query.Where("(event_type = '' OR event_type IS NULL)")
	}
	return query.Where("event_type = ?", norm)
}

func whereBrandID(query *gorm.DB, brandID string) *gorm.DB {
	b := strings.TrimSpace(brandID)
	if b == "" {
		return query.Where("(brand_id = '' OR brand_id IS NULL)")
	}
	return query.Where("brand_id = ?", b)
}

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

type presetFieldOverride struct {
	Label       string
	Placeholder string
	IsActive    *bool
	IsRequired  *bool
}

func boolPtr(v bool) *bool { return &v }

func buildPresetFields(overrides map[string]presetFieldOverride, extras []models.FormFieldDefinition) []models.FormFieldDefinition {
	base := GetDefaultFormFields()
	result := make([]models.FormFieldDefinition, 0, len(base)+len(extras))
	for _, f := range base {
		if ov, ok := overrides[f.Key]; ok {
			if ov.Label != "" {
				f.Label = ov.Label
			}
			if ov.Placeholder != "" {
				f.Placeholder = ov.Placeholder
			}
			if ov.IsActive != nil {
				f.IsActive = *ov.IsActive
			}
			if ov.IsRequired != nil {
				f.IsRequired = *ov.IsRequired
			}
		}
		result = append(result, f)
	}
	for _, ex := range extras {
		ex.IsSystem = false
		result = append(result, ex)
	}
	for i := range result {
		result[i].SortOrder = i + 1
	}
	return result
}

// GetPresetFormFields mengembalikan template field bawaan sesuai jenis acara (wedding, lamaran, birthday, aqiqah, atau umum).
func GetPresetFormFields(eventType string) []models.FormFieldDefinition {
	switch NormalizeEventType(eventType) {
	case "wedding":
		return buildPresetFields(map[string]presetFieldOverride{
			"client_name":      {Label: "Nama Kedua Pengantin / Pemesan", Placeholder: "Contoh: Rian & Nisa"},
			"backdrop_title":   {Label: "Tulisan / Nama di Pelaminan", Placeholder: "Contoh: The Wedding of Rian & Nisa"},
			"model_id":         {Label: "Pilihan Model Pelaminan", Placeholder: "Pilih model pelaminan yang diinginkan"},
			"decoration_theme": {Label: "Tema Dekorasi Wedding", Placeholder: "Contoh: Adat Jawa / Sunda Siger / Modern Minimalis", IsActive: boolPtr(true)},
			"color_preference": {Label: "Preferensi Warna Bunga & Kain", Placeholder: "Contoh: Putih, Sage Green & Gold", IsActive: boolPtr(true)},
		}, []models.FormFieldDefinition{
			{
				Key:        "wedding_venue_type",
				Label:      "Area Lokasi Acara",
				Type:       "radio",
				Options:    []string{"Indoor / Gedung", "Semi Outdoor", "Outdoor / Tenda"},
				IsActive:   true,
				IsRequired: false,
			},
			{
				Key:        "wedding_ceremony",
				Label:      "Rangkaian Acara",
				Type:       "checkbox",
				Options:    []string{"Akad Nikah / Pemberkatan", "Resepsi", "Temu Manten / Adat"},
				IsActive:   true,
				IsRequired: false,
			},
			{
				Key:         "wedding_loading_time",
				Label:       "Jam Akses Loading Lokasi",
				Type:        "text",
				Placeholder: "Contoh: H-1 Malam pukul 21.00 WIB",
				IsActive:    true,
				IsRequired:  false,
			},
		})
	case "lamaran":
		return buildPresetFields(map[string]presetFieldOverride{
			"client_name":      {Label: "Nama Pasangan / Pemesan", Placeholder: "Contoh: Rian & Nisa"},
			"backdrop_title":   {Label: "Tulisan / Nama di Backdrop Lamaran", Placeholder: "Contoh: The Engagement of Rian & Nisa"},
			"model_id":         {Label: "Pilihan Model Backdrop Lamaran", Placeholder: "Pilih model backdrop lamaran"},
			"decoration_theme": {Label: "Tema Dekorasi Lamaran", Placeholder: "Contoh: Rustic / Minimalis Elegan / Garden", IsActive: boolPtr(true)},
			"color_preference": {Label: "Preferensi Warna Bunga & Kain", Placeholder: "Contoh: Dusty Pink, Cream & Sage", IsActive: boolPtr(true)},
		}, []models.FormFieldDefinition{
			{
				Key:        "lamaran_seating",
				Label:      "Kebutuhan Kursi / Sofa",
				Type:       "select",
				Options:    []string{"2 Kursi Lamaran", "1 Sofa Pasangan", "Lesehan / Tanpa Kursi", "Sesuai Paket"},
				IsActive:   true,
				IsRequired: false,
			},
			{
				Key:         "lamaran_seserahan",
				Label:       "Perkiraan Jumlah Kotak Seserahan",
				Type:        "number",
				Placeholder: "Contoh: 7 kotak (untuk penataan meja/alas)",
				IsActive:    true,
				IsRequired:  false,
			},
			{
				Key:         "lamaran_welcome_sign",
				Label:       "Tulisan Welcome Sign (Jika Ada)",
				Type:        "text",
				Placeholder: "Contoh: Welcome to Our Engagement Rian & Nisa",
				IsActive:    true,
				IsRequired:  false,
			},
		})
	case "birthday":
		return buildPresetFields(map[string]presetFieldOverride{
			"client_name":      {Label: "Nama Pemesan / Orang Tua", Placeholder: "Contoh: Ibu Rina / Kak Budi"},
			"backdrop_title":   {Label: "Tulisan / Nama di Backdrop Ultah", Placeholder: "Contoh: Happy 5th Birthday Kenzo / Sweet 17 Clarissa"},
			"model_id":         {Label: "Pilihan Model Backdrop Ultah", Placeholder: "Pilih model backdrop ulang tahun"},
			"decoration_theme": {Label: "Tema / Karakter Ulang Tahun", Placeholder: "Contoh: Dino / Frozen / Pastel Minimalis / Sweet 17", IsActive: boolPtr(true)},
			"color_preference": {Label: "Preferensi Warna Balon & Dekor", Placeholder: "Contoh: Baby Blue, Putih & Gold", IsActive: boolPtr(true)},
		}, []models.FormFieldDefinition{
			{
				Key:         "bday_celebrant_name",
				Label:       "Nama yang Berulang Tahun",
				Type:        "text",
				Placeholder: "Contoh: Kenzo Aditya",
				IsActive:    true,
				IsRequired:  true,
			},
			{
				Key:         "bday_age",
				Label:       "Ulang Tahun ke-Berapa (Usia)",
				Type:        "number",
				Placeholder: "Contoh: 5 (untuk angka LED / styrofoam)",
				IsActive:    true,
				IsRequired:  true,
			},
			{
				Key:        "bday_balloon_style",
				Label:      "Model / Susunan Balon",
				Type:       "select",
				Options:    []string{"Balon Garland Organik", "Balon Foil Angka + Latex", "Tanpa Balon / Bunga Saja", "Sesuai Paket"},
				IsActive:   true,
				IsRequired: false,
			},
			{
				Key:        "bday_cake_table",
				Label:      "Kebutuhan Meja Kue / Cylinder",
				Type:       "radio",
				Options:    []string{"3 Cylinder Pedestal", "5 Cylinder Pedestal", "1 Meja Kue Utama", "Sudah Ada / Tidak Perlu"},
				IsActive:   true,
				IsRequired: false,
			},
		})
	case "aqiqah":
		return buildPresetFields(map[string]presetFieldOverride{
			"client_name":      {Label: "Nama Orang Tua / Pemesan", Placeholder: "Contoh: Bapak Andi & Ibu Sari"},
			"backdrop_title":   {Label: "Tulisan / Nama di Backdrop Aqiqah", Placeholder: "Contoh: Tasyakuran Aqiqah Baby Arsyil"},
			"model_id":         {Label: "Pilihan Model Backdrop Aqiqah", Placeholder: "Pilih model backdrop aqiqah"},
			"decoration_theme": {Label: "Tema Dekorasi Aqiqah", Placeholder: "Contoh: Islami Minimalis / Pastel / Rustic", IsActive: boolPtr(true)},
			"color_preference": {Label: "Preferensi Warna Kain & Balon/Bunga", Placeholder: "Contoh: Mint, Cream & Putih", IsActive: boolPtr(true)},
		}, []models.FormFieldDefinition{
			{
				Key:         "aqiqah_baby_name",
				Label:       "Nama Lengkap Bayi",
				Type:        "text",
				Placeholder: "Contoh: Muhammad Arsyil Pratama",
				IsActive:    true,
				IsRequired:  true,
			},
			{
				Key:        "aqiqah_baby_gender",
				Label:      "Jenis Kelamin Bayi",
				Type:       "radio",
				Options:    []string{"Laki-laki", "Perempuan"},
				IsActive:   true,
				IsRequired: false,
			},
			{
				Key:        "aqiqah_box_cradle",
				Label:      "Perlengkapan Box Bayi / Ayunan",
				Type:       "radio",
				Options:    []string{"Pakai Box / Keranjang Bayi dari Vendor", "Pakai Ayunan dari Keluarga", "Tanpa Box Bayi"},
				IsActive:   true,
				IsRequired: false,
			},
		})
	default:
		return GetDefaultFormFields()
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

func findFormConfigRecord(companyID string, brandID string, eventType string) (models.CompanyFormConfig, bool) {
	var cfg models.CompanyFormConfig
	q := whereEventType(whereBrandID(config.DB.Where("company_id = ?", companyID), brandID), eventType)
	if err := q.First(&cfg).Error; err == nil {
		cfg.EventType = NormalizeEventType(cfg.EventType)
		cfg.EventLabel = GetEventTypeLabel(cfg.EventType, cfg.EventLabel)
		cfg.Fields = mergeSystemFieldsIfMissing(cfg.Fields)
		return cfg, true
	}
	return models.CompanyFormConfig{}, false
}

// GetEffectiveFormConfig mengambil konfigurasi form untuk Company, Sub-Brand, dan Jenis Acara tertentu.
// Mengembalikan (config, isInheritedFromParentOrDefault)
func GetEffectiveFormConfig(companyID string, brandID string, eventType string) (models.CompanyFormConfig, bool) {
	brandID = strings.TrimSpace(brandID)
	normEvent := NormalizeEventType(eventType)

	// 1. Cek konfigurasi persis untuk (company, brand, eventType)
	if brandID != "" {
		if brandCfg, ok := findFormConfigRecord(companyID, brandID, normEvent); ok {
			return brandCfg, false
		}
	}

	// 2. Cek konfigurasi Brand Utama untuk jenis acara tersebut (company, '', eventType)
	if companyEventCfg, ok := findFormConfigRecord(companyID, "", normEvent); ok {
		if brandID != "" {
			companyEventCfg.BrandID = brandID
			return companyEventCfg, true
		}
		return companyEventCfg, false
	}

	// 3. Jika jenis acara adalah preset bawaan sistem (wedding, lamaran, birthday, aqiqah)
	if normEvent != "" && IsBuiltInEventType(normEvent) {
		inheritedShowPrice := true
		if brandID != "" {
			if brandDefault, ok := findFormConfigRecord(companyID, brandID, ""); ok {
				inheritedShowPrice = brandDefault.ShowPrice
			} else if companyDefault, ok := findFormConfigRecord(companyID, "", ""); ok {
				inheritedShowPrice = companyDefault.ShowPrice
			}
		} else if companyDefault, ok := findFormConfigRecord(companyID, "", ""); ok {
			inheritedShowPrice = companyDefault.ShowPrice
		}

		return models.CompanyFormConfig{
			CompanyID:  companyID,
			BrandID:    brandID,
			EventType:  normEvent,
			EventLabel: GetEventTypeLabel(normEvent, ""),
			ShowPrice:  inheritedShowPrice,
			Fields:     GetPresetFormFields(normEvent),
		}, true
	}

	// 4. Fallback ke default brand / company (event_type = '') jika jenis acara custom belum punya config
	if normEvent != "" {
		if brandID != "" {
			if brandDefault, ok := findFormConfigRecord(companyID, brandID, ""); ok {
				brandDefault.EventType = normEvent
				brandDefault.EventLabel = GetEventTypeLabel(normEvent, "")
				return brandDefault, true
			}
		}
		if companyDefault, ok := findFormConfigRecord(companyID, "", ""); ok {
			companyDefault.BrandID = brandID
			companyDefault.EventType = normEvent
			companyDefault.EventLabel = GetEventTypeLabel(normEvent, "")
			return companyDefault, true
		}
	}

	// 5. Fallback ke template standar sistem
	return models.CompanyFormConfig{
		CompanyID:  companyID,
		BrandID:    brandID,
		EventType:  normEvent,
		EventLabel: GetEventTypeLabel(normEvent, ""),
		ShowPrice:  true,
		Fields:     GetDefaultFormFields(),
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

// GetCompanyFormTemplates handles GET /api/company/form-config/templates?brand_id=...
func GetCompanyFormTemplates(c *fiber.Ctx) error {
	companyID := resolveFormConfigCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Company ID tidak ditemukan",
		})
	}

	brandID := strings.TrimSpace(c.Query("brand_id"))

	var allRows []models.CompanyFormConfig
	config.DB.Where("company_id = ?", companyID).Order("created_at ASC").Find(&allRows)

	brandSavedMap := map[string]models.CompanyFormConfig{}
	companySavedMap := map[string]models.CompanyFormConfig{}
	customOrder := []string{}
	customSeen := map[string]bool{}

	for _, row := range allRows {
		et := NormalizeEventType(row.EventType)
		if row.BrandID == brandID {
			brandSavedMap[et] = row
		}
		if strings.TrimSpace(row.BrandID) == "" {
			companySavedMap[et] = row
		}
		if !IsBuiltInEventType(et) && et != "" && !customSeen[et] {
			// Tampilkan template custom milik company/brand ini
			if strings.TrimSpace(row.BrandID) == "" || row.BrandID == brandID {
				customSeen[et] = true
				customOrder = append(customOrder, et)
			}
		}
	}

	templates := make([]models.FormTemplateSummary, 0, len(builtInEventTypeOrder)+len(customOrder))
	for _, et := range builtInEventTypeOrder {
		_, isBrandSaved := brandSavedMap[et]
		cfg, _ := GetEffectiveFormConfig(companyID, brandID, et)
		templates = append(templates, models.FormTemplateSummary{
			EventType:      et,
			EventLabel:     GetEventTypeLabel(et, ""),
			IsSystemPreset: true,
			IsCustomized:   isBrandSaved,
			ShowPrice:      cfg.ShowPrice,
		})
	}

	for _, et := range customOrder {
		row, hasBrandRow := brandSavedMap[et]
		if !hasBrandRow {
			row = companySavedMap[et]
		}
		templates = append(templates, models.FormTemplateSummary{
			EventType:      et,
			EventLabel:     GetEventTypeLabel(et, row.EventLabel),
			IsSystemPreset: false,
			IsCustomized:   hasBrandRow,
			ShowPrice:      row.ShowPrice,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil daftar template jenis acara",
		"data":    templates,
	})
}

// GetCompanyFormConfig handles GET /api/company/form-config?brand_id=...&event_type=...
func GetCompanyFormConfig(c *fiber.Ctx) error {
	companyID := resolveFormConfigCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Company ID tidak ditemukan",
		})
	}

	brandID := strings.TrimSpace(c.Query("brand_id"))
	eventType := c.Query("event_type")
	cfg, isInherited := GetEffectiveFormConfig(companyID, brandID, eventType)

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
	rawEventType := strings.TrimSpace(req.EventType)
	eventLabel := strings.TrimSpace(req.EventLabel)

	var eventType string
	if rawEventType == "__new__" {
		if eventLabel == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Nama jenis acara wajib diisi",
			})
		}
		eventType = SlugifyCustomEventType(eventLabel)
		var dupCount int64
		whereEventType(config.DB.Model(&models.CompanyFormConfig{}).Where("company_id = ?", companyID), eventType).Count(&dupCount)
		if dupCount > 0 {
			eventType = fmt.Sprintf("%s_%s", eventType, strings.ReplaceAll(uuid.New().String()[:4], "-", ""))
		}
	} else {
		eventType = NormalizeEventType(rawEventType)
	}
	eventLabel = GetEventTypeLabel(eventType, eventLabel)

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

	if len(sanitizedFields) == 0 && rawEventType == "__new__" {
		sanitizedFields = GetDefaultFormFields()
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
	err = whereEventType(whereBrandID(config.DB.Where("company_id = ?", companyID), brandID), eventType).First(&existing).Error
	if err != nil {
		existing = models.CompanyFormConfig{
			ID:         fmt.Sprintf("fcfg_%s", uuid.New().String()[:8]),
			CompanyID:  companyID,
			BrandID:    brandID,
			EventType:  eventType,
			EventLabel: eventLabel,
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
		existing.EventType = eventType
		existing.EventLabel = eventLabel
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

// ResetCompanyFormConfig handles POST /api/company/form-config/reset?brand_id=...&event_type=...
func ResetCompanyFormConfig(c *fiber.Ctx) error {
	companyID := resolveFormConfigCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Company ID tidak ditemukan",
		})
	}

	brandID := strings.TrimSpace(c.Query("brand_id"))
	eventType := NormalizeEventType(c.Query("event_type"))
	if err := whereEventType(whereBrandID(config.DB.Where("company_id = ?", companyID), brandID), eventType).Delete(&models.CompanyFormConfig{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mereset pengaturan form client",
		})
	}

	cfg, isInherited := GetEffectiveFormConfig(companyID, brandID, eventType)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":      "Pengaturan form client dikembalikan ke setelan default",
		"data":         cfg,
		"is_inherited": isInherited,
	})
}

// DeleteCompanyFormConfigTemplate handles DELETE /api/company/form-config?brand_id=...&event_type=...
// Menghapus template custom jenis acara untuk company/brand terkait.
func DeleteCompanyFormConfigTemplate(c *fiber.Ctx) error {
	companyID := resolveFormConfigCompanyID(c)
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Company ID tidak ditemukan",
		})
	}

	brandID := strings.TrimSpace(c.Query("brand_id"))
	eventType := NormalizeEventType(c.Query("event_type"))
	if eventType == "" || IsBuiltInEventType(eventType) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Preset bawaan sistem tidak dapat dihapus, gunakan Reset Default",
		})
	}

	query := whereEventType(config.DB.Where("company_id = ?", companyID), eventType)
	if brandID != "" {
		query = whereBrandID(query, brandID)
	}
	if err := query.Delete(&models.CompanyFormConfig{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menghapus template jenis acara",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Template jenis acara berhasil dihapus",
	})
}

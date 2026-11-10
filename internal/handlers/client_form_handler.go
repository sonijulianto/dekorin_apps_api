package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"
	"dekorin_apps_api/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// BookingDetailViewData adalah data yang dipassing ke halaman web /form/:token (Form Dinamis & Ringkasan Konfirmasi)
type BookingDetailViewData struct {
	CompanyName            string
	CompanyPhone           string
	CompanyPhoneClean      string
	CompanyAddress         string
	CompanyLogo            string
	CompanyInstagram       string
	CompanyBankInfo        string
	ClientName             string
	ClientPhone            string
	BackdropTitle          string
	EventDateFormatted     string
	EventTimeFormatted     string
	EventAddress           string
	MapsURL                string
	PackageName            string
	PackageDesc            string
	PackageImage           string
	PackagePrice           float64
	PackagePriceFormatted  string
	ModelName              string
	ModelWidth             string
	ModelImageURL          string
	ModelAddPrice          float64
	ModelAddPriceFormatted string
	SelectedAddons         string
	AddonsPrice            float64
	AddonsPriceFormatted   string
	TransportFee           float64
	TransportFeeFormatted  string
	TotalCost              float64
	TotalCostFormatted     string
	DpAmount               float64
	DpAmountFormatted      string
	RemainingPayment       float64
	RemainingFormatted     string
	PaymentStatus          string
	PaymentStatusLabel     string
	IsPaidOff              bool
	HasDP                  bool
	Status                 string
	Notes                  string
	ReferencePhotoURL      string
	FormToken              string
	FormStatus             string
	IsLocked               bool
	ShowPrice              bool
	HasValidEventDate      bool
	CustomAnswers          []models.CustomFormAnswer
	FieldsJSON             template.JS
	InitialValuesJSON      template.JS
	PackagesJSON           template.JS
	ModelsJSON             template.JS
	AddonsJSON             template.JS
}

func formatRupiahID(val float64) string {
	s := fmt.Sprintf("%.0f", val)
	n := len(s)
	if n <= 3 {
		return "Rp " + s
	}
	var res []string
	remainder := n % 3
	if remainder > 0 {
		res = append(res, s[:remainder])
	}
	for i := remainder; i < n; i += 3 {
		res = append(res, s[i:i+3])
	}
	return "Rp " + strings.Join(res, ".")
}

func hasValidAgendaDate(t time.Time) bool {
	return !t.IsZero() && t.Year() > 1970
}

func getSystemFieldValue(agenda models.Agenda, key string) string {
	switch key {
	case "client_name":
		name := strings.TrimSpace(agenda.ClientName)
		if agenda.IsFormDraft && strings.EqualFold(name, "Menunggu Isian Client") {
			return ""
		}
		return name
	case "client_phone":
		return strings.TrimSpace(agenda.ClientPhone)
	case "backdrop_title":
		return strings.TrimSpace(agenda.BackdropTitle)
	case "event_date_time":
		if hasValidAgendaDate(agenda.EventDateTime) {
			return agenda.EventDateTime.Format("2006-01-02T15:04")
		}
		return ""
	case "event_address":
		addr := strings.TrimSpace(agenda.EventAddress)
		if addr == "" && agenda.ClientForm != nil {
			addr = strings.TrimSpace(agenda.ClientForm.EventAddress)
		}
		return addr
	case "maps_url":
		return strings.TrimSpace(agenda.MapsURL)
	case "package_id":
		return strings.TrimSpace(agenda.PackageID)
	case "model_id":
		return strings.TrimSpace(agenda.ModelID)
	case "addons":
		if agenda.ClientForm != nil {
			return strings.TrimSpace(agenda.ClientForm.SelectedAddons)
		}
		return ""
	case "decoration_theme":
		if agenda.ClientForm != nil {
			return strings.TrimSpace(agenda.ClientForm.DecorationTheme)
		}
		return ""
	case "color_preference":
		if agenda.ClientForm != nil {
			return strings.TrimSpace(agenda.ClientForm.ColorPreference)
		}
		return ""
	case "reference_photo_url":
		if agenda.ClientForm != nil {
			return strings.TrimSpace(agenda.ClientForm.ReferencePhotoURL)
		}
		return ""
	case "notes":
		n := strings.TrimSpace(agenda.Notes)
		if n == "" && agenda.ClientForm != nil {
			n = strings.TrimSpace(agenda.ClientForm.SpecialRequests)
		}
		return n
	default:
		if agenda.ClientForm != nil {
			for _, ans := range agenda.ClientForm.CustomAnswers {
				if ans.Key == key {
					return strings.TrimSpace(ans.Value)
				}
			}
		}
		return ""
	}
}

// IsClientFormComplete memeriksa apakah client telah mengisi seluruh field wajib (required) dan status sudah filled.
// Mengembalikan (isLocked, missingRequiredLabels)
func IsClientFormComplete(agenda models.Agenda, cfg models.CompanyFormConfig) (bool, []string) {
	if strings.EqualFold(agenda.Status, "completed") || strings.EqualFold(agenda.Status, "cancelled") {
		return true, nil
	}

	var missing []string
	for _, f := range cfg.Fields {
		if !f.IsActive || !f.IsRequired {
			continue
		}
		val := getSystemFieldValue(agenda, f.Key)
		if strings.TrimSpace(val) == "" {
			missing = append(missing, f.Label)
		}
	}

	if strings.EqualFold(agenda.FormStatus, "filled") && len(missing) == 0 {
		return true, nil
	}

	return false, missing
}

// ShowClientForm menangani GET /form/:token (Form Pengisian Dinamis ATAU Ringkasan Konfirmasi Terkunci)
func ShowClientForm(c *fiber.Ctx) error {
	token := strings.TrimSpace(c.Params("token"))
	if token == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Token tidak valid")
	}

	var agenda models.Agenda
	if err := config.DB.Preload("ClientForm").Where("form_token = ?", token).First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Data pesanan / link form tidak ditemukan atau sudah tidak aktif.")
	}

	// Ambil data Vendor / Perusahaan
	var company models.Company
	if agenda.CompanyID != "" {
		config.DB.Where("id = ?", agenda.CompanyID).First(&company)
	}
	if company.Name == "" {
		company.Name = "Dekorin Organizer"
	}

	displayCompanyName := company.Name
	displayCompanyLogo := company.LogoURL
	displayCompanyPhone := company.Phone
	displayCompanyInstagram := company.Instagram

	// Jika agenda memiliki Brand khusus, gunakan identitas Brand
	if agenda.BrandID != "" {
		var brand models.CompanyBrand
		if err := config.DB.Where("id = ?", agenda.BrandID).First(&brand).Error; err == nil {
			if strings.TrimSpace(brand.Name) != "" {
				displayCompanyName = brand.Name
			}
			if strings.TrimSpace(brand.LogoURL) != "" {
				displayCompanyLogo = brand.LogoURL
			}
			if strings.TrimSpace(brand.Phone) != "" {
				displayCompanyPhone = brand.Phone
			}
			if strings.TrimSpace(brand.Instagram) != "" {
				displayCompanyInstagram = brand.Instagram
			}
		} else if strings.TrimSpace(agenda.BrandName) != "" {
			displayCompanyName = agenda.BrandName
			if strings.TrimSpace(agenda.BrandLogo) != "" {
				displayCompanyLogo = agenda.BrandLogo
			}
			if strings.TrimSpace(agenda.BrandPhone) != "" {
				displayCompanyPhone = agenda.BrandPhone
			}
			if strings.TrimSpace(agenda.BrandInstagram) != "" {
				displayCompanyInstagram = agenda.BrandInstagram
			}
		}
	}

	// Bersihkan nomor WhatsApp vendor
	companyPhoneClean := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, displayCompanyPhone)
	if strings.HasPrefix(companyPhoneClean, "0") {
		companyPhoneClean = "62" + companyPhoneClean[1:]
	}

	// Ambil Konfigurasi Form Dinamis sesuai Company & Sub-Brand
	formCfg, _ := GetEffectiveFormConfig(agenda.CompanyID, agenda.BrandID)
	isLocked, _ := IsClientFormComplete(agenda, formCfg)

	// Ambil deskripsi & banner paket jika ada
	packageDesc := ""
	packageImage := ""
	if agenda.PackageID != "" {
		var pkg models.Package
		if err := config.DB.Where("id = ?", agenda.PackageID).First(&pkg).Error; err == nil {
			packageDesc = pkg.Description
			packageImage = pkg.ImageURL
		}
	}

	// Ambil foto & lebar model dekorasi jika ada
	modelImg := agenda.ModelImageURL
	modelWidth := agenda.ModelWidth
	modelName := agenda.ModelName
	if agenda.ModelID != "" {
		var decorModel models.DecorationModel
		if err := config.DB.Where("id = ?", agenda.ModelID).First(&decorModel).Error; err == nil {
			if modelImg == "" {
				modelImg = decorModel.ImageURL
			}
			if modelWidth == "" {
				modelWidth = decorModel.Width
			}
			if modelName == "" {
				modelName = decorModel.Name
			}
		}
	}

	// Format tanggal Indonesia jika tanggal valid
	hasValidDate := hasValidAgendaDate(agenda.EventDateTime)
	dateFormatted := "Belum ditentukan"
	timeFormatted := "-"
	if hasValidDate {
		days := []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
		months := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
		t := agenda.EventDateTime
		dateFormatted = fmt.Sprintf("%s, %02d %s %d", days[t.Weekday()], t.Day(), months[t.Month()], t.Year())
		timeFormatted = fmt.Sprintf("%02d:%02d WIB", t.Hour(), t.Minute())
	}

	// Hitung addons & total keseluruhan
	addonsPrice := 0.0
	selectedAddonsStr := ""
	refPhotoURL := ""
	var customAnswers []models.CustomFormAnswer
	if agenda.ClientForm != nil {
		addonsPrice = agenda.ClientForm.AddonsPrice
		selectedAddonsStr = agenda.ClientForm.SelectedAddons
		refPhotoURL = agenda.ClientForm.ReferencePhotoURL
		customAnswers = agenda.ClientForm.CustomAnswers
	}

	totalCost := agenda.PackagePrice + agenda.ModelAdditionalPrice + addonsPrice + agenda.TransportFee
	dpAmount := agenda.DpAmount
	remainingPayment := totalCost - dpAmount
	if remainingPayment < 0 {
		remainingPayment = 0
	}

	isPaidOff := strings.EqualFold(agenda.PaymentStatus, "paid") || (totalCost > 0 && dpAmount >= totalCost)
	hasDP := dpAmount > 0 && !isPaidOff

	paymentStatusLabel := "Belum Bayar"
	if isPaidOff {
		paymentStatusLabel = "LUNAS"
	} else if hasDP {
		paymentStatusLabel = "DP TERBAYAR"
	}

	// Siapkan data katalog (Paket, Model, Addon) jika Mode Form Aktif
	var packagesList []models.Package
	var modelsList []models.DecorationModel
	var addonsList []models.Addon

	if !isLocked {
		pkgQuery := config.DB.Preload("PackageModels").Where("company_id = ?", agenda.CompanyID)
		if agenda.BrandID != "" {
			pkgQuery = pkgQuery.Where("brand_id = ? OR brand_id = '' OR brand_id IS NULL", agenda.BrandID)
		} else {
			pkgQuery = pkgQuery.Where("brand_id = '' OR brand_id IS NULL")
		}
		pkgQuery.Order("sort_order ASC, created_at ASC").Find(&packagesList)

		modQuery := config.DB.Where("company_id = ?", agenda.CompanyID)
		if agenda.BrandID != "" {
			modQuery = modQuery.Where("brand_id = ? OR brand_id = '' OR brand_id IS NULL", agenda.BrandID)
		} else {
			modQuery = modQuery.Where("brand_id = '' OR brand_id IS NULL")
		}
		modQuery.Order("sort_order ASC, created_at ASC").Find(&modelsList)

		config.DB.Where("company_id = ?", agenda.CompanyID).Order("sort_order ASC, created_at ASC").Find(&addonsList)
	}

	// Siapkan nilai awal untuk setiap field aktif agar pre-filled saat client mengisi
	initialValues := map[string]string{}
	var activeFields []models.FormFieldDefinition
	for _, f := range formCfg.Fields {
		if f.IsActive {
			activeFields = append(activeFields, f)
			initialValues[f.Key] = getSystemFieldValue(agenda, f.Key)
		}
	}

	// Susun list jawaban yang rapi untuk tampilan Ringkasan Konfirmasi (gabungkan kustom answers & system fields ekstra)
	var summaryAnswers []models.CustomFormAnswer
	if isLocked {
		// Map jawaban kustom
		customMap := map[string]models.CustomFormAnswer{}
		for _, ca := range customAnswers {
			customMap[ca.Key] = ca
		}
		coreKeys := map[string]bool{
			"client_name": true, "client_phone": true, "backdrop_title": true,
			"event_date_time": true, "event_address": true, "maps_url": true,
			"package_id": true, "model_id": true, "addons": true, "notes": true,
			"reference_photo_url": true,
		}
		for _, f := range activeFields {
			if coreKeys[f.Key] {
				continue
			}
			val := getSystemFieldValue(agenda, f.Key)
			if val == "" {
				if ca, ok := customMap[f.Key]; ok {
					val = ca.Value
				}
			}
			if strings.TrimSpace(val) != "" {
				summaryAnswers = append(summaryAnswers, models.CustomFormAnswer{
					Key:   f.Key,
					Label: f.Label,
					Type:  f.Type,
					Value: val,
				})
			}
		}
		// Tambahkan jawaban kustom lama jika key tidak ada lagi di activeFields
		seenSummary := map[string]bool{}
		for _, sa := range summaryAnswers {
			seenSummary[sa.Key] = true
		}
		for _, ca := range customAnswers {
			if !coreKeys[ca.Key] && !seenSummary[ca.Key] && strings.TrimSpace(ca.Value) != "" {
				summaryAnswers = append(summaryAnswers, ca)
			}
		}
	}

	fieldsBytes, _ := json.Marshal(activeFields)
	valuesBytes, _ := json.Marshal(initialValues)
	pkgsBytes, _ := json.Marshal(packagesList)
	modsBytes, _ := json.Marshal(modelsList)
	addonsBytes, _ := json.Marshal(addonsList)

	displayClientName := agenda.ClientName
	if agenda.IsFormDraft && strings.EqualFold(displayClientName, "Menunggu Isian Client") {
		displayClientName = ""
	}

	data := BookingDetailViewData{
		CompanyName:            displayCompanyName,
		CompanyPhone:           displayCompanyPhone,
		CompanyPhoneClean:      companyPhoneClean,
		CompanyAddress:         company.Address,
		CompanyLogo:            displayCompanyLogo,
		CompanyInstagram:       displayCompanyInstagram,
		CompanyBankInfo:        company.BankInfo,
		ClientName:             displayClientName,
		ClientPhone:            agenda.ClientPhone,
		BackdropTitle:          agenda.BackdropTitle,
		EventDateFormatted:     dateFormatted,
		EventTimeFormatted:     timeFormatted,
		EventAddress:           getSystemFieldValue(agenda, "event_address"),
		MapsURL:                agenda.MapsURL,
		PackageName:            agenda.PackageName,
		PackageDesc:            packageDesc,
		PackageImage:           packageImage,
		PackagePrice:           agenda.PackagePrice,
		PackagePriceFormatted:  formatRupiahID(agenda.PackagePrice),
		ModelName:              modelName,
		ModelWidth:             modelWidth,
		ModelImageURL:          modelImg,
		ModelAddPrice:          agenda.ModelAdditionalPrice,
		ModelAddPriceFormatted: formatRupiahID(agenda.ModelAdditionalPrice),
		SelectedAddons:         selectedAddonsStr,
		AddonsPrice:            addonsPrice,
		AddonsPriceFormatted:   formatRupiahID(addonsPrice),
		TransportFee:           agenda.TransportFee,
		TransportFeeFormatted:  formatRupiahID(agenda.TransportFee),
		TotalCost:              totalCost,
		TotalCostFormatted:     formatRupiahID(totalCost),
		DpAmount:               dpAmount,
		DpAmountFormatted:      formatRupiahID(dpAmount),
		RemainingPayment:       remainingPayment,
		RemainingFormatted:     formatRupiahID(remainingPayment),
		PaymentStatus:          agenda.PaymentStatus,
		PaymentStatusLabel:     paymentStatusLabel,
		IsPaidOff:              isPaidOff,
		HasDP:                  hasDP,
		Status:                 agenda.Status,
		Notes:                  getSystemFieldValue(agenda, "notes"),
		ReferencePhotoURL:      refPhotoURL,
		FormToken:              agenda.FormToken,
		FormStatus:             agenda.FormStatus,
		IsLocked:               isLocked,
		ShowPrice:              formCfg.ShowPrice,
		HasValidEventDate:      hasValidDate,
		CustomAnswers:          summaryAnswers,
		FieldsJSON:             template.JS(fieldsBytes),
		InitialValuesJSON:      template.JS(valuesBytes),
		PackagesJSON:           template.JS(pkgsBytes),
		ModelsJSON:             template.JS(modsBytes),
		AddonsJSON:             template.JS(addonsBytes),
	}

	tmpl, err := template.ParseFiles("web/templates/client_form.html")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Gagal memuat template: " + err.Error())
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Set("Pragma", "no-cache")
	c.Set("Expires", "0")
	return tmpl.Execute(c.Response().BodyWriter(), data)
}

// UploadClientFormMedia menangani POST /api/client-form/:token/upload (Upload foto referensi/denah oleh client)
func UploadClientFormMedia(c *fiber.Ctx) error {
	token := strings.TrimSpace(c.Params("token"))
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token form tidak valid",
		})
	}

	var agenda models.Agenda
	if err := config.DB.Preload("ClientForm").Where("form_token = ?", token).First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Link form tidak ditemukan",
		})
	}

	formCfg, _ := GetEffectiveFormConfig(agenda.CompanyID, agenda.BrandID)
	if locked, _ := IsClientFormComplete(agenda, formCfg); locked {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Form sudah terkunci sebagai ringkasan konfirmasi",
		})
	}

	file, err := c.FormFile("image")
	if err != nil {
		file, err = c.FormFile("file")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "File gambar wajib diunggah",
			})
		}
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
	}
	if !allowedExts[ext] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format file tidak didukung. Gunakan JPG, PNG, atau WEBP.",
		})
	}

	const maxFileSize = 10 * 1024 * 1024 // 10 MB
	if file.Size > maxFileSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Ukuran gambar maksimal 10 MB",
		})
	}

	filename, err := utils.ProcessAndSaveImage(file, "./uploads", "cform")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memproses gambar: " + err.Error(),
		})
	}

	proto := strings.TrimSpace(c.Get("X-Forwarded-Proto"))
	if proto == "" {
		proto = c.Protocol()
	}
	fileURL := fmt.Sprintf("%s://%s/uploads/%s", proto, c.Hostname(), filename)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Gambar berhasil diunggah",
		"url":     fileURL,
	})
}

// SubmitClientForm menangani POST /api/client-form/:token (Menerima isian form dinamis dari client & mengunci jika lengkap)
func SubmitClientForm(c *fiber.Ctx) error {
	token := strings.TrimSpace(c.Params("token"))
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token form tidak valid",
		})
	}

	var agenda models.Agenda
	if err := config.DB.Preload("ClientForm").Where("form_token = ?", token).First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Agenda tidak ditemukan",
		})
	}

	formCfg, _ := GetEffectiveFormConfig(agenda.CompanyID, agenda.BrandID)

	// Jika permintaan datang dari public client (tanpa Auth token admin) dan form sudah terkunci, tolak perubahan!
	authHeader := strings.TrimSpace(c.Get("Authorization"))
	if authHeader == "" {
		if locked, _ := IsClientFormComplete(agenda, formCfg); locked {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Data pesanan sudah terkonfirmasi dan terkunci. Silakan hubungi admin melalui WhatsApp untuk perubahan atau tambahan.",
			})
		}
	}

	var req models.SubmitClientFormRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format data tidak valid",
		})
	}

	// Gabungkan input nilai ke dalam peta nilai efektif (mengutamakan request terbaru, fallback nilai lama di agenda)
	getEffectiveVal := func(key string) string {
		if req.CustomAnswers != nil {
			if v, ok := req.CustomAnswers[key]; ok {
				return strings.TrimSpace(v)
			}
		}
		switch key {
		case "client_name":
			if strings.TrimSpace(req.ClientName) != "" {
				return strings.TrimSpace(req.ClientName)
			}
		case "client_phone":
			if strings.TrimSpace(req.ClientPhone) != "" {
				return strings.TrimSpace(req.ClientPhone)
			}
		case "backdrop_title":
			if strings.TrimSpace(req.BackdropTitle) != "" {
				return strings.TrimSpace(req.BackdropTitle)
			}
		case "event_date_time":
			if strings.TrimSpace(req.EventDateTime) != "" {
				return strings.TrimSpace(req.EventDateTime)
			}
		case "event_address":
			if strings.TrimSpace(req.EventAddress) != "" {
				return strings.TrimSpace(req.EventAddress)
			}
		case "maps_url":
			if strings.TrimSpace(req.MapsURL) != "" {
				return strings.TrimSpace(req.MapsURL)
			}
		case "package_id":
			if strings.TrimSpace(req.PackageID) != "" {
				return strings.TrimSpace(req.PackageID)
			}
		case "model_id":
			if strings.TrimSpace(req.ModelID) != "" {
				return strings.TrimSpace(req.ModelID)
			}
		case "addons":
			if len(req.AddonItems) > 0 || len(req.AddonIDs) > 0 {
				return "selected"
			}
		case "decoration_theme":
			if strings.TrimSpace(req.DecorationTheme) != "" {
				return strings.TrimSpace(req.DecorationTheme)
			}
		case "color_preference":
			if strings.TrimSpace(req.ColorPreference) != "" {
				return strings.TrimSpace(req.ColorPreference)
			}
		case "reference_photo_url":
			if strings.TrimSpace(req.ReferencePhotoURL) != "" {
				return strings.TrimSpace(req.ReferencePhotoURL)
			}
		case "notes":
			if strings.TrimSpace(req.Notes) != "" {
				return strings.TrimSpace(req.Notes)
			}
			if strings.TrimSpace(req.SpecialRequests) != "" {
				return strings.TrimSpace(req.SpecialRequests)
			}
		}
		return getSystemFieldValue(agenda, key)
	}

	// Validasi seluruh field aktif yang bertanda Required
	var missingRequired []string
	for _, f := range formCfg.Fields {
		if !f.IsActive || !f.IsRequired {
			continue
		}
		if strings.TrimSpace(getEffectiveVal(f.Key)) == "" {
			missingRequired = append(missingRequired, f.Label)
		}
	}
	if len(missingRequired) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":          fmt.Sprintf("Mohon lengkapi kolom wajib: %s", strings.Join(missingRequired, ", ")),
			"missing_fields": missingRequired,
		})
	}

	// Update field-field pokok pada Agenda
	if cn := getEffectiveVal("client_name"); cn != "" {
		agenda.ClientName = cn
	} else if agenda.ClientName == "" {
		agenda.ClientName = "Klien Dekorin"
	}

	if cp := getEffectiveVal("client_phone"); cp != "" {
		agenda.ClientPhone = cp
	}

	if bt := getEffectiveVal("backdrop_title"); bt != "" {
		agenda.BackdropTitle = bt
	} else if agenda.BackdropTitle == "" && agenda.ClientName != "" && agenda.ClientName != "Menunggu Isian Client" {
		agenda.BackdropTitle = fmt.Sprintf("Acara %s", agenda.ClientName)
	}

	if dtStr := getEffectiveVal("event_date_time"); dtStr != "" {
		timeFormats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
			"2006-01-02T15:04",
			"2006-01-02 15:04",
			"2006-01-02",
		}
		for _, format := range timeFormats {
			if t, err := time.Parse(format, dtStr); err == nil {
				agenda.EventDateTime = t
				break
			}
		}
	}

	eventAddress := getEffectiveVal("event_address")
	if eventAddress != "" {
		agenda.EventAddress = eventAddress
	}

	if mapsVal := getEffectiveVal("maps_url"); mapsVal != "" {
		if !strings.HasPrefix(mapsVal, "http://") && !strings.HasPrefix(mapsVal, "https://") {
			mapsVal = "https://www.google.com/maps/search/?api=1&query=" + mapsVal
		}
		agenda.MapsURL = mapsVal
	}

	// Paket Dekorasi
	packageID := getEffectiveVal("package_id")
	packagePrice := agenda.PackagePrice
	packageName := agenda.PackageName
	if packageID != "" {
		var selectedPkg models.Package
		if err := config.DB.Where("id = ?", packageID).First(&selectedPkg).Error; err == nil {
			packagePrice = selectedPkg.BasePrice
			packageName = selectedPkg.Name
			agenda.PackageID = packageID
			agenda.PackageName = packageName
			agenda.PackagePrice = packagePrice
		}
	}

	// Model Dekorasi
	modelID := getEffectiveVal("model_id")
	modelName := agenda.ModelName
	if modelID != "" {
		var decorModel models.DecorationModel
		if err := config.DB.Where("id = ?", modelID).First(&decorModel).Error; err == nil {
			modelName = decorModel.Name
			agenda.ModelID = decorModel.ID
			agenda.ModelName = decorModel.Name
			agenda.ModelWidth = decorModel.Width
			agenda.ModelImageURL = decorModel.ImageURL

			// Cek harga tambahan model di dalam paket jika ada
			var pm models.PackageDecorationModel
			if packageID != "" && config.DB.Where("package_id = ? AND decoration_model_id = ?", packageID, modelID).First(&pm).Error == nil {
				agenda.ModelAdditionalPrice = pm.AdditionalPrice
			}
		}
	}

	// Hitung Addons
	addonsPrice := 0.0
	var selectedAddonNames []string
	if len(req.AddonItems) > 0 {
		var ids []string
		qtyMap := map[string]int{}
		for _, item := range req.AddonItems {
			if item.Quantity > 0 && item.AddonID != "" {
				ids = append(ids, item.AddonID)
				qtyMap[item.AddonID] = item.Quantity
			}
		}
		if len(ids) > 0 {
			var addons []models.Addon
			config.DB.Where("id IN ?", ids).Find(&addons)
			for _, a := range addons {
				qty := qtyMap[a.ID]
				if qty <= 0 {
					qty = 1
				}
				sub := a.Price * float64(qty)
				addonsPrice += sub
				if qty > 1 {
					selectedAddonNames = append(selectedAddonNames, fmt.Sprintf("%s x%d (%s)", a.Name, qty, formatRupiahID(sub)))
				} else {
					selectedAddonNames = append(selectedAddonNames, fmt.Sprintf("%s (%s)", a.Name, formatRupiahID(sub)))
				}
			}
		}
	} else if len(req.AddonIDs) > 0 {
		var addons []models.Addon
		config.DB.Where("id IN ?", req.AddonIDs).Find(&addons)
		for _, a := range addons {
			addonsPrice += a.Price
			selectedAddonNames = append(selectedAddonNames, fmt.Sprintf("%s (%s)", a.Name, formatRupiahID(a.Price)))
		}
	} else if agenda.ClientForm != nil {
		addonsPrice = agenda.ClientForm.AddonsPrice
		if agenda.ClientForm.SelectedAddons != "" {
			selectedAddonNames = []string{agenda.ClientForm.SelectedAddons}
		}
	}

	selectedAddonsStr := strings.Join(selectedAddonNames, ", ")
	totalPrice := packagePrice + addonsPrice

	notesVal := getEffectiveVal("notes")
	if notesVal != "" {
		agenda.Notes = notesVal
	}

	decTheme := getEffectiveVal("decoration_theme")
	colorPref := getEffectiveVal("color_preference")
	refPhoto := getEffectiveVal("reference_photo_url")

	// Susun seluruh jawaban kustom (snapshot label + tipe + value) sesuai sort_order
	var customAnswers []models.CustomFormAnswer
	for _, f := range formCfg.Fields {
		if !f.IsActive {
			continue
		}
		val := getEffectiveVal(f.Key)
		if f.Key == "addons" && selectedAddonsStr != "" {
			val = selectedAddonsStr
		}
		if f.Key == "package_id" && packageName != "" {
			val = packageName
		}
		if f.Key == "model_id" && modelName != "" {
			val = modelName
		}
		if strings.TrimSpace(val) != "" {
			customAnswers = append(customAnswers, models.CustomFormAnswer{
				Key:   f.Key,
				Label: f.Label,
				Type:  f.Type,
				Value: val,
			})
		}
	}
	customAnswersBytes, _ := json.Marshal(customAnswers)

	// Upsert ClientForm
	var existingForm models.ClientForm
	err := config.DB.Where("agenda_id = ?", agenda.ID).First(&existingForm).Error
	if err == nil {
		existingForm.ClientName = agenda.ClientName
		existingForm.ClientPhone = agenda.ClientPhone
		existingForm.BackdropTitle = agenda.BackdropTitle
		existingForm.EventDateTime = getEffectiveVal("event_date_time")
		existingForm.MapsURL = agenda.MapsURL
		existingForm.PackageID = packageID
		existingForm.PackageName = packageName
		existingForm.PackagePrice = packagePrice
		existingForm.ModelID = modelID
		existingForm.ModelName = modelName
		existingForm.SelectedAddons = selectedAddonsStr
		existingForm.AddonsPrice = addonsPrice
		existingForm.TotalPrice = totalPrice
		existingForm.EventAddress = eventAddress
		existingForm.DecorationTheme = decTheme
		existingForm.ColorPreference = colorPref
		existingForm.SpecialRequests = notesVal
		existingForm.ReferencePhotoURL = refPhoto
		existingForm.CustomAnswersJSON = string(customAnswersBytes)
		existingForm.CustomAnswers = customAnswers
		existingForm.SubmittedAt = time.Now()

		if err := config.DB.Save(&existingForm).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal memperbarui data form",
			})
		}
	} else {
		newForm := models.ClientForm{
			ID:                fmt.Sprintf("frm_%s", uuid.New().String()[:8]),
			AgendaID:          agenda.ID,
			ClientName:        agenda.ClientName,
			ClientPhone:       agenda.ClientPhone,
			BackdropTitle:     agenda.BackdropTitle,
			EventDateTime:     getEffectiveVal("event_date_time"),
			MapsURL:           agenda.MapsURL,
			PackageID:         packageID,
			PackageName:       packageName,
			PackagePrice:      packagePrice,
			ModelID:           modelID,
			ModelName:         modelName,
			SelectedAddons:    selectedAddonsStr,
			AddonsPrice:       addonsPrice,
			TotalPrice:        totalPrice,
			EventAddress:      eventAddress,
			DecorationTheme:   decTheme,
			ColorPreference:   colorPref,
			SpecialRequests:   notesVal,
			ReferencePhotoURL: refPhoto,
			CustomAnswersJSON: string(customAnswersBytes),
			CustomAnswers:     customAnswers,
			SubmittedAt:       time.Now(),
		}

		if err := config.DB.Create(&newForm).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal menyimpan data form",
			})
		}
	}

	// Aktifkan ke Agenda utama & kunci form sebagai ringkasan konfirmasi
	agenda.IsFormDraft = false
	agenda.FormStatus = "filled"
	if !hasValidAgendaDate(agenda.EventDateTime) {
		agenda.EventDateTime = time.Now().Add(24 * time.Hour)
	}
	agenda.UpdatedAt = time.Now()
	if err := config.DB.Save(&agenda).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memperbarui status agenda",
		})
	}

	// Auto-populate Pick List jika model dipilih dan belum ada item pick list
	if modelID != "" {
		var propCount int64
		config.DB.Model(&models.AgendaProperty{}).Where("agenda_id = ?", agenda.ID).Count(&propCount)
		if propCount == 0 {
			var modelProps []models.DecorationModelProperty
			config.DB.Where("decoration_model_id = ?", modelID).Find(&modelProps)
			for _, mp := range modelProps {
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
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":     "Form berhasil dikirim dan terkonfirmasi!",
		"total_price": totalPrice,
		"locked":      true,
	})
}

// GetClientFormByAgendaID menangani GET /api/agendas/:id/client-form (Dipanggil dari Flutter & Web)
func GetClientFormByAgendaID(c *fiber.Ctx) error {
	agendaID := c.Params("id")
	companyID := middleware.GetCompanyID(c)
	userRole, _ := c.Locals("user_role").(string)

	// Verifikasi hak akses agenda sesuai tenant
	var agenda models.Agenda
	agendaQuery := config.DB.Where("id = ?", agendaID)
	if !strings.EqualFold(userRole, "superadmin") && companyID != "" {
		agendaQuery = agendaQuery.Where("company_id = ?", companyID)
	}
	if err := agendaQuery.First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Agenda tidak ditemukan atau Anda tidak memiliki akses",
		})
	}

	var clientForm models.ClientForm
	if err := config.DB.Where("agenda_id = ?", agendaID).First(&clientForm).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Data form belum diisi oleh client",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil mengambil data form client",
		"data":    clientForm,
	})
}

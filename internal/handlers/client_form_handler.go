package handlers

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ShowClientForm menangani GET /form/:token (menampilkan Halaman HTML Form Publik)
func ShowClientForm(c *fiber.Ctx) error {
	token := c.Params("token")
	if strings.TrimSpace(token) == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Token tidak valid")
	}

	var agenda models.Agenda
	if err := config.DB.Where("form_token = ?", token).First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Form tidak ditemukan atau tautan sudah kadaluarsa.")
	}

	// Fetch semua paket & addons master dari DB
	var packages []models.Package
	config.DB.Order("base_price ASC").Find(&packages)

	var addons []models.Addon
	config.DB.Order("price ASC").Find(&addons)

	// Format tanggal untuk tampilan
	formattedDate := agenda.EventDateTime.Format("02 January 2006, 15:04 WIB")

	data := struct {
		ClientName             string
		BackdropTitle          string
		EventDateTimeFormatted string
		PackageName            string
		FormStatus             string
		FormToken              string
		Packages               []models.Package
		Addons                 []models.Addon
	}{
		ClientName:             agenda.ClientName,
		BackdropTitle:          agenda.BackdropTitle,
		EventDateTimeFormatted: formattedDate,
		PackageName:            agenda.PackageName,
		FormStatus:             agenda.FormStatus,
		FormToken:              agenda.FormToken,
		Packages:               packages,
		Addons:                 addons,
	}

	tmpl, err := template.ParseFiles("web/templates/client_form.html")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Gagal memuat template: " + err.Error())
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.Execute(c.Response().BodyWriter(), data)
}

// SubmitClientForm menangani POST /api/client-form/:token (Menerima isian form dari client)
func SubmitClientForm(c *fiber.Ctx) error {
	token := c.Params("token")
	if strings.TrimSpace(token) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token form tidak valid",
		})
	}

	var agenda models.Agenda
	if err := config.DB.Where("form_token = ?", token).First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Agenda tidak ditemukan",
		})
	}

	var req models.SubmitClientFormRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format data tidak valid",
		})
	}

	if strings.TrimSpace(req.EventAddress) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Alamat acara wajib diisi",
		})
	}

	// Hitung harga paket pilihan client
	var selectedPkg models.Package
	packagePrice := 0.0
	packageName := agenda.PackageName
	packageID := req.PackageID
	if packageID == "" {
		packageID = agenda.PackageID
	}

	if packageID != "" {
		if err := config.DB.Where("id = ?", packageID).First(&selectedPkg).Error; err == nil {
			packagePrice = selectedPkg.BasePrice
			packageName = selectedPkg.Name
		}
	}

	// Hitung harga item tambahan (Addons)
	addonsPrice := 0.0
	var selectedAddonNames []string
	if len(req.AddonIDs) > 0 {
		var addons []models.Addon
		config.DB.Where("id IN ?", req.AddonIDs).Find(&addons)
		for _, a := range addons {
			addonsPrice += a.Price
			selectedAddonNames = append(selectedAddonNames, fmt.Sprintf("%s (Rp %.0f)", a.Name, a.Price))
		}
	}

	totalPrice := packagePrice + addonsPrice
	selectedAddonsStr := strings.Join(selectedAddonNames, ", ")

	// Cek apakah sudah ada ClientForm untuk Agenda ini
	var existingForm models.ClientForm
	err := config.DB.Where("agenda_id = ?", agenda.ID).First(&existingForm).Error

	if err == nil {
		// Update data form yang sudah ada
		existingForm.PackageID = packageID
		existingForm.PackageName = packageName
		existingForm.PackagePrice = packagePrice
		existingForm.SelectedAddons = selectedAddonsStr
		existingForm.AddonsPrice = addonsPrice
		existingForm.TotalPrice = totalPrice
		existingForm.EventAddress = strings.TrimSpace(req.EventAddress)
		existingForm.DecorationTheme = strings.TrimSpace(req.DecorationTheme)
		existingForm.ColorPreference = strings.TrimSpace(req.ColorPreference)
		existingForm.SpecialRequests = strings.TrimSpace(req.SpecialRequests)
		existingForm.ReferencePhotoURL = strings.TrimSpace(req.ReferencePhotoURL)
		existingForm.SubmittedAt = time.Now()

		if err := config.DB.Save(&existingForm).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal meng-update data form",
			})
		}
	} else {
		// Buat ClientForm baru
		newForm := models.ClientForm{
			ID:                fmt.Sprintf("frm_%s", uuid.New().String()[:8]),
			AgendaID:          agenda.ID,
			PackageID:         packageID,
			PackageName:       packageName,
			PackagePrice:      packagePrice,
			SelectedAddons:    selectedAddonsStr,
			AddonsPrice:       addonsPrice,
			TotalPrice:        totalPrice,
			EventAddress:      strings.TrimSpace(req.EventAddress),
			DecorationTheme:   strings.TrimSpace(req.DecorationTheme),
			ColorPreference:   strings.TrimSpace(req.ColorPreference),
			SpecialRequests:   strings.TrimSpace(req.SpecialRequests),
			ReferencePhotoURL: strings.TrimSpace(req.ReferencePhotoURL),
			SubmittedAt:       time.Now(),
		}

		if err := config.DB.Create(&newForm).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal menyimpan data form",
			})
		}
	}

	// Update status form & paket di Agenda
	agenda.FormStatus = "filled"
	if packageID != "" {
		agenda.PackageID = packageID
		agenda.PackageName = packageName
	}
	config.DB.Save(&agenda)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":     "Form berhasil disimpan",
		"total_price": totalPrice,
	})
}

// GetClientFormByAgendaID menangani GET /api/agendas/:id/client-form (Dipanggil dari Flutter)
func GetClientFormByAgendaID(c *fiber.Ctx) error {
	agendaID := c.Params("id")

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

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

// BookingDetailViewData adalah data yang dipassing ke halaman web invoice/detail pesanan klien
type BookingDetailViewData struct {
	CompanyName            string
	CompanyPhone           string
	CompanyPhoneClean      string
	CompanyAddress         string
	CompanyLogo            string
	ClientName             string
	ClientPhone            string
	BackdropTitle          string
	EventDateFormatted     string
	EventTimeFormatted     string
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
	FormToken              string
}

// ShowClientForm menangani GET /form/:token (menampilkan Halaman Detail & Konfirmasi Booking Klien)
func ShowClientForm(c *fiber.Ctx) error {
	token := c.Params("token")
	if strings.TrimSpace(token) == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Token tidak valid")
	}

	var agenda models.Agenda
	if err := config.DB.Where("form_token = ?", token).First(&agenda).Error; err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Data pesanan / booking tidak ditemukan atau tautan sudah tidak aktif.")
	}

	// Ambil data Vendor / Perusahaan
	var company models.Company
	if agenda.CompanyID != "" {
		config.DB.Where("id = ?", agenda.CompanyID).First(&company)
	}
	if company.Name == "" {
		company.Name = "Dekorin Organizer"
	}

	// Bersihkan nomor WhatsApp vendor
	companyPhoneClean := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, company.Phone)
	if strings.HasPrefix(companyPhoneClean, "0") {
		companyPhoneClean = "62" + companyPhoneClean[1:]
	}

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

	// Helper format Rupiah
	formatRupiah := func(val float64) string {
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

	// Format tanggal Indonesia
	days := []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
	months := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	t := agenda.EventDateTime
	dateFormatted := fmt.Sprintf("%s, %02d %s %d", days[t.Weekday()], t.Day(), months[t.Month()], t.Year())
	timeFormatted := fmt.Sprintf("%02d:%02d WIB", t.Hour(), t.Minute())

	// Hitung total dan sisa pembayaran
	totalCost := agenda.PackagePrice + agenda.ModelAdditionalPrice + agenda.TransportFee
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

	data := BookingDetailViewData{
		CompanyName:            company.Name,
		CompanyPhone:           company.Phone,
		CompanyPhoneClean:      companyPhoneClean,
		CompanyAddress:         company.Address,
		CompanyLogo:            company.LogoURL,
		ClientName:             agenda.ClientName,
		ClientPhone:            agenda.ClientPhone,
		BackdropTitle:          agenda.BackdropTitle,
		EventDateFormatted:     dateFormatted,
		EventTimeFormatted:     timeFormatted,
		MapsURL:                agenda.MapsURL,
		PackageName:            agenda.PackageName,
		PackageDesc:            packageDesc,
		PackageImage:           packageImage,
		PackagePrice:           agenda.PackagePrice,
		PackagePriceFormatted:  formatRupiah(agenda.PackagePrice),
		ModelName:              modelName,
		ModelWidth:             modelWidth,
		ModelImageURL:          modelImg,
		ModelAddPrice:          agenda.ModelAdditionalPrice,
		ModelAddPriceFormatted: formatRupiah(agenda.ModelAdditionalPrice),
		TransportFee:           agenda.TransportFee,
		TransportFeeFormatted:  formatRupiah(agenda.TransportFee),
		TotalCost:              totalCost,
		TotalCostFormatted:     formatRupiah(totalCost),
		DpAmount:               dpAmount,
		DpAmountFormatted:      formatRupiah(dpAmount),
		RemainingPayment:       remainingPayment,
		RemainingFormatted:     formatRupiah(remainingPayment),
		PaymentStatus:          agenda.PaymentStatus,
		PaymentStatusLabel:     paymentStatusLabel,
		IsPaidOff:              isPaidOff,
		HasDP:                  hasDP,
		Status:                 agenda.Status,
		Notes:                  agenda.Notes,
		FormToken:              agenda.FormToken,
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

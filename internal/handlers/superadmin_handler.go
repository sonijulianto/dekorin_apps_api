package handlers

import (
	"fmt"
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"
	"dekorin_apps_api/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// SuperadminLogin handles POST /api/superadmin/login
func SuperadminLogin(c *fiber.Ctx) error {
	req := new(models.LoginRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format data login tidak valid",
		})
	}

	cleanPhone := utils.NormalizePhone(req.Phone)
	identifier := cleanPhone
	isPhone := true

	if identifier == "" && req.Email != "" {
		identifier = strings.TrimSpace(req.Email)
		isPhone = false
	}

	if identifier == "" || strings.TrimSpace(req.Password) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No HP/Email dan password wajib diisi",
		})
	}

	var user models.User
	var err error

	if isPhone {
		err = config.DB.Where("phone = ? AND role = ?", cleanPhone, "superadmin").First(&user).Error
		if err != nil && strings.Contains(req.Phone, "@") {
			err = config.DB.Where("email = ? AND role = ?", strings.TrimSpace(req.Phone), "superadmin").First(&user).Error
		}
	} else {
		err = config.DB.Where("email = ? AND role = ?", identifier, "superadmin").First(&user).Error
	}

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Kredensial Superadmin tidak valid",
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Kredensial Superadmin tidak valid",
		})
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":         user.ID,
		"company_id": "",
		"phone":      user.Phone,
		"email":      user.Email,
		"role":       "superadmin",
		"exp":        time.Now().Add(time.Hour * 24 * 7).Unix(),
	})

	tokenString, err := token.SignedString(middleware.JwtSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal membuat sesi login Superadmin",
		})
	}

	// Set cookie untuk kemudahan Web Superadmin
	c.Cookie(&fiber.Cookie{
		Name:     "superadmin_token",
		Value:    tokenString,
		Expires:  time.Now().Add(time.Hour * 24 * 7),
		HTTPOnly: false,
		SameSite: "Lax",
	})

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Login Superadmin berhasil",
		"token":   tokenString,
		"data": models.UserResponse{
			ID:       user.ID,
			FullName: user.FullName,
			Phone:    user.Phone,
			Email:    user.Email,
			Role:     user.Role,
		},
	})
}

// GetAllCompanies handles GET /api/superadmin/companies
func GetAllCompanies(c *fiber.Ctx) error {
	var companies []models.Company
	if err := config.DB.Order("created_at desc").Find(&companies).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil daftar perusahaan: " + err.Error(),
		})
	}

	var results []models.CompanyDetailResponse
	for _, comp := range companies {
		var totalOwners, totalAdmins, totalCrew, totalAgendas int64

		config.DB.Model(&models.User{}).Where("company_id = ? AND role = ?", comp.ID, "owner").Count(&totalOwners)
		config.DB.Model(&models.User{}).Where("company_id = ? AND role = ?", comp.ID, "admin").Count(&totalAdmins)
		config.DB.Model(&models.User{}).Where("company_id = ? AND role = ?", comp.ID, "crew").Count(&totalCrew)
		config.DB.Model(&models.Agenda{}).Where("company_id = ?", comp.ID).Count(&totalAgendas)

		var owners []models.User
		config.DB.Where("company_id = ? AND role = ?", comp.ID, "owner").Find(&owners)
		var ownersList []string
		for _, o := range owners {
			ownersList = append(ownersList, fmt.Sprintf("%s (%s)", o.FullName, o.Phone))
		}

		results = append(results, models.CompanyDetailResponse{
			ID:           comp.ID,
			Name:         comp.Name,
			Address:      comp.Address,
			Phone:        comp.Phone,
			LogoURL:      comp.LogoURL,
			TotalOwners:  totalOwners,
			TotalAdmins:  totalAdmins,
			TotalCrew:    totalCrew,
			TotalAgendas: totalAgendas,
			OwnersList:   ownersList,
			CreatedAt:    comp.CreatedAt,
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Berhasil memuat daftar perusahaan",
		"data":    results,
	})
}

// CreateCompanyWithInitialOwner handles POST /api/superadmin/companies
func CreateCompanyWithInitialOwner(c *fiber.Ctx) error {
	req := new(models.CreateCompanyRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	req.CompanyName = strings.TrimSpace(req.CompanyName)
	if req.CompanyName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nama Perusahaan / Usaha wajib diisi",
		})
	}

	req.OwnerName = strings.TrimSpace(req.OwnerName)
	if req.OwnerName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nama Owner wajib diisi",
		})
	}

	cleanOwnerPhone := utils.NormalizePhone(req.OwnerPhone)
	if len(cleanOwnerPhone) < 9 || len(cleanOwnerPhone) > 15 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nomor Handphone Owner tidak valid (minimal 9 digit)",
		})
	}

	if len(strings.TrimSpace(req.OwnerPassword)) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password Owner minimal 6 karakter",
		})
	}

	// Cek apakah No HP Owner sudah terdaftar
	var existingUser models.User
	if err := config.DB.Where("phone = ?", cleanOwnerPhone).First(&existingUser).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Nomor Handphone Owner sudah terdaftar pada akun lain",
		})
	}

	// 1. Buat Company Baru
	companyID := fmt.Sprintf("cmp_%s", uuid.New().String()[:8])
	company := models.Company{
		ID:        companyID,
		Name:      req.CompanyName,
		Address:   strings.TrimSpace(req.CompanyAddress),
		Phone:     utils.NormalizePhone(req.CompanyPhone),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := config.DB.Create(&company).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal membuat entitas Perusahaan baru: " + err.Error(),
		})
	}

	// 2. Hash Password Owner & Buat Akun Owner Pertama
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.OwnerPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengenkripsi password",
		})
	}

	ownerUser := models.User{
		ID:        fmt.Sprintf("usr_%s", uuid.New().String()[:8]),
		CompanyID: companyID,
		FullName:  req.OwnerName,
		Phone:     cleanOwnerPhone,
		Email:     strings.TrimSpace(req.OwnerEmail),
		Password:  string(hashedPassword),
		Role:      "owner",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := config.DB.Create(&ownerUser).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal membuat akun Owner: " + err.Error(),
		})
	}

	// 3. Salin paket dekorasi & addon template awal agar usaha baru langsung siap pakai
	seedInitialPackagesAndAddonsForCompany(companyID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Usaha Dekorasi & Akun Owner berhasil didaftarkan!",
		"company": company,
		"owner": models.UserResponse{
			ID:        ownerUser.ID,
			CompanyID: companyID,
			FullName:  ownerUser.FullName,
			Phone:     ownerUser.Phone,
			Email:     ownerUser.Email,
			Role:      ownerUser.Role,
		},
	})
}

// UpdateCompany handles PUT /api/superadmin/companies/:id
func UpdateCompany(c *fiber.Ctx) error {
	id := c.Params("id")
	var comp models.Company
	if err := config.DB.Where("id = ?", id).First(&comp).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Perusahaan tidak ditemukan",
		})
	}

	type updateReq struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		Phone   string `json:"phone"`
		LogoURL string `json:"logo_url"`
	}
	var req updateReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if strings.TrimSpace(req.Name) != "" {
		comp.Name = strings.TrimSpace(req.Name)
	}
	if req.Address != "" {
		comp.Address = strings.TrimSpace(req.Address)
	}
	if req.Phone != "" {
		comp.Phone = utils.NormalizePhone(req.Phone)
	}
	if req.LogoURL != "" {
		comp.LogoURL = strings.TrimSpace(req.LogoURL)
	}

	comp.UpdatedAt = time.Now()
	if err := config.DB.Save(&comp).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memperbarui data perusahaan: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Data perusahaan berhasil diperbarui",
		"data":    comp,
	})
}

// DeleteCompany handles DELETE /api/superadmin/companies/:id
func DeleteCompany(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "cmp_default" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Perusahaan default utama tidak dapat dihapus",
		})
	}

	var comp models.Company
	if err := config.DB.Where("id = ?", id).First(&comp).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Perusahaan tidak ditemukan"})
	}

	// Hapus data berelasi
	config.DB.Where("company_id = ?", id).Delete(&models.User{})
	config.DB.Where("company_id = ?", id).Delete(&models.Agenda{})
	config.DB.Where("company_id = ?", id).Delete(&models.Expense{})
	config.DB.Where("company_id = ?", id).Delete(&models.Package{})
	config.DB.Where("company_id = ?", id).Delete(&models.Addon{})
	config.DB.Where("company_id = ?", id).Delete(&models.Invitation{})
	config.DB.Delete(&comp)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": fmt.Sprintf("Perusahaan %s beserta seluruh datanya berhasil dihapus", comp.Name),
	})
}

// Helper untuk memberikan paket dan addon awal untuk company baru
func seedInitialPackagesAndAddonsForCompany(companyID string) {
	samplePackages := []models.Package{
		{
			ID:          fmt.Sprintf("pkg_%s", uuid.New().String()[:8]),
			CompanyID:   companyID,
			Name:        "Paket Akad Minimalis",
			Description: "Backdrop 2.5m, rangkaian bunga artificial premium, 2 standing flower, karpet akad, set kursi pengantin",
			BasePrice:   2500000,
		},
		{
			ID:          fmt.Sprintf("pkg_%s", uuid.New().String()[:8]),
			CompanyID:   companyID,
			Name:        "Paket Lamaran Modern Rustic",
			Description: "Backdrop 3m hexagon/arch, full bunga kombinasi, spotlight, welcome sign akrilik, ring box",
			BasePrice:   4000000,
		},
		{
			ID:          fmt.Sprintf("pkg_%s", uuid.New().String()[:8]),
			CompanyID:   companyID,
			Name:        "Paket Pernikahan Elegan Gold",
			Description: "Pelaminan 6m, mini gallery photo, pergola masuk, lighting ambience lengkap, karpet jalan",
			BasePrice:   9500000,
		},
	}
	for _, p := range samplePackages {
		config.DB.Create(&p)
	}

	sampleAddons := []models.Addon{
		{
			ID:          fmt.Sprintf("adn_%s", uuid.New().String()[:8]),
			CompanyID:   companyID,
			Name:        "Lighting Ambient & Spotlight Warm",
			Price:       350000,
			Description: "Set lighting sorot dekorasi warm white untuk suasana hangat",
		},
		{
			ID:          fmt.Sprintf("adn_%s", uuid.New().String()[:8]),
			CompanyID:   companyID,
			Name:        "Welcome Sign Akrilik Custom Text",
			Price:       250000,
			Description: "Papan ucapan selamat datang dari akrilik bening dengan ukiran nama",
		},
	}
	for _, a := range sampleAddons {
		config.DB.Create(&a)
	}
}

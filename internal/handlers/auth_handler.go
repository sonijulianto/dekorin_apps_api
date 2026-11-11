package handlers

import (
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
	"gorm.io/gorm"
)

// issueAuthTokenAndResponse membuat JWT token 7 hari dan payload UserResponse yang konsisten untuk Login & Register.
func issueAuthTokenAndResponse(user models.User) (string, models.UserResponse, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":         user.ID,
		"company_id": user.CompanyID,
		"phone":      user.Phone,
		"email":      user.Email,
		"role":       user.Role,
		"exp":        time.Now().Add(time.Hour * 24 * 7).Unix(),
	})

	tokenString, err := token.SignedString(middleware.JwtSecret)
	if err != nil {
		return "", models.UserResponse{}, err
	}

	companyName := ""
	companyAddress := ""
	companyPhone := ""
	companyLogo := ""
	companyEmail := ""
	companyDescription := ""
	companyInstagram := ""
	companyBankInfo := ""
	companyInvoiceTemplate := DefaultInvoiceTemplate
	if user.Company != nil {
		companyName = user.Company.Name
		companyAddress = user.Company.Address
		companyPhone = user.Company.Phone
		companyLogo = user.Company.LogoURL
		companyEmail = user.Company.Email
		companyDescription = user.Company.Description
		companyInstagram = user.Company.Instagram
		companyBankInfo = user.Company.BankInfo
		companyInvoiceTemplate = EffectiveInvoiceTemplate(user.Company.InvoiceTemplate)
	}

	perms := GetEffectiveUserPermissions(user.CompanyID, user.Role)
	if perms == nil {
		perms = []string{}
	}

	resp := models.UserResponse{
		ID:                     user.ID,
		CompanyID:              user.CompanyID,
		CompanyName:            companyName,
		CompanyAddress:         companyAddress,
		CompanyPhone:           companyPhone,
		CompanyLogo:            companyLogo,
		CompanyEmail:           companyEmail,
		CompanyDescription:     companyDescription,
		CompanyInstagram:       companyInstagram,
		CompanyBankInfo:        companyBankInfo,
		CompanyInvoiceTemplate: companyInvoiceTemplate,
		FullName:               user.FullName,
		Phone:                  user.Phone,
		Email:                  user.Email,
		Role:                   user.Role,
		Permissions:            perms,
	}
	return tokenString, resp, nil
}

// Login handles the authentication endpoint using Phone or Email and Password
func Login(c *fiber.Ctx) error {
	req := new(models.LoginRequest)

	// Parse JSON body
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
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
			"error": "Nomor Handphone dan password wajib diisi",
		})
	}

	var user models.User
	var err error

	// Cari user di database berdasarkan No HP atau Email beserta data Company
	if isPhone {
		err = config.DB.Preload("Company").Where("phone = ?", cleanPhone).First(&user).Error
		// Fallback: Jika tidak ketemu, coba cari jika user menginputkan email di field phone
		if err != nil && strings.Contains(req.Phone, "@") {
			err = config.DB.Preload("Company").Where("email = ?", strings.TrimSpace(req.Phone)).First(&user).Error
		}
	} else {
		err = config.DB.Preload("Company").Where("email = ?", identifier).First(&user).Error
	}

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Nomor Handphone atau password salah",
		})
	}

	// Validasi Password menggunakan Bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Nomor Handphone atau password salah",
		})
	}

	tokenString, userResp, err := issueAuthTokenAndResponse(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal membuat token autentikasi",
		})
	}

	// Return data user beserta token-nya
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Login berhasil",
		"token":   tokenString,
		"data":    userResp,
	})
}

// RegisterCouple handles POST /api/auth/register-couple untuk pendaftaran mandiri Calon Pengantin
func RegisterCouple(c *fiber.Ctx) error {
	req := new(models.RegisterCoupleRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format data pendaftaran tidak valid",
		})
	}

	fullName := strings.TrimSpace(req.FullName)
	cleanPhone := utils.NormalizePhone(req.Phone)
	cleanEmail := strings.TrimSpace(strings.ToLower(req.Email))
	password := strings.TrimSpace(req.Password)

	if fullName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nama lengkap wajib diisi",
		})
	}
	if cleanPhone == "" || len(cleanPhone) < 9 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nomor WhatsApp / HP tidak valid (minimal 9 digit)",
		})
	}
	if len(password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password minimal 6 karakter",
		})
	}

	var existingCount int64
	config.DB.Model(&models.User{}).Where("phone = ?", cleanPhone).Count(&existingCount)
	if existingCount > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Nomor WhatsApp sudah terdaftar. Silakan masuk menggunakan akun Anda.",
		})
	}
	if cleanEmail != "" {
		var emailCount int64
		config.DB.Model(&models.User{}).Where("LOWER(email) = ?", cleanEmail).Count(&emailCount)
		if emailCount > 0 {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Email sudah digunakan oleh akun lain.",
			})
		}
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengenkripsi password",
		})
	}

	now := time.Now()
	newUser := models.User{
		ID:        "usr_" + uuid.New().String()[:12],
		CompanyID: "",
		FullName:  fullName,
		Phone:     cleanPhone,
		Email:     cleanEmail,
		Password:  string(hashed),
		Role:      "couple",
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&newUser).Error; err != nil {
			return err
		}
		_, err := CreateOrJoinWeddingPlanForCouple(
			tx,
			&newUser,
			strings.TrimSpace(req.PartnerName),
			strings.TrimSpace(req.WeddingDate),
			strings.TrimSpace(req.City),
			strings.TrimSpace(req.VenueName),
			req.BudgetTarget,
			strings.TrimSpace(req.JoinCode),
		)
		return err
	})
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	tokenString, userResp, err := issueAuthTokenAndResponse(newUser)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Akun berhasil dibuat, namun gagal membuat token login",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Pendaftaran akun Calon Pengantin berhasil",
		"token":   tokenString,
		"data":    userResp,
	})
}

// ChangePassword handles POST /api/auth/change-password
func ChangePassword(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Sesi tidak valid atau telah berakhir",
		})
	}

	req := new(models.ChangePasswordRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	currentPassword := strings.TrimSpace(req.CurrentPassword)
	newPassword := strings.TrimSpace(req.NewPassword)

	if currentPassword == "" || newPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password saat ini dan password baru wajib diisi",
		})
	}

	if len(newPassword) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password baru minimal 6 karakter",
		})
	}

	var user models.User
	if err := config.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Pengguna tidak ditemukan",
		})
	}

	// Verifikasi password saat ini menggunakan Bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password saat ini salah",
		})
	}

	// Cegah penggunaan password yang sama persis
	if currentPassword == newPassword {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password baru tidak boleh sama dengan password saat ini",
		})
	}

	// Hash password baru
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengenkripsi password baru",
		})
	}

	user.Password = string(hashed)
	user.UpdatedAt = time.Now()
	if err := config.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memperbarui password",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Password berhasil diubah",
	})
}


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
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// GetAllUsers handles GET /api/users
// Otomatis difilter berdasarkan company_id user yang login
func GetAllUsers(c *fiber.Ctx) error {
	creatorCompanyID := middleware.GetCompanyID(c)
	creatorRole, _ := c.Locals("user_role").(string)

	query := config.DB.Preload("Company")

	// Jika bukan superadmin, hanya tampilkan user dari perusahaannya sendiri
	if !strings.EqualFold(creatorRole, "superadmin") {
		if creatorCompanyID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Company ID tidak ditemukan pada sesi user",
			})
		}
		query = query.Where("company_id = ?", creatorCompanyID)
	} else {
		// Superadmin bisa opsional memfilter by company_id lewat query param
		if qCompany := c.Query("company_id"); qCompany != "" {
			query = query.Where("company_id = ?", qCompany)
		}
	}

	var users []models.User
	if err := query.Order("role asc, created_at desc").Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil data user: " + err.Error(),
		})
	}

	var resp []models.UserResponse
	for _, u := range users {
		companyName := ""
		if u.Company != nil {
			companyName = u.Company.Name
		}
		resp = append(resp, models.UserResponse{
			ID:          u.ID,
			CompanyID:   u.CompanyID,
			CompanyName: companyName,
			FullName:    u.FullName,
			Phone:       u.Phone,
			Email:       u.Email,
			Role:        u.Role,
			CreatedAt:   u.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   u.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

// CreateUser handles POST /api/users
// Aturan Multi-Company:
// - Owner dapat membuat akun Owner (Co-Owner), Admin, dan Staff/Crew untuk perusahaannya sendiri
// - Admin HANYA dapat membuat akun Staff/Crew untuk perusahaannya sendiri
// - Staff/Crew tidak memiliki izin membuat akun
func CreateUser(c *fiber.Ctx) error {
	creatorRole, _ := c.Locals("user_role").(string)
	creatorRole = strings.ToLower(strings.TrimSpace(creatorRole))
	creatorCompanyID := middleware.GetCompanyID(c)

	if creatorRole != "owner" && creatorRole != "admin" && creatorRole != "superadmin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Anda tidak memiliki izin untuk membuat akun",
		})
	}

	req := new(models.CreateUserRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format data tidak valid",
		})
	}

	// Normalisasi dan validasi role yang akan dibuat
	targetRole := strings.ToLower(strings.TrimSpace(req.Role))
	if targetRole == "staff" {
		targetRole = "crew"
	}
	if targetRole == "" {
		targetRole = "crew"
	}

	// Validasi hak akses berdasarkan role pembuat
	if creatorRole == "admin" {
		if targetRole != "crew" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Admin hanya diperbolehkan membuat akun Staff/Crew",
			})
		}
	} else if creatorRole == "owner" {
		// Owner BISA membuat akun: 'owner' (Co-Owner), 'admin', atau 'crew'
		if targetRole != "owner" && targetRole != "admin" && targetRole != "crew" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Role tidak valid. Pilihan role: 'owner', 'admin', atau 'crew'",
			})
		}
	}

	// Validasi input data
	req.FullName = strings.TrimSpace(req.FullName)
	if req.FullName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nama lengkap wajib diisi",
		})
	}

	cleanPhone := utils.NormalizePhone(req.Phone)
	if len(cleanPhone) < 9 || len(cleanPhone) > 15 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Nomor Handphone tidak valid (minimal 9 digit)",
		})
	}

	if len(strings.TrimSpace(req.Password)) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password minimal 6 karakter",
		})
	}

	// Cek apakah No HP sudah terdaftar di seluruh sistem
	var existing models.User
	if err := config.DB.Where("phone = ?", cleanPhone).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Nomor Handphone sudah terdaftar pada akun lain",
		})
	}

	// Hash password dengan Bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengenkripsi password",
		})
	}

	// Buat user baru dan ikat otomatis ke CompanyID pembuatnya
	newUser := models.User{
		ID:        fmt.Sprintf("usr_%s", uuid.New().String()[:8]),
		CompanyID: creatorCompanyID,
		FullName:  req.FullName,
		Phone:     cleanPhone,
		Email:     strings.TrimSpace(req.Email),
		Password:  string(hashedPassword),
		Role:      targetRole,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := config.DB.Create(&newUser).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan akun baru ke database",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Akun berhasil dibuat",
		"data": models.UserResponse{
			ID:        newUser.ID,
			CompanyID: newUser.CompanyID,
			FullName:  newUser.FullName,
			Phone:     newUser.Phone,
			Email:     newUser.Email,
			Role:      newUser.Role,
			CreatedAt: newUser.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	})
}

// UpdateUser handles PUT /api/users/:id
func UpdateUser(c *fiber.Ctx) error {
	creatorRole, _ := c.Locals("user_role").(string)
	creatorRole = strings.ToLower(strings.TrimSpace(creatorRole))
	creatorID, _ := c.Locals("user_id").(string)
	creatorCompanyID := middleware.GetCompanyID(c)
	targetID := c.Params("id")

	var targetUser models.User
	if err := config.DB.Where("id = ?", targetID).First(&targetUser).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Akun pengguna tidak ditemukan",
		})
	}

	// Pastikan user berasal dari perusahaan yang sama jika bukan superadmin
	if !strings.EqualFold(creatorRole, "superadmin") && targetUser.CompanyID != creatorCompanyID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Akses ditolak: Pengguna ini bukan dari perusahaan Anda",
		})
	}

	isSelf := creatorID == targetID
	if !isSelf {
		if creatorRole == "admin" && targetUser.Role != "crew" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Admin hanya diperbolehkan mengelola akun Staff/Crew",
			})
		}
		if creatorRole != "owner" && creatorRole != "admin" && creatorRole != "superadmin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Anda tidak memiliki izin untuk mengedit akun ini",
			})
		}
	}

	req := new(models.UpdateUserRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format data tidak valid",
		})
	}

	if strings.TrimSpace(req.FullName) != "" {
		targetUser.FullName = strings.TrimSpace(req.FullName)
	}

	if strings.TrimSpace(req.Phone) != "" {
		cleanPhone := utils.NormalizePhone(req.Phone)
		if len(cleanPhone) >= 9 {
			var duplicate models.User
			if err := config.DB.Where("phone = ? AND id != ?", cleanPhone, targetID).First(&duplicate).Error; err == nil {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error": "Nomor Handphone sudah digunakan oleh akun lain",
				})
			}
			targetUser.Phone = cleanPhone
		}
	}

	if req.Email != "" {
		targetUser.Email = strings.TrimSpace(req.Email)
	}

	// Update role hanya jika dilakukan oleh Owner (atau Superadmin)
	if req.Role != "" && !isSelf {
		newRole := strings.ToLower(strings.TrimSpace(req.Role))
		if newRole == "staff" {
			newRole = "crew"
		}
		if creatorRole == "admin" && newRole != "crew" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Admin tidak dapat mengubah role selain Crew",
			})
		}
		if creatorRole == "owner" || creatorRole == "superadmin" {
			if newRole == "owner" || newRole == "admin" || newRole == "crew" {
				targetUser.Role = newRole
			}
		}
	}

	// Update password jika diisi
	if strings.TrimSpace(req.Password) != "" {
		if len(strings.TrimSpace(req.Password)) < 6 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Password baru minimal 6 karakter",
			})
		}
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err == nil {
			targetUser.Password = string(hashed)
		}
	}

	targetUser.UpdatedAt = time.Now()
	if err := config.DB.Save(&targetUser).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memperbarui data user",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Data pengguna berhasil diperbarui",
		"data": models.UserResponse{
			ID:        targetUser.ID,
			CompanyID: targetUser.CompanyID,
			FullName:  targetUser.FullName,
			Phone:     targetUser.Phone,
			Email:     targetUser.Email,
			Role:      targetUser.Role,
			UpdatedAt: targetUser.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
	})
}

// DeleteUser handles DELETE /api/users/:id
func DeleteUser(c *fiber.Ctx) error {
	creatorRole, _ := c.Locals("user_role").(string)
	creatorRole = strings.ToLower(strings.TrimSpace(creatorRole))
	creatorID, _ := c.Locals("user_id").(string)
	creatorCompanyID := middleware.GetCompanyID(c)
	targetID := c.Params("id")

	if targetID == creatorID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Anda tidak dapat menghapus akun Anda sendiri",
		})
	}

	var targetUser models.User
	if err := config.DB.Where("id = ?", targetID).First(&targetUser).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Pengguna tidak ditemukan",
		})
	}

	if !strings.EqualFold(creatorRole, "superadmin") && targetUser.CompanyID != creatorCompanyID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Akses ditolak: Pengguna ini bukan dari perusahaan Anda",
		})
	}

	if targetUser.Role == "superadmin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Akun Superadmin tidak dapat dihapus",
		})
	}

	if creatorRole == "admin" && targetUser.Role != "crew" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin hanya diperbolehkan menghapus akun Staff/Crew",
		})
	}

	if creatorRole != "owner" && creatorRole != "admin" && creatorRole != "superadmin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Anda tidak memiliki izin untuk menghapus akun ini",
		})
	}

	if err := config.DB.Delete(&targetUser).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menghapus pengguna",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": fmt.Sprintf("Akun %s berhasil dihapus", targetUser.FullName),
	})
}

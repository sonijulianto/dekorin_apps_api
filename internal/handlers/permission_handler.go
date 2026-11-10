package handlers

import (
	"encoding/json"
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetAllPermissionDefinitions mengembalikan definisi lengkap modul & hak akses yang tersedia
func GetAllPermissionDefinitions() []models.PermissionCategory {
	return []models.PermissionCategory{
		{
			Category: "Keuangan & Sensitif",
			Permissions: []models.PermissionItem{
				{
					ID:          "finance.view_summary",
					Name:        "Lihat Ringkasan Keuangan (Dashboard)",
					Description: "Melihat omzet, sisa tagihan klien, dan estimasi profit di dashboard",
				},
				{
					ID:          "finance.view_price",
					Name:        "Lihat Nominal Harga & Pembayaran",
					Description: "Melihat rincian harga paket, DP, dan sisa tagihan pada agenda",
				},
				{
					ID:          "expenses.view",
					Name:        "Lihat Riwayat Pengeluaran",
					Description: "Melihat catatan biaya operasional dan pengeluaran belanja acara",
				},
				{
					ID:          "expenses.manage",
					Name:        "Kelola Pengeluaran & Scan Struk AI",
					Description: "Menambah, mengedit, atau menghapus pengeluaran dan memakai scan struk AI",
				},
			},
		},
		{
			Category: "Agenda & Acara",
			Permissions: []models.PermissionItem{
				{
					ID:          "agenda.view",
					Name:        "Lihat Agenda & Kalender Acara",
					Description: "Melihat daftar agenda dekorasi, jadwal acara, dan detail lokasi",
				},
				{
					ID:          "agenda.create",
					Name:        "Buat Agenda Baru",
					Description: "Menambahkan jadwal agenda atau booking acara baru",
				},
				{
					ID:          "agenda.edit",
					Name:        "Edit Rincian Agenda",
					Description: "Mengubah jadwal, lokasi, paket, model dekorasi, dan catatan agenda",
				},
				{
					ID:          "agenda.delete",
					Name:        "Hapus Agenda Acara",
					Description: "Menghapus data agenda dekorasi dari sistem",
				},
				{
					ID:          "agenda.download_invoice",
					Name:        "Unduh Invoice & Surat Jalan PDF",
					Description: "Mencetak atau mengunduh invoice resmi & surat jalan acara",
				},
			},
		},
		{
			Category: "Inventaris & Pick List Lapangan",
			Permissions: []models.PermissionItem{
				{
					ID:          "picklist.checklist",
					Name:        "Centang Muat & Bongkar (Checklist)",
					Description: "Mencentang status muat (packed) dan kembali (returned) barang dekorasi",
				},
				{
					ID:          "picklist.manage_items",
					Name:        "Kelola Barang Bawaan Acara",
					Description: "Menambah atau mengurangi barang dari checklist muat bongkar acara",
				},
				{
					ID:          "inventory.view",
					Name:        "Lihat Inventaris & Cek Ketersediaan Alat",
					Description: "Melihat katalog properti dekorasi, alat kerja, dan kalender ketersediaan",
				},
				{
					ID:          "inventory.manage",
					Name:        "Kelola Master Properti & Stok Alat",
					Description: "Menambah, mengedit, atau menghapus data inventaris barang dekorasi",
				},
			},
		},
		{
			Category: "Katalog Master & Brand",
			Permissions: []models.PermissionItem{
				{
					ID:          "catalog.manage",
					Name:        "Kelola Paket & Model Dekorasi",
					Description: "Mengatur katalog paket harga, addon, dan model dekorasi",
				},
				{
					ID:          "brand.manage",
					Name:        "Kelola Multi-Brand / Sub-Brand",
					Description: "Mengatur identitas sub-brand dan logo terpisah untuk market berbeda",
				},
				{
					ID:          "invitation.manage",
					Name:        "Kelola Undangan Digital",
					Description: "Membuat, mengedit, dan mengelola website undangan digital pengantin",
				},
			},
		},
		{
			Category: "Fitur AI & Tim",
			Permissions: []models.PermissionItem{
				{
					ID:          "ai.use_studio",
					Name:        "Akses AI Decor Studio & Asisten",
					Description: "Menggunakan simulasi warna dekorasi berbasis AI & AI chat assistant",
				},
				{
					ID:          "team.manage_crew",
					Name:        "Kelola Akun Crew / Staf",
					Description: "Menambah, mengubah sandi, atau menghapus akun Crew lapangan",
				},
			},
		},
	}
}

// GetAllPermissionKeys mengembalikan daftar seluruh permission key
func GetAllPermissionKeys() []string {
	var keys []string
	for _, cat := range GetAllPermissionDefinitions() {
		for _, p := range cat.Permissions {
			keys = append(keys, p.ID)
		}
	}
	return keys
}

// GetDefaultPermissionsForRole mengembalikan izin default jika belum pernah diatur
func GetDefaultPermissionsForRole(role string) []string {
	normRole := strings.ToLower(strings.TrimSpace(role))
	switch normRole {
	case "superadmin", "owner":
		return GetAllPermissionKeys()
	case "admin":
		// Admin default: semua fitur KECUALI 'agenda.delete' (untuk mencegah ketidaksengajaan terhapus)
		var adminPerms []string
		for _, k := range GetAllPermissionKeys() {
			if k != "agenda.delete" {
				adminPerms = append(adminPerms, k)
			}
		}
		return adminPerms
	case "crew", "staff":
		// Crew default: operasional lapangan saja (lihat agenda, centang muat bongkar, cek stok properti)
		return []string{
			"agenda.view",
			"picklist.checklist",
			"inventory.view",
		}
	default:
		return []string{}
	}
}

// GetEffectiveUserPermissions mengambil izin aktif seorang user berdasarkan company_id dan role-nya
func GetEffectiveUserPermissions(companyID string, role string) []string {
	normRole := strings.ToLower(strings.TrimSpace(role))
	if normRole == "superadmin" || normRole == "owner" {
		return GetAllPermissionKeys()
	}

	if companyID == "" {
		return GetDefaultPermissionsForRole(normRole)
	}

	var permRecord models.CompanyRolePermission
	err := config.DB.Where("company_id = ? AND role = ?", companyID, normRole).First(&permRecord).Error
	if err != nil {
		// Belum pernah disetel khusus oleh owner, gunakan default role
		return GetDefaultPermissionsForRole(normRole)
	}

	var perms []string
	if err := json.Unmarshal([]byte(permRecord.Permissions), &perms); err != nil {
		return GetDefaultPermissionsForRole(normRole)
	}
	return perms
}

// GetCompanyRolePermissions handles GET /api/company/role-permissions
func GetCompanyRolePermissions(c *fiber.Ctx) error {
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
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Company ID tidak ditemukan pada sesi user",
		})
	}

	var records []models.CompanyRolePermission
	config.DB.Where("company_id = ?", companyID).Find(&records)

	roleMap := map[string][]string{
		"admin": GetDefaultPermissionsForRole("admin"),
		"crew":  GetDefaultPermissionsForRole("crew"),
	}

	for _, rec := range records {
		var perms []string
		if err := json.Unmarshal([]byte(rec.Permissions), &perms); err == nil {
			roleMap[rec.Role] = perms
		}
	}

	return c.Status(fiber.StatusOK).JSON(models.RolePermissionsResponse{
		Definitions:     GetAllPermissionDefinitions(),
		RolePermissions: roleMap,
	})
}

// UpdateCompanyRolePermissions handles PUT /api/company/role-permissions
// Dapat diakses oleh Owner dan Superadmin
func UpdateCompanyRolePermissions(c *fiber.Ctx) error {
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
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Company ID tidak ditemukan pada sesi user",
		})
	}

	req := new(models.UpdateRolePermissionRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format data tidak valid",
		})
	}

	targetRole := strings.ToLower(strings.TrimSpace(req.Role))
	if targetRole == "staff" {
		targetRole = "crew"
	}

	if targetRole != "admin" && targetRole != "crew" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Role yang dapat diatur hak aksesnya hanya 'admin' dan 'crew'",
		})
	}

	// Validasi bahwa permission yang dikirim adalah valid key
	allKeys := GetAllPermissionKeys()
	validKeyMap := make(map[string]bool)
	for _, k := range allKeys {
		validKeyMap[k] = true
	}

	var sanitizedPerms []string
	for _, p := range req.Permissions {
		p = strings.TrimSpace(p)
		if validKeyMap[p] {
			sanitizedPerms = append(sanitizedPerms, p)
		}
	}
	if sanitizedPerms == nil {
		sanitizedPerms = []string{}
	}

	jsonBytes, err := json.Marshal(sanitizedPerms)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal memproses data permission: " + err.Error(),
		})
	}

	var existing models.CompanyRolePermission
	err = config.DB.Where("company_id = ? AND role = ?", companyID, targetRole).First(&existing).Error
	if err != nil {
		// Record belum ada -> buat baru
		newRecord := models.CompanyRolePermission{
			ID:          "crp_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:24],
			CompanyID:   companyID,
			Role:        targetRole,
			Permissions: string(jsonBytes),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := config.DB.Create(&newRecord).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal menyimpan hak akses: " + err.Error(),
			})
		}
	} else {
		// Record sudah ada -> update
		existing.Permissions = string(jsonBytes)
		existing.UpdatedAt = time.Now()
		if err := config.DB.Save(&existing).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Gagal memperbarui hak akses: " + err.Error(),
			})
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":     "Hak akses untuk role " + targetRole + " berhasil disimpan",
		"role":        targetRole,
		"permissions": sanitizedPerms,
	})
}

// ResetCompanyRolePermissions handles POST /api/company/role-permissions/reset
// Mereset hak akses ke setelan bawaan sistem (Owner dan Superadmin)
func ResetCompanyRolePermissions(c *fiber.Ctx) error {
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
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Company ID tidak ditemukan pada sesi user",
		})
	}

	role := strings.ToLower(strings.TrimSpace(c.Query("role")))
	query := config.DB.Where("company_id = ?", companyID)
	if role == "admin" || role == "crew" {
		query = query.Where("role = ?", role)
	}

	if err := query.Delete(&models.CompanyRolePermission{}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mereset hak akses: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Hak akses berhasil dikembalikan ke pengaturan default sistem",
	})
}

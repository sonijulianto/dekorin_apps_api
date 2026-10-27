package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetLatestAppVersion menangani GET /api/app/version dan GET /api/apk/check-version (Publik)
func GetLatestAppVersion(c *fiber.Ctx) error {
	platform := c.Query("platform", "android")

	var appVersion models.AppVersion
	err := config.DB.Where("platform = ?", platform).Order("version_code DESC, created_at DESC").First(&appVersion).Error
	if err != nil {
		// Fallback default jika belum ada data versi di database
		baseURL := fmt.Sprintf("%s://%s", c.Protocol(), c.Hostname())
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "success",
			"message": "Versi aplikasi berhasil diambil",
			"data": models.AppVersionResponse{
				Platform:       platform,
				VersionName:    "1.0.0",
				VersionCode:    1,
				MinVersionCode: 1,
				DownloadURL:    baseURL + "/uploads/apks/dekorin-latest.apk",
				ReleaseNotes:   "Versi awal aplikasi Dekorin",
				ForceUpdate:    false,
			},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Versi aplikasi berhasil diambil",
		"data": models.AppVersionResponse{
			Platform:       appVersion.Platform,
			VersionName:    appVersion.VersionName,
			VersionCode:    appVersion.VersionCode,
			MinVersionCode: appVersion.MinVersionCode,
			DownloadURL:    appVersion.DownloadURL,
			ReleaseNotes:   appVersion.ReleaseNotes,
			ForceUpdate:    appVersion.ForceUpdate,
		},
	})
}

// GetAppVersionHistory menangani GET /api/app/version-history (Khusus Superadmin)
func GetAppVersionHistory(c *fiber.Ctx) error {
	platform := c.Query("platform", "android")
	var versions []models.AppVersion
	if err := config.DB.Where("platform = ?", platform).Order("version_code DESC, created_at DESC").Find(&versions).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengambil histori rilis APK: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   versions,
	})
}

// UpdateAppVersion menangani POST/PUT /api/app/version untuk mengatur versi rilis baru (Khusus Superadmin)
func UpdateAppVersion(c *fiber.Ctx) error {
	type UpdateVersionReq struct {
		Platform       string `json:"platform"`
		VersionName    string `json:"version_name"`
		VersionCode    int    `json:"version_code"`
		MinVersionCode int    `json:"min_version_code"`
		DownloadURL    string `json:"download_url"`
		ReleaseNotes   string `json:"release_notes"`
		ForceUpdate    bool   `json:"force_update"`
	}

	var req UpdateVersionReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format request tidak valid",
		})
	}

	if req.Platform == "" {
		req.Platform = "android"
	}
	if req.VersionName == "" || req.VersionCode == 0 || req.DownloadURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "version_name, version_code, dan download_url wajib diisi",
		})
	}
	if req.MinVersionCode == 0 {
		req.MinVersionCode = 1
	}

	newVersion := models.AppVersion{
		ID:             "ver_" + uuid.New().String()[:8],
		Platform:       req.Platform,
		VersionName:    req.VersionName,
		VersionCode:    req.VersionCode,
		MinVersionCode: req.MinVersionCode,
		DownloadURL:    req.DownloadURL,
		ReleaseNotes:   req.ReleaseNotes,
		ForceUpdate:    req.ForceUpdate,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := config.DB.Create(&newVersion).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan data versi baru: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Informasi update aplikasi berhasil disimpan",
		"data":    newVersion,
	})
}

// UploadApkFile menangani POST /api/app/upload-apk (Khusus Superadmin)
// Mendukung upload file APK langsung dan menyimpan info rilis versi ke database
func UploadApkFile(c *fiber.Ctx) error {
	file, err := c.FormFile("apk")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File APK wajib diunggah (form key: apk)",
		})
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".apk" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File harus berekstensi .apk",
		})
	}

	apkDir := "./uploads/apks"
	if err := os.MkdirAll(apkDir, os.ModePerm); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal membuat direktori upload APK di server",
		})
	}

	versionName := strings.TrimSpace(c.FormValue("version_name"))
	if versionName == "" {
		versionName = "1.0.0"
	}

	versionCodeStr := strings.TrimSpace(c.FormValue("version_code"))
	versionCode, _ := strconv.Atoi(versionCodeStr)
	if versionCode <= 0 {
		versionCode = 1
	}

	minVersionCodeStr := strings.TrimSpace(c.FormValue("min_version_code"))
	minVersionCode, _ := strconv.Atoi(minVersionCodeStr)
	if minVersionCode <= 0 {
		minVersionCode = 1
	}

	releaseNotes := strings.TrimSpace(c.FormValue("release_notes"))
	forceUpdate := strings.ToLower(strings.TrimSpace(c.FormValue("force_update"))) == "true" || c.FormValue("force_update") == "1"

	// Nama file unik: dekorin_v<version>_<timestamp>.apk
	filename := fmt.Sprintf("dekorin_v%s_%d.apk", versionName, time.Now().Unix())
	savePath := filepath.Join(apkDir, filename)

	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan file APK ke server: " + err.Error(),
		})
	}

	// Simpan juga salinan sebagai dekorin-latest.apk
	latestPath := filepath.Join(apkDir, "dekorin-latest.apk")
	_ = c.SaveFile(file, latestPath)

	downloadURL := fmt.Sprintf("%s://%s/uploads/apks/%s", c.Protocol(), c.Hostname(), filename)

	// Simpan data ke tabel database AppVersion
	newVersion := models.AppVersion{
		ID:             "ver_" + uuid.New().String()[:8],
		Platform:       "android",
		VersionName:    versionName,
		VersionCode:    versionCode,
		MinVersionCode: minVersionCode,
		DownloadURL:    downloadURL,
		ReleaseNotes:   releaseNotes,
		ForceUpdate:    forceUpdate,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := config.DB.Create(&newVersion).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "File APK berhasil diunggah tapi gagal mencatat ke database: " + err.Error(),
		})
	}

	// Auto Retention: Bersihkan file APK lama di disk, simpan hanya 5 rilis terakhir
	cleanOldApkFiles(5)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":      "File APK dan versi baru berhasil dirilis!",
		"filename":     filename,
		"download_url": downloadURL,
		"data":         newVersion,
	})
}

// DeleteAppVersion menangani DELETE /api/app/version/:id (Khusus Superadmin)
// Menghapus data dari database sekaligus menghapus file APK fisik dari disk server
func DeleteAppVersion(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID versi tidak valid",
		})
	}

	var appVersion models.AppVersion
	if err := config.DB.First(&appVersion, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Data versi tidak ditemukan",
		})
	}

	// Hapus file fisik APK dari disk server
	if appVersion.DownloadURL != "" {
		parts := strings.Split(appVersion.DownloadURL, "/")
		if len(parts) > 0 {
			fname := parts[len(parts)-1]
			// Jangan hapus dekorin-latest.apk
			if fname != "dekorin-latest.apk" && strings.HasSuffix(fname, ".apk") {
				filePath := filepath.Join("./uploads/apks", fname)
				_ = os.Remove(filePath)
			}
		}
	}

	// Hapus record dari database
	if err := config.DB.Delete(&appVersion).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menghapus versi: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Data versi dan file fisik APK berhasil dibersihkan dari server",
	})
}

// cleanOldApkFiles menjaga kuota disk dengan menghapus file APK lama melebihi batas keepLatest
func cleanOldApkFiles(keepLatest int) {
	var versions []models.AppVersion
	if err := config.DB.Where("platform = ?", "android").Order("version_code DESC, created_at DESC").Find(&versions).Error; err != nil {
		return
	}

	if len(versions) > keepLatest {
		for _, v := range versions[keepLatest:] {
			if v.DownloadURL != "" {
				parts := strings.Split(v.DownloadURL, "/")
				if len(parts) > 0 {
					fname := parts[len(parts)-1]
					if fname != "dekorin-latest.apk" && strings.HasSuffix(fname, ".apk") {
						filePath := filepath.Join("./uploads/apks", fname)
						_ = os.Remove(filePath)
					}
				}
			}
		}
	}
}

// DownloadLatestApk menangani GET /download/apk dan GET /api/app/download-latest
// Mengarahkan pengguna (Owner, Admin, Crew) langsung ke link download file APK versi terbaru
func DownloadLatestApk(c *fiber.Ctx) error {
	var appVersion models.AppVersion
	err := config.DB.Where("platform = ?", "android").Order("version_code DESC, created_at DESC").First(&appVersion).Error
	if err != nil || appVersion.DownloadURL == "" {
		baseURL := fmt.Sprintf("%s://%s", c.Protocol(), c.Hostname())
		return c.Redirect(baseURL + "/uploads/apks/dekorin-latest.apk")
	}
	return c.Redirect(appVersion.DownloadURL)
}


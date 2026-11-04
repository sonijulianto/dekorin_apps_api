package handlers

import (
	"fmt"
	"path/filepath"
	"strings"

	"dekorin_apps_api/internal/utils"

	"github.com/gofiber/fiber/v2"
)

// UploadFile menangani POST /api/upload (Upload gambar dari Admin app dengan auto-resize & kompresi)
func UploadFile(c *fiber.Ctx) error {
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File gambar wajib diunggah (key: image)",
		})
	}

	// Validasi ekstensi file
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
	}

	if !allowedExts[ext] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Format file tidak didukung. Harap unggah file .jpg, .jpeg, .png, atau .webp",
		})
	}

	// Validasi ukuran file (maksimal 8MB untuk gambar mentah sebelum dikompresi otomatis)
	const maxFileSize = 8 * 1024 * 1024 // 8 MB
	if file.Size > maxFileSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Ukuran file terlalu besar. Maksimal ukuran gambar adalah 8 MB",
		})
	}

	// Simpan dan kompresi otomatis ke folder ./uploads (maksimal lebar 1600px & JPEG quality 82)
	uploadsDir := "./uploads"
	filename, err := utils.ProcessAndSaveImage(file, uploadsDir, "img")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal mengoptimasi dan menyimpan gambar di server: " + err.Error(),
		})
	}

	// Buat URL lengkap yang bisa diakses publik (mendukung HTTPS di balik reverse proxy Nginx/Cloudflare)
	proto := strings.TrimSpace(c.Get("X-Forwarded-Proto"))
	if proto == "" {
		proto = c.Protocol()
	}
	fileURL := fmt.Sprintf("%s://%s/uploads/%s", proto, c.Hostname(), filename)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":  "File gambar berhasil diunggah dan dioptimasi",
		"filename": filename,
		"url":      fileURL,
	})
}

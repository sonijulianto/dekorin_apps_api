package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// UploadFile menangani POST /api/upload (Upload gambar dari Admin app)
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

	// Buat direktori ./uploads jika belum ada
	uploadsDir := "./uploads"
	if err := os.MkdirAll(uploadsDir, os.ModePerm); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal membuat direktori upload di server",
		})
	}

	// Buat nama file unik
	filename := fmt.Sprintf("img_%d_%s%s", time.Now().Unix(), uuid.New().String()[:8], ext)
	savePath := filepath.Join(uploadsDir, filename)

	// Simpan file ke disk
	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Gagal menyimpan file gambar ke server",
		})
	}

	// Buat URL lengkap yang bisa diakses publik
	fileURL := fmt.Sprintf("%s://%s/uploads/%s", c.Protocol(), c.Hostname(), filename)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":  "File gambar berhasil diunggah",
		"filename": filename,
		"url":      fileURL,
	})
}

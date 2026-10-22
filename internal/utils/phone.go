package utils

import (
	"regexp"
	"strings"
)

// NormalizePhone membersihkan nomor HP ke format standar Indonesia (08...)
// Contoh:
// "+62812345678" -> "0812345678"
// "62812345678"  -> "0812345678"
// "0812-3456-789" -> "08123456789"
// "0812 3456 789" -> "08123456789"
func NormalizePhone(phone string) string {
	// Hapus semua karakter selain angka
	re := regexp.MustCompile(`[^0-9]`)
	cleaned := re.ReplaceAllString(phone, "")

	if strings.HasPrefix(cleaned, "62") {
		cleaned = "0" + strings.TrimPrefix(cleaned, "62")
	}

	return cleaned
}

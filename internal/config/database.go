package config

import (
	"fmt"
	"log"

	"dekorin_apps_api/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// ConnectDB membuka koneksi ke PostgreSQL
func ConnectDB() {
	// Credentials sesuai dengan input user
	dsn := "host=localhost user=magnatech password=123456 dbname=dekorin port=5432 sslmode=disable TimeZone=Asia/Jakarta"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal koneksi ke database PostgreSQL: \n", err)
	}

	fmt.Println("✅ Koneksi Database PostgreSQL berhasil!")
	DB = db

	// Menjalankan Auto Migration (membuat tabel users jika belum ada)
	err = DB.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatal("Gagal melakukan auto migrate: ", err)
	}

	// Menjalankan Seeding otomatis
	SeedAdmin(db)
}

// SeedAdmin memeriksa apakah tabel user kosong, jika ya buatkan admin default
func SeedAdmin(db *gorm.DB) {
	var count int64
	db.Model(&models.User{}).Count(&count)

	if count == 0 {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)

		admin := models.User{
			ID:       "usr_001",
			FullName: "Admin Dekorin",
			Email:    "admin@dekorin.com",
			Password: string(hashedPassword),
		}

		db.Create(&admin)
		fmt.Println("🌱 Seeding selesai: Default admin (admin@dekorin.com) berhasil dibuat ke dalam database!")
	}
}

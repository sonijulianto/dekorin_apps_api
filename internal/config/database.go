package config

import (
	"fmt"
	"log"
	"time"

	"dekorin_apps_api/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// ConnectDB membuka koneksi ke PostgreSQL
func ConnectDB() {
	// Credentials sesuai dengan input user
	dsn := "host=localhost user=postgres password=postgres dbname=dekorin port=5432 sslmode=disable TimeZone=Asia/Jakarta client_encoding=UTF8"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal koneksi ke database PostgreSQL: \n", err)
	}

	fmt.Println("✅ Koneksi Database PostgreSQL berhasil!")
	DB = db

	// Menjalankan Auto Migration (membuat tabel jika belum ada)
	err = DB.AutoMigrate(&models.User{}, &models.Package{}, &models.Agenda{}, &models.ClientForm{})
	if err != nil {
		log.Fatal("Gagal melakukan auto migrate: ", err)
	}

	// Menjalankan Seeding otomatis
	SeedAdmin(db)
	SeedPackagesAndAgendas(db)
}

// SeedPackagesAndAgendas mengisi data awal master paket dan agenda contoh jika kosong
func SeedPackagesAndAgendas(db *gorm.DB) {
	var pkgCount int64
	db.Model(&models.Package{}).Count(&pkgCount)

	if pkgCount == 0 {
		packages := []models.Package{
			{
				ID:          "pkg_01",
				Name:        "Paket Akad Minimalis",
				Description: "Backdrop 2.5m, rangkaian bunga artificial premium, 2 standing flower, karpet akad, set kursi pengantin",
				BasePrice:   2500000,
			},
			{
				ID:          "pkg_02",
				Name:        "Paket Lamaran Rustic / Modern",
				Description: "Backdrop 3m hexagon/arch, full bunga kombinasi, spotlight, welcome sign akrilik, ring box",
				BasePrice:   4000000,
			},
			{
				ID:          "pkg_03",
				Name:        "Paket Pernikahan Elegan Gold",
				Description: "Pelaminan 6m, mini gallery photo, pergola masuk, lighting ambience lengkap, karpet jalan",
				BasePrice:   9500000,
			},
			{
				ID:          "pkg_04",
				Name:        "Paket Grand Luxury Ballroom",
				Description: "Pelaminan 10-12m full fresh flower, chandelier, photobooth thematic, gate entrance eksklusif",
				BasePrice:   18000000,
			},
		}

		for _, p := range packages {
			db.Create(&p)
		}
		fmt.Println("🌱 Seeding selesai: Master paket dekorasi berhasil ditambahkan!")
	}

	var agdCount int64
	db.Model(&models.Agenda{}).Count(&agdCount)
	if agdCount == 0 {
		now := time.Now()
		sampleAgendas := []models.Agenda{
			{
				ID:            "agd_001",
				ClientName:    "Rian & Nisa",
				BackdropTitle: "The Engagement of Rian & Nisa",
				EventDateTime: now.Add(24 * time.Hour), // Besok
				MapsURL:       "https://maps.google.com/?q=-6.2088,106.8456",
				PackageID:     "pkg_02",
				PackageName:   "Paket Lamaran Rustic / Modern",
				Status:        "upcoming",
				Notes:         "Tema warna Sage Green & Cream. Pasang dekorasi H-3 jam sebelum acara.",
			},
			{
				ID:            "agd_002",
				ClientName:    "Dimas & Sarah",
				BackdropTitle: "Wedding Reception Dimas & Sarah",
				EventDateTime: now.Add(48 * time.Hour), // Lusa
				MapsURL:       "https://maps.google.com/?q=-6.9175,107.6191",
				PackageID:     "pkg_03",
				PackageName:   "Paket Pernikahan Elegan Gold",
				Status:        "upcoming",
				Notes:         "Gedung Puri Ardhya Garini, loading barang jam 23.00 malam sebelumnya.",
			},
			{
				ID:            "agd_003",
				ClientName:    "Keluarga Wijaya",
				BackdropTitle: "Syukuran Khitanan Fakhri",
				EventDateTime: now.Add(7 * 24 * time.Hour), // Minggu depan
				MapsURL:       "https://maps.google.com/?q=-6.2297,106.6894",
				PackageID:     "pkg_01",
				PackageName:   "Paket Akad Minimalis",
				Status:        "upcoming",
				Notes:         "Lokasi rumah, butuh kabel rol tambahan 10m.",
			},
		}

		for _, a := range sampleAgendas {
			db.Create(&a)
		}
		fmt.Println("🌱 Seeding selesai: Data sample agenda berhasil ditambahkan!")
	}
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

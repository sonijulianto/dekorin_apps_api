package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"dekorin_apps_api/internal/models"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// ConnectDB membuka koneksi ke PostgreSQL
func ConnectDB() {
	// Muat file .env jika ada (misal di local development)
	_ = godotenv.Load()

	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "postgres")
	dbname := getEnv("DB_NAME", "dekorin")
	sslmode := getEnv("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Jakarta client_encoding=UTF8",
		host, user, password, dbname, port, sslmode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Gagal koneksi ke database PostgreSQL: \n", err)
	}

	fmt.Println("✅ Koneksi Database PostgreSQL berhasil!")
	DB = db

	// Menjalankan Auto Migration (membuat tabel jika belum ada)
	err = DB.AutoMigrate(
		&models.Company{},
		&models.User{},
		&models.Package{},
		&models.Agenda{},
		&models.ClientForm{},
		&models.Addon{},
		&models.DecorationModel{},
		&models.PackageDecorationModel{},
		&models.Invitation{},
		&models.InvitationWish{},
		&models.Expense{},
		&models.ApiLog{},
	)
	if err != nil {
		log.Println("⚠️ Peringatan saat auto migrate: ", err)
	}

	// Hapus unique index lama pada kolom email (sisa skema awal: `email uniqueIndex`).
	// AutoMigrate tidak menghapus index lama, sehingga anggota tim tanpa email
	// (email = '') bentrok satu sama lain dan gagal disimpan.
	if DB.Migrator().HasIndex(&models.User{}, "idx_users_email") {
		if err := DB.Migrator().DropIndex(&models.User{}, "idx_users_email"); err != nil {
			log.Println("⚠️ Gagal menghapus unique index idx_users_email: ", err)
		} else {
			log.Println("✅ Unique index lama idx_users_email berhasil dihapus")
		}
	}

	// Menjalankan Seeding otomatis
	SeedCompany(db)
	SeedUsers(db)
	SeedPackagesAndAgendas(db)
	SeedDecorationModels(db)
	SeedAddons(db)
	SeedExpenses(db)
	SeedApiLogs(db)
}

// SeedAddons mengisi data awal master item tambahan jika kosong
func SeedAddons(db *gorm.DB) {
	var addonCount int64
	db.Model(&models.Addon{}).Count(&addonCount)

	if addonCount == 0 {
		addons := []models.Addon{
			{
				ID:          "adn_01",
				Name:        "Lighting Ambient & Spotlight Warm",
				ImageURL:    "https://images.unsplash.com/photo-1519741497674-611481863552?w=500&q=80",
				Price:       350000,
				Description: "Set lighting sorot dekorasi warm white untuk suasana hangat",
			},
			{
				ID:          "adn_02",
				Name:        "Photobooth Thematic + Properti",
				ImageURL:    "https://images.unsplash.com/photo-1527529482837-4698179dc6ce?w=500&q=80",
				Price:       850000,
				Description: "Backdrop foto tamu 2x2m fleksibel sesuai tema acara",
			},
			{
				ID:          "adn_03",
				Name:        "Standing Flower Fresh Premium (2 Pcs)",
				ImageURL:    "https://images.unsplash.com/photo-1561181286-d3fee7d55364?w=500&q=80",
				Price:       500000,
				Description: "Sepasang standing flower bunga segar pilihan di jalur pelaminan",
			},
			{
				ID:          "adn_04",
				Name:        "Welcome Sign Akrilik Custom Text",
				ImageURL:    "https://images.unsplash.com/photo-1511285560929-80b456fea0bc?w=500&q=80",
				Price:       250000,
				Description: "Papan ucapan selamat datang dari akrilik bening dengan ukiran nama",
			},
		}

		for _, a := range addons {
			db.Create(&a)
		}
		fmt.Println("🌱 Seeding selesai: Master item tambahan (Addons) berhasil ditambahkan!")
	}
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

// SeedCompany memastikan perusahaan default ada dan melakukan backfill data lama
func SeedCompany(db *gorm.DB) {
	var comp models.Company
	if err := db.Where("id = ?", "cmp_default").First(&comp).Error; err != nil {
		defaultCompany := models.Company{
			ID:      "cmp_default",
			Name:    "Dekorin Wedding & Event Decoration",
			Address: "Jl. Melati Raya No. 123, Jakarta Selatan",
			Phone:   "081234567890",
		}
		db.Create(&defaultCompany)
		fmt.Println("🌱 Seeding: Perusahaan default berhasil dibuat!")
	}

	// Backfill company_id untuk entitas yang masih kosong
	db.Model(&models.User{}).Where("company_id = '' OR company_id IS NULL").Where("role != ?", "superadmin").Update("company_id", "cmp_default")
	db.Model(&models.Package{}).Where("company_id = '' OR company_id IS NULL").Update("company_id", "cmp_default")
	db.Model(&models.Agenda{}).Where("company_id = '' OR company_id IS NULL").Update("company_id", "cmp_default")
	db.Model(&models.Addon{}).Where("company_id = '' OR company_id IS NULL").Update("company_id", "cmp_default")
	db.Model(&models.DecorationModel{}).Where("company_id = '' OR company_id IS NULL").Update("company_id", "cmp_default")
	db.Model(&models.PackageDecorationModel{}).Where("company_id = '' OR company_id IS NULL").Update("company_id", "cmp_default")
	db.Model(&models.Expense{}).Where("company_id = '' OR company_id IS NULL").Update("company_id", "cmp_default")
	db.Model(&models.Invitation{}).Where("company_id = '' OR company_id IS NULL").Update("company_id", "cmp_default")
	db.Model(&models.ApiLog{}).Where("company_id = '' OR company_id IS NULL").Update("company_id", "cmp_default")
}

// SeedDecorationModels mengisi data awal master model dekorasi jika kosong dan menautkannya ke paket
func SeedDecorationModels(db *gorm.DB) {
	var modelCount int64
	db.Model(&models.DecorationModel{}).Count(&modelCount)

	if modelCount == 0 {
		modelsList := []models.DecorationModel{
			{
				ID:          "mdl_01",
				CompanyID:   "cmp_default",
				Name:        "Model Rustic Arch Wood",
				ImageURL:    "https://images.unsplash.com/photo-1519741497674-611481863552?w=500&q=80",
				Price:       1500000,
				Width:       "3 Meter",
				Description: "Backdrop arch kayu natural dengan sentuhan bunga pampas kering & eucalyptus",
				SortOrder:   1,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				ID:          "mdl_02",
				CompanyID:   "cmp_default",
				Name:        "Model Modern Hexagon Gold",
				ImageURL:    "https://images.unsplash.com/photo-1511285560929-80b456fea0bc?w=500&q=80",
				Price:       2500000,
				Width:       "4 Meter",
				Description: "Frame geometri heksagonal finishing gold metallic dengan rangkaian bunga kombinasi",
				SortOrder:   2,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				ID:          "mdl_03",
				CompanyID:   "cmp_default",
				Name:        "Model Classic White Floral Panel",
				ImageURL:    "https://images.unsplash.com/photo-1465495976277-4387d4b0b4c6?w=500&q=80",
				Price:       4000000,
				Width:       "6 Meter",
				Description: "Pelaminan panel ukir broken white elegan dengan lighting sorot & karpet jalan",
				SortOrder:   3,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				ID:          "mdl_04",
				CompanyID:   "cmp_default",
				Name:        "Model Grand Luxury Botanical Arch",
				ImageURL:    "https://images.unsplash.com/photo-1527529482837-4698179dc6ce?w=500&q=80",
				Price:       7500000,
				Width:       "10 Meter",
				Description: "Pelaminan ballroom megah full bunga segar pilihan, chandelier gantung & panggung pelaminan",
				SortOrder:   4,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		}

		for _, m := range modelsList {
			db.Create(&m)
		}
		fmt.Println("🌱 Seeding selesai: Master Model Dekorasi berhasil ditambahkan!")
	}

	// Hubungkan contoh model ke paket jika belum ada
	var packageModelCount int64
	db.Model(&models.PackageDecorationModel{}).Count(&packageModelCount)
	if packageModelCount == 0 {
		samplePackageModels := []models.PackageDecorationModel{
			// Paket Lamaran Rustic (pkg_02): Model Rustic Arch Wood (0) dan Model Modern Hexagon (+200rb)
			{
				ID:                "pm_01",
				CompanyID:         "cmp_default",
				PackageID:         "pkg_02",
				DecorationModelID: "mdl_01",
				AdditionalPrice:   0,
				SortOrder:         1,
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
			},
			{
				ID:                "pm_02",
				CompanyID:         "cmp_default",
				PackageID:         "pkg_02",
				DecorationModelID: "mdl_02",
				AdditionalPrice:   200000,
				SortOrder:         2,
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
			},
			// Paket Pernikahan Elegan Gold (pkg_03): Model Modern Hexagon (0) dan Model Classic Panel (+500rb)
			{
				ID:                "pm_03",
				CompanyID:         "cmp_default",
				PackageID:         "pkg_03",
				DecorationModelID: "mdl_02",
				AdditionalPrice:   0,
				SortOrder:         1,
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
			},
			{
				ID:                "pm_04",
				CompanyID:         "cmp_default",
				PackageID:         "pkg_03",
				DecorationModelID: "mdl_03",
				AdditionalPrice:   500000,
				SortOrder:         2,
				CreatedAt:         time.Now(),
				UpdatedAt:         time.Now(),
			},
		}
		for _, pm := range samplePackageModels {
			db.Create(&pm)
		}
		fmt.Println("🌱 Seeding selesai: Hubungan Paket Dekorasi & Model Dekorasi berhasil ditambahkan!")
	}
}

// SeedUsers memastikan akun default untuk setiap role (superadmin, owner, admin, crew) tersedia di database
func SeedUsers(db *gorm.DB) {
	defaultUsers := []models.User{
		{
			ID:        "usr_superadmin",
			CompanyID: "cmp_default",
			FullName:  "Superadmin Dekorin",
			Phone:     "081111111111",
			Email:     "superadmin@dekorin.com",
			Password:  "superadmin123",
			Role:      "superadmin",
		},
		{
			ID:        "usr_owner",
			CompanyID: "cmp_default",
			FullName:  "Owner Dekorin",
			Phone:     "081234567890",
			Email:     "owner@dekorin.com",
			Password:  "owner123",
			Role:      "owner",
		},
		{
			ID:        "usr_admin",
			CompanyID: "cmp_default",
			FullName:  "Admin Dekorin",
			Phone:     "081234567891",
			Email:     "admin@dekorin.com",
			Password:  "admin123",
			Role:      "admin",
		},
		{
			ID:        "usr_crew",
			CompanyID: "cmp_default",
			FullName:  "Crew Lapangan",
			Phone:     "081234567892",
			Email:     "crew@dekorin.com",
			Password:  "crew123",
			Role:      "crew",
		},
	}

	for _, u := range defaultUsers {
		var existing models.User
		if err := db.Where("phone = ? OR email = ? OR id = ?", u.Phone, u.Email, u.ID).First(&existing).Error; err != nil {
			// User belum ada, buat user baru
			hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
			u.Password = string(hashedPassword)
			if createErr := db.Create(&u).Error; createErr != nil {
				fmt.Printf("⚠️ Gagal membuat user %s: %v\n", u.FullName, createErr)
			} else {
				fmt.Printf("🌱 Seeding: Akun %s (No HP: %s - Role: %s) berhasil dibuat!\n", u.FullName, u.Phone, u.Role)
			}
		} else {
			// User sudah ada, pastikan phone, role, dan company_id sinkron jika masih kosong/berbeda
			updates := map[string]interface{}{}
			if existing.Phone == "" {
				updates["phone"] = u.Phone
			}
			if existing.Role != u.Role {
				updates["role"] = u.Role
			}
			if existing.CompanyID == "" {
				updates["company_id"] = u.CompanyID
			}
			if len(updates) > 0 {
				db.Model(&existing).Updates(updates)
				fmt.Printf("🌱 Seeding: Sinkronisasi akun %s (No HP: %s) berhasil diperbarui!\n", u.FullName, u.Phone)
			}
		}
	}
}

// SeedExpenses mengisi sample pengeluaran operasional dekorasi jika tabel masih kosong
func SeedExpenses(db *gorm.DB) {
	var count int64
	db.Model(&models.Expense{}).Count(&count)
	if count == 0 {
		now := time.Now()
		sampleExpenses := []models.Expense{
			{
				ID:        "exp_001",
				Title:     "Pembelian Bunga Mawar & Baby Breath Fresh",
				Amount:    1250000,
				Category:  "Bunga",
				Date:      now.Add(-24 * time.Hour),
				Notes:     "Pasar Rawa Belong, 15 ikat mawar dan 5 ikat baby breath",
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				ID:        "exp_002",
				Title:     "Kain Backdrop & Lampu Sorot Warm",
				Amount:    850000,
				Category:  "Properti Dekor",
				Date:      now.Add(-48 * time.Hour),
				Notes:     "Bahan kain velvet broken white 20m dan 2 spotlight",
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				ID:        "exp_003",
				Title:     "Transport Pick-up & Bongkar Muat",
				Amount:    450000,
				Category:  "Operasional",
				Date:      now.Add(-72 * time.Hour),
				Notes:     "Sewa mobil pick up operasional round-trip",
				CreatedAt: now,
				UpdatedAt: now,
			},
		}
		for _, exp := range sampleExpenses {
			db.Create(&exp)
		}
		fmt.Println("🌱 Seeding selesai: Data sample pengeluaran dekorasi berhasil ditambahkan!")
	}

	// Pastikan agenda yang sudah ada memiliki payment_status terisi
	db.Model(&models.Agenda{}).Where("payment_status IS NULL OR payment_status = ''").Updates(map[string]interface{}{
		"payment_status": "unpaid",
	})
	// Jika ada agenda dengan ID agd_001, jadikan sample yang sudah lunas dan selesai agar dashboard menampilkan omset
	var firstAgenda models.Agenda
	if err := db.Where("id = ?", "agd_001").First(&firstAgenda).Error; err == nil {
		if firstAgenda.PaymentStatus == "unpaid" || firstAgenda.PaymentStatus == "" {
			db.Model(&firstAgenda).Updates(map[string]interface{}{
				"payment_status": "paid",
				"package_price":  4000000,
				"transport_fee":  250000,
				"dp_amount":      4250000,
			})
		}
	}
}

// SeedApiLogs mengisi sample data log API awal sesuai screenshot
func SeedApiLogs(db *gorm.DB) {
	var count int64
	db.Model(&models.ApiLog{}).Count(&count)
	if count == 0 {
		now := time.Now()
		// Format sample response data sesuai Screenshot 2
		sampleSyncResp := `{
  "status_code": 200,
  "success": true,
  "message": "Successfully",
  "data": {
    "id": "eb005e2a-697b-473e-af55-83101121883a",
    "url": "/api/sync/upload",
    "method": "POST",
    "ip_addr": "111.94.110.66",
    "username": "pagsls001",
    "log_time": "2026-10-22 18:57:23",
    "req_data": "{\"STOCKS\":[],\"ADJUSTMENTS\":[]}",
    "response_data": "{\"status_code\":200,\"success\":true}",
    "status": "SUCCESS",
    "status_code": "200",
    "remark": "Synchronization has been successfully.",
    "apk_version": "25.11.17148",
    "device_model": "onc",
    "build_date": null,
    "name": "M Refli Devano A",
    "branch_code": "JT",
    "branch_name": "JAKARTA TIMUR"
  }
}`

		sampleSyncReq := `{
  "id": "eb005e2a-697b-473e-af55-83101121883a"
}`

		sampleLogs := []models.ApiLog{
			{
				ID:           "log_01",
				CompanyID:    "cmp_default",
				Method:       "GET",
				Status:       "SUCCESS",
				StatusCode:   200,
				URL:          "/api/apilog/get",
				IPAddr:       "111.94.110.66",
				Username:     "super_soni",
				LogTime:      now,
				ReqData:      "{\n  \"date\": \"22/10/2026\",\n  \"page\": 1,\n  \"limit\": 10\n}",
				ResponseData: "{\n  \"status_code\": 200,\n  \"success\": true,\n  \"message\": \"Successfully\",\n  \"total\": 275695\n}",
				CreatedAt:    now,
			},
			{
				ID:           "log_02",
				CompanyID:    "cmp_default",
				Method:       "GET",
				Status:       "SUCCESS",
				StatusCode:   200,
				URL:          "/api/apk/check-version",
				IPAddr:       "180.252.164.21",
				Username:     "",
				LogTime:      now.Add(-2 * time.Minute),
				ReqData:      "{}",
				ResponseData: "{\n  \"status_code\": 200,\n  \"success\": true,\n  \"data\": {\n    \"version\": \"25.11.17148\",\n    \"force_update\": false\n  }\n}",
				CreatedAt:    now.Add(-2 * time.Minute),
			},
			{
				ID:           "eb005e2a-697b-473e-af55-83101121883a",
				CompanyID:    "cmp_default",
				Method:       "POST",
				Status:       "SUCCESS",
				StatusCode:   200,
				URL:          "/api/sync/upload",
				IPAddr:       "111.94.110.66",
				Username:     "pagsls001",
				LogTime:      now.Add(-4 * time.Minute),
				ReqData:      sampleSyncReq,
				ResponseData: sampleSyncResp,
				Remark:       "Synchronization has been successfully.",
				ApkVersion:   "25.11.17148",
				DeviceModel:  "onc",
				Name:         "M Refli Devano A",
				BranchCode:   "JT",
				BranchName:   "JAKARTA TIMUR",
				CreatedAt:    now.Add(-4 * time.Minute),
			},
			{
				ID:           "log_04",
				CompanyID:    "cmp_default",
				Method:       "GET",
				Status:       "SUCCESS",
				StatusCode:   200,
				URL:          "/api/visit/get-visit-status",
				IPAddr:       "111.94.110.66",
				Username:     "pagsls001",
				LogTime:      now.Add(-4 * time.Minute - 30*time.Second),
				ReqData:      "{\n  \"user_id\": \"pagsls001\"\n}",
				ResponseData: "{\n  \"status_code\": 200,\n  \"success\": true,\n  \"message\": \"Successfully\"\n}",
				CreatedAt:    now.Add(-4 * time.Minute - 30*time.Second),
			},
			{
				ID:           "log_05",
				CompanyID:    "cmp_default",
				Method:       "GET",
				Status:       "SUCCESS",
				StatusCode:   200,
				URL:          "/api/apk/check-version",
				IPAddr:       "36.72.215.11",
				Username:     "",
				LogTime:      now.Add(-5 * time.Minute),
				ReqData:      "{}",
				ResponseData: "{\n  \"status_code\": 200,\n  \"success\": true\n}",
				CreatedAt:    now.Add(-5 * time.Minute),
			},
			{
				ID:           "log_06",
				CompanyID:    "cmp_default",
				Method:       "GET",
				Status:       "SUCCESS",
				StatusCode:   200,
				URL:          "/api/apk/check-version",
				IPAddr:       "36.72.215.11",
				Username:     "",
				LogTime:      now.Add(-11 * time.Minute),
				ReqData:      "{}",
				ResponseData: "{\n  \"status_code\": 200,\n  \"success\": true\n}",
				CreatedAt:    now.Add(-11 * time.Minute),
			},
			{
				ID:           "log_07",
				CompanyID:    "cmp_default",
				Method:       "GET",
				Status:       "SUCCESS",
				StatusCode:   200,
				URL:          "/api/apk/check-version",
				IPAddr:       "180.252.164.21",
				Username:     "",
				LogTime:      now.Add(-13 * time.Minute),
				ReqData:      "{}",
				ResponseData: "{\n  \"status_code\": 200,\n  \"success\": true\n}",
				CreatedAt:    now.Add(-13 * time.Minute),
			},
			{
				ID:           "log_08",
				CompanyID:    "cmp_default",
				Method:       "GET",
				Status:       "SUCCESS",
				StatusCode:   200,
				URL:          "/api/dashboard",
				IPAddr:       "114.124.238.10",
				Username:     "pagadm04",
				LogTime:      now.Add(-14 * time.Minute),
				ReqData:      "{}",
				ResponseData: "{\n  \"status_code\": 200,\n  \"success\": true,\n  \"data\": {\n    \"total_agendas\": 18,\n    \"revenue\": 145000000\n  }\n}",
				CreatedAt:    now.Add(-14 * time.Minute),
			},
			{
				ID:           "log_09",
				CompanyID:    "cmp_default",
				Method:       "GET",
				Status:       "SUCCESS",
				StatusCode:   200,
				URL:          "/api/apk/check-version",
				IPAddr:       "36.72.215.11",
				Username:     "",
				LogTime:      now.Add(-23 * time.Minute),
				ReqData:      "{}",
				ResponseData: "{\n  \"status_code\": 200,\n  \"success\": true\n}",
				CreatedAt:    now.Add(-23 * time.Minute),
			},
			{
				ID:           "log_10",
				CompanyID:    "cmp_default",
				Method:       "GET",
				Status:       "SUCCESS",
				StatusCode:   200,
				URL:          "/api/apk/check-version",
				IPAddr:       "180.252.164.21",
				Username:     "",
				LogTime:      now.Add(-24 * time.Minute),
				ReqData:      "{}",
				ResponseData: "{\n  \"status_code\": 200,\n  \"success\": true\n}",
				CreatedAt:    now.Add(-24 * time.Minute),
			},
			{
				ID:           "log_11",
				CompanyID:    "cmp_default",
				Method:       "POST",
				Status:       "SUCCESS",
				StatusCode:   200,
				URL:          "/api/packages",
				IPAddr:       "111.94.110.66",
				Username:     "super_soni",
				LogTime:      now.Add(-30 * time.Minute),
				ReqData:      "{\n  \"name\": \"Paket Platinum VIP\",\n  \"base_price\": 12000000\n}",
				ResponseData: "{\n  \"status_code\": 200,\n  \"message\": \"Paket dekorasi berhasil dibuat\"\n}",
				CreatedAt:    now.Add(-30 * time.Minute),
			},
			{
				ID:           "log_12",
				CompanyID:    "cmp_default",
				Method:       "POST",
				Status:       "FAILED",
				StatusCode:   400,
				URL:          "/api/agendas",
				IPAddr:       "111.94.110.66",
				Username:     "pagadm04",
				LogTime:      now.Add(-35 * time.Minute),
				ReqData:      "{\n  \"client_name\": \"\"\n}",
				ResponseData: "{\n  \"status_code\": 400,\n  \"error\": \"Nama client wajib diisi\"\n}",
				CreatedAt:    now.Add(-35 * time.Minute),
			},
		}

		for _, l := range sampleLogs {
			db.Create(&l)
		}
		fmt.Println("🌱 Seeding selesai: Data sample API logs berhasil ditambahkan!")
	}
}


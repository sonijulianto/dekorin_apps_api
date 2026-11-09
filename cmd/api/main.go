package main

import (
	"log"
	"os"
	"strings"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/handlers"
	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	// Koneksi ke Database PostgreSQL
	config.ConnectDB()

	// Menjalankan background worker untuk pembersihan disk & log lama secara otomatis
	startAutoCleanWorker()

	// Initialize Fiber app dengan limit body 150MB untuk upload file APK & gambar
	app := fiber.New(fiber.Config{
		BodyLimit: 150 * 1024 * 1024, // 150 MB
	})

	// Middleware
	app.Use(logger.New()) // Log every request
	app.Use(cors.New())   // Enable CORS for all origins
	app.Use(middleware.ApiLogger()) // Automatic API Request & Response Logger

	// Helper untuk mendeteksi IP klien secara akurat (termasuk di balik reverse proxy Nginx)
	clientIP := func(c *fiber.Ctx) string {
		if xff := c.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[0])
		}
		if xRealIP := c.Get("X-Real-IP"); xRealIP != "" {
			return xRealIP
		}
		return c.IP()
	}

	// Rate Limiter anti brute-force untuk endpoint login (maksimal 10 request per menit per IP)
	loginLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Minute,
		KeyGenerator: clientIP,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"status":  "error",
				"message": "Terlalu banyak percobaan login. Silakan tunggu 1 menit sebelum mencoba kembali.",
			})
		},
	})

	// Rate Limiter untuk form booking klien (maksimal 30 request per menit per IP)
	formLimiter := limiter.New(limiter.Config{
		Max:        30,
		Expiration: 1 * time.Minute,
		KeyGenerator: clientIP,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"status":  "error",
				"message": "Terlalu banyak permintaan ke formulir klien. Silakan coba beberapa saat lagi.",
			})
		},
	})

	// Rate Limiter untuk pengiriman ucapan / RSVP tamu (maksimal 15 ucapan per menit per IP anti-spam)
	wishLimiter := limiter.New(limiter.Config{
		Max:          15,
		Expiration:   1 * time.Minute,
		KeyGenerator: clientIP,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Terlalu banyak pengiriman ucapan dalam waktu singkat. Harap tunggu sebentar.",
			})
		},
	})

	// Group routes
	api := app.Group("/api")
	auth := api.Group("/auth")

	// Routes
	auth.Post("/login", loginLimiter, handlers.Login)
	auth.Post("/change-password", middleware.Protected(), handlers.ChangePassword)

	// Superadmin / Web Admin routes (dapat diakses superadmin, owner, dan admin)
	superadmin := api.Group("/superadmin")
	superadmin.Post("/login", loginLimiter, handlers.SuperadminLogin)
	superadmin.Get("/me", middleware.Protected(), handlers.GetCurrentUserProfile)
	superadmin.Get("/dashboard-stats", middleware.Protected(), middleware.RequireRole("superadmin", "owner", "admin"), handlers.GetSuperadminDashboardStats)
	superadmin.Get("/companies", middleware.Protected(), middleware.RequireRole("superadmin", "owner", "admin"), handlers.GetAllCompanies)
	superadmin.Post("/companies", middleware.Protected(), middleware.RequireSuperadmin(), handlers.CreateCompanyWithInitialOwner)
	superadmin.Put("/companies/:id", middleware.Protected(), middleware.RequireRole("superadmin", "owner", "admin"), handlers.UpdateCompany)
	superadmin.Delete("/companies/:id", middleware.Protected(), middleware.RequireSuperadmin(), handlers.DeleteCompany)

	// Company profile routes (Data profil badan usaha masing-masing vendor dekorasi)
	api.Get("/company/profile", middleware.Protected(), handlers.GetMyCompanyProfile)
	api.Put("/company/profile", middleware.Protected(), middleware.RequireRole("owner", "admin"), handlers.UpdateMyCompanyProfile)

	// Company Brand (Sub-Brand / Multi-Brand) routes
	api.Get("/company/brands", middleware.Protected(), handlers.GetCompanyBrands)
	api.Post("/company/brands", middleware.Protected(), middleware.RequireRole("owner", "admin"), handlers.CreateCompanyBrand)
	api.Put("/company/brands/:id", middleware.Protected(), middleware.RequireRole("owner", "admin"), handlers.UpdateCompanyBrand)
	api.Delete("/company/brands/:id", middleware.Protected(), middleware.RequireRole("owner", "admin"), handlers.DeleteCompanyBrand)

	// User routes
	api.Get("/users", middleware.Protected(), handlers.GetAllUsers)
	api.Post("/users", middleware.Protected(), handlers.CreateUser)
	api.Put("/users/:id", middleware.Protected(), handlers.UpdateUser)
	api.Delete("/users/:id", middleware.Protected(), handlers.DeleteUser)

	// Dashboard summary route
	api.Get("/dashboard/summary", middleware.Protected(), handlers.GetDashboardSummary)

	// Agenda routes
	api.Get("/agendas", middleware.Protected(), handlers.GetAgendas)
	api.Post("/agendas", middleware.Protected(), handlers.CreateAgenda)
	api.Put("/agendas/:id", middleware.Protected(), handlers.UpdateAgenda)
	api.Put("/agendas/:id/schedule", middleware.Protected(), handlers.UpdateAgendaSchedule)
	api.Put("/agendas/:id/location", middleware.Protected(), handlers.UpdateAgendaLocation)
	api.Put("/agendas/:id/dp", middleware.Protected(), handlers.UpdateAgendaDp)
	api.Put("/agendas/:id/transport", middleware.Protected(), handlers.UpdateAgendaTransport)
	api.Put("/agendas/:id/package", middleware.Protected(), handlers.UpdateAgendaPackage)
	api.Put("/agendas/:id/model", middleware.Protected(), handlers.UpdateAgendaModel)
	api.Put("/agendas/:id/notes", middleware.Protected(), handlers.UpdateAgendaNotes)
	api.Put("/agendas/:id/payment-status", middleware.Protected(), handlers.UpdateAgendaPaymentStatus)
	api.Put("/agendas/:id/status", middleware.Protected(), handlers.UpdateAgendaStatus)
	api.Put("/agendas/:id/brand", middleware.Protected(), handlers.UpdateAgendaBrand)
	api.Delete("/agendas/:id", middleware.Protected(), handlers.DeleteAgenda)

	// Expense routes
	api.Get("/expenses", middleware.Protected(), handlers.GetExpenses)
	api.Post("/expenses", middleware.Protected(), handlers.CreateExpense)
	api.Delete("/expenses/:id", middleware.Protected(), handlers.DeleteExpense)

	// AI routes (Smart OCR, Gemini Assistant & Decor Color Simulation)
	api.Post("/ai/scan-receipt", middleware.Protected(), handlers.ScanReceiptAI)
	api.Post("/ai/assistant", middleware.Protected(), handlers.HandleAIAssistant)
	api.Post("/ai/simulate-decor", middleware.Protected(), handlers.SimulateDecorAI)

	// Master Package routes
	api.Get("/packages", middleware.Protected(), handlers.GetPackages)
	api.Post("/packages", middleware.Protected(), handlers.CreatePackage)
	api.Put("/packages/reorder", middleware.Protected(), handlers.ReorderPackages)
	api.Put("/packages/:id", middleware.Protected(), handlers.UpdatePackage)
	api.Delete("/packages/:id", middleware.Protected(), handlers.DeletePackage)

	// Master Addon routes
	api.Get("/addons", middleware.Protected(), handlers.GetAddons)
	api.Post("/addons", middleware.Protected(), handlers.CreateAddon)
	api.Put("/addons/reorder", middleware.Protected(), handlers.ReorderAddons)
	api.Put("/addons/:id", middleware.Protected(), handlers.UpdateAddon)
	api.Delete("/addons/:id", middleware.Protected(), handlers.DeleteAddon)

	// Master Decoration Model routes
	api.Get("/decoration-models", middleware.Protected(), handlers.GetDecorationModels)
	api.Post("/decoration-models", middleware.Protected(), handlers.CreateDecorationModel)
	api.Put("/decoration-models/reorder", middleware.Protected(), handlers.ReorderDecorationModels)
	api.Put("/decoration-models/:id", middleware.Protected(), handlers.UpdateDecorationModel)
	api.Delete("/decoration-models/:id", middleware.Protected(), handlers.DeleteDecorationModel)

	// Master Properties & Tools routes (Inventaris Properti Dekorasi & Alat Kerja)
	api.Get("/properties", middleware.Protected(), handlers.GetProperties)
	api.Get("/properties/availability", middleware.Protected(), handlers.GetPropertiesAvailability)
	api.Post("/properties", middleware.Protected(), handlers.CreateProperty)
	api.Put("/properties/:id", middleware.Protected(), handlers.UpdateProperty)
	api.Delete("/properties/:id", middleware.Protected(), handlers.DeleteProperty)

	// Agenda Pick List routes (Paperless Checklist Muat & Bongkar)
	api.Get("/agendas/:id/properties", middleware.Protected(), handlers.GetAgendaProperties)
	api.Post("/agendas/:id/properties/populate", middleware.Protected(), handlers.PopulateAgendaProperties)
	api.Post("/agendas/:id/properties/batch", middleware.Protected(), handlers.BatchAddAgendaProperties)
	api.Delete("/agendas/:id/properties/clear", middleware.Protected(), handlers.ClearAgendaProperties)
	api.Post("/agendas/:id/properties", middleware.Protected(), handlers.AddAgendaProperty)
	api.Put("/agendas/:id/properties/:item_id", middleware.Protected(), handlers.UpdateAgendaPropertyItem)
	api.Patch("/agendas/:id/properties/:item_id/pack", middleware.Protected(), handlers.ToggleAgendaPropertyPacked)
	api.Patch("/agendas/:id/properties/:item_id/return", middleware.Protected(), handlers.ToggleAgendaPropertyReturned)
	api.Delete("/agendas/:id/properties/:item_id", middleware.Protected(), handlers.DeleteAgendaPropertyItem)

	// API Logs routes (Hanya dapat diakses oleh Superadmin)
	api.Get("/apilog/get", middleware.Protected(), middleware.RequireSuperadmin(), handlers.GetApiLogs)
	api.Get("/apilog/:id", middleware.Protected(), middleware.RequireSuperadmin(), handlers.GetApiLogByID)
	api.Get("/apilogs", middleware.Protected(), middleware.RequireSuperadmin(), handlers.GetApiLogs)

	// App Version & In-App Update routes (Hanya Superadmin yang berhak upload & atur versi)
	api.Get("/app/version", handlers.GetLatestAppVersion)
	api.Get("/apk/check-version", handlers.GetLatestAppVersion)
	api.Get("/app/download-latest", handlers.DownloadLatestApk)
	app.Get("/download/apk", handlers.DownloadLatestApk)
	api.Get("/app/version-history", middleware.Protected(), middleware.RequireSuperadmin(), handlers.GetAppVersionHistory)
	api.Post("/app/version", middleware.Protected(), middleware.RequireSuperadmin(), handlers.UpdateAppVersion)
	api.Post("/app/upload-apk", middleware.Protected(), middleware.RequireSuperadmin(), handlers.UploadApkFile)
	api.Patch("/app/version/:id/toggle-force", middleware.Protected(), middleware.RequireSuperadmin(), handlers.ToggleForceUpdate)
	api.Delete("/app/version/:id", middleware.Protected(), middleware.RequireSuperadmin(), handlers.DeleteAppVersion)

	// Serve static upload directory dengan Cache-Control, Kompresi, dan ByteRange (Streaming Audio)
	app.Static("/uploads", "./uploads", fiber.Static{
		Compress:  true,
		ByteRange: true,
		Browse:    false,
		MaxAge:    86400 * 30, // 30 hari cache browser & Cloudflare CDN untuk foto & musik
	})
	app.Static("/assets", "./web/assets", fiber.Static{
		Compress:  true,
		ByteRange: true,
		Browse:    false,
		MaxAge:    86400 * 7, // 7 hari cache untuk template static assets
	})
	app.Static("/images", "./web/assets/images", fiber.Static{
		Compress:  true,
		ByteRange: true,
		Browse:    false,
		MaxAge:    86400 * 7,
	})

	// Direct Favicon handlers for root domain queries with cache busting headers
	app.Get("/favicon.ico", func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate, max-age=0")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		return c.SendFile("./web/assets/dekorin-favicon.ico")
	})
	app.Get("/favicon.png", func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate, max-age=0")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		return c.SendFile("./web/assets/dekorin-favicon.png")
	})
	app.Get("/apple-touch-icon.png", func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate, max-age=0")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		return c.SendFile("./web/assets/dekorin-apple-touch-icon.png")
	})

	// Upload route (Wajib autentikasi)
	api.Post("/upload", middleware.Protected(), handlers.UploadFile)

	// Landing Page route (atau Portal Pengantin jika diakses dari subdomain undangan.dekorin.my.id)
	app.Get("/", func(c *fiber.Ctx) error {
		if handlers.IsInvitationSubdomain(c) {
			return handlers.ShowInvitationPortalPage(c)
		}
		return c.SendFile("./web/templates/landing.html")
	})

	// Web Superadmin Dashboard route
	app.Get("/superadmin", func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		return c.SendFile("./web/templates/superadmin.html")
	})

	// Client Web Form routes (dengan rate limit untuk mencegah brute force token)
	app.Get("/form/:token", formLimiter, handlers.ShowClientForm)             // Public HTML form view
	api.Post("/client-form/:token", formLimiter, handlers.SubmitClientForm)   // Submit form API
	api.Get("/agendas/:id/client-form", middleware.Protected(), handlers.GetClientFormByAgendaID) // Fetch submitted form for Flutter app

	// Rate Limiter khusus Portal Kelola Undangan Pengantin (90 req/menit agar upload banyak foto galeri lancar)
	portalLimiter := limiter.New(limiter.Config{
		Max:          90,
		Expiration:   1 * time.Minute,
		KeyGenerator: clientIP,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Terlalu banyak permintaan dalam waktu singkat. Silakan tunggu beberapa detik.",
			})
		},
	})

	// Portal Mandiri Pengantin / Tuan Rumah Acara (undangan.dekorin.my.id)
	app.Get("/login", handlers.ShowInvitationPortalPage)
	app.Get("/kelola-undangan", handlers.ShowInvitationPortalPage)
	app.Get("/kelola", handlers.ShowInvitationPortalPage)
	app.Get("/kelola/:token", portalLimiter, handlers.ShowInvitationPortalPage)
	app.Get("/manage", handlers.ShowInvitationPortalPage)
	app.Get("/manage/:token", portalLimiter, handlers.ShowInvitationPortalPage)

	portalApi := api.Group("/invitation-portal")
	portalApi.Post("/login", loginLimiter, handlers.PortalLogin)
	portalApi.Get("/:token", portalLimiter, handlers.GetPortalInvitation)
	portalApi.Put("/:token", portalLimiter, handlers.UpdatePortalInvitation)
	portalApi.Put("/:token/guests", portalLimiter, handlers.SavePortalGuestList)
	portalApi.Post("/:token/upload", portalLimiter, handlers.UploadPortalMedia)
	portalApi.Get("/:token/export-csv", portalLimiter, handlers.ExportPortalWishesCSV)

	// Digital Invitation routes
	app.Get("/invitation/preview/:template", handlers.ShowTemplatePreview)
	app.Get("/preview/:template", handlers.ShowTemplatePreview)
	app.Get("/invitation/:slug", handlers.ShowPublicInvitation) // Public HTML wedding invitation
	api.Get("/invitations", middleware.Protected(), handlers.GetInvitations)
	api.Post("/invitations", middleware.Protected(), handlers.CreateInvitation)
	api.Get("/invitations/:id", middleware.Protected(), handlers.GetInvitationByID)
	api.Put("/invitations/:id", middleware.Protected(), handlers.UpdateInvitation)
	api.Post("/invitations/:id/reset-access", middleware.Protected(), handlers.ResetInvitationAccess)
	api.Delete("/invitations/:id", middleware.Protected(), handlers.DeleteInvitation)
	api.Get("/invitations/:id/wishes", handlers.GetInvitationWishes)
	api.Post("/invitations/:slug/wishes", wishLimiter, handlers.SubmitInvitationWish)

	// Short-URL undangan publik khusus subdomain undangan.dekorin.my.id/:slug (wajib paling akhir)
	app.Get("/:slug", handlers.ShowSubdomainPublicInvitation)

	// Start server (default port 3000 or from environment variable)
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Printf("Server started at http://localhost:%s\n", port)
	log.Fatal(app.Listen(":" + port))
}

// startAutoCleanWorker menjalankan pembersihan berkala otomatis di latar belakang
func startAutoCleanWorker() {
	go func() {
		// Jalankan sekali saat startup server
		runLogCleanup()

		// Jadwalkan pembersihan setiap 24 jam
		ticker := time.NewTicker(24 * time.Hour)
		for range ticker.C {
			runLogCleanup()
		}
	}()
}

func runLogCleanup() {
	cutoff := time.Now().AddDate(0, 0, -30) // Log lebih lama dari 30 hari
	res := config.DB.Where("created_at < ?", cutoff).Delete(&models.ApiLog{})
	if res.Error == nil && res.RowsAffected > 0 {
		log.Printf("🧹 [AutoClean] Berhasil membersihkan %d log API lama (> 30 hari)\n", res.RowsAffected)
	}
}

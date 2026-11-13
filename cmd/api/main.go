package main

import (
	"log"
	"os"
	"path/filepath"
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
	auth.Post("/register-couple", loginLimiter, handlers.RegisterCouple)
	auth.Post("/change-password", middleware.Protected(), handlers.ChangePassword)

	// Direktori Vendor Publik (dapat diakses oleh Calon Pengantin maupun Vendor terautentikasi)
	api.Get("/vendors/directory", middleware.Protected(), handlers.GetPublicVendors)
	api.Get("/vendors/directory/:id", middleware.Protected(), handlers.GetPublicVendorDetail)

	// Couple / Calon Pengantin routes (Wedding Plan, Vendor & Budget, Checklist, Klaim Undangan)
	couple := api.Group("/couple", middleware.Protected(), middleware.RequireCouple())
	couple.Get("/plan", handlers.GetMyWeddingPlan)
	couple.Put("/plan", handlers.UpdateMyWeddingPlan)
	couple.Post("/plan/join", handlers.JoinWeddingPlanByCode)
	couple.Get("/vendors", handlers.GetWeddingVendors)
	couple.Post("/vendors", handlers.AddManualWeddingVendor)
	couple.Post("/vendors/link-agenda", handlers.LinkDekorinAgenda)
	couple.Post("/vendors/from-directory", handlers.AddPublicVendorToPlan)
	couple.Put("/vendors/:id", handlers.UpdateWeddingVendor)
	couple.Delete("/vendors/:id", handlers.DeleteWeddingVendor)
	couple.Get("/tasks", handlers.GetWeddingTasks)
	couple.Post("/tasks", handlers.CreateWeddingTask)
	couple.Put("/tasks/:id", handlers.UpdateWeddingTask)
	couple.Patch("/tasks/:id/toggle", handlers.ToggleWeddingTask)
	couple.Delete("/tasks/:id", handlers.DeleteWeddingTask)
	couple.Get("/invitations", handlers.GetCoupleInvitations)
	couple.Post("/invitations/claim", handlers.ClaimCoupleInvitation)
	couple.Get("/invitations/:id", handlers.GetCoupleInvitationDetail)
	couple.Put("/invitations/:id", handlers.UpdateCoupleInvitation)
	couple.Put("/invitations/:id/guests", handlers.SaveCoupleInvitationGuests)
	couple.Post("/invitations/:id/sync-plan", handlers.SyncWeddingPlanToInvitation)
	couple.Delete("/invitations/:id/claim", handlers.UnclaimCoupleInvitation)

	// Superadmin / Web Admin routes (dapat diakses superadmin, owner, dan admin)
	superadmin := api.Group("/superadmin")
	superadmin.Post("/login", loginLimiter, handlers.SuperadminLogin)
	superadmin.Get("/me", middleware.Protected(), handlers.GetCurrentUserProfile)
	superadmin.Get("/dashboard-stats", middleware.Protected(), middleware.RequireRole("superadmin", "owner", "admin"), handlers.GetSuperadminDashboardStats)
	superadmin.Get("/companies", middleware.Protected(), middleware.RequireRole("superadmin", "owner", "admin"), handlers.GetAllCompanies)
	superadmin.Post("/companies", middleware.Protected(), middleware.RequireSuperadmin(), handlers.CreateCompanyWithInitialOwner)
	superadmin.Put("/companies/:id", middleware.Protected(), middleware.RequireRole("superadmin", "owner", "admin"), handlers.UpdateCompany)
	superadmin.Delete("/companies/:id", middleware.Protected(), middleware.RequireSuperadmin(), handlers.DeleteCompany)

	// Company profile routes (Data profil badan usaha masing-masing vendor)
	api.Get("/company/profile", middleware.Protected(), middleware.RequireVendor(), handlers.GetMyCompanyProfile)
	api.Put("/company/profile", middleware.Protected(), middleware.RequireRole("owner", "admin"), handlers.UpdateMyCompanyProfile)

	// Company Invoice Template routes (Pengaturan Desain Template Invoice per Brand)
	api.Get("/company/invoice-template", middleware.Protected(), middleware.RequireVendor(), handlers.GetCompanyInvoiceTemplate)
	api.Put("/company/invoice-template", middleware.Protected(), middleware.RequireRole("owner", "admin"), handlers.UpdateCompanyInvoiceTemplate)

	// Company Brand (Sub-Brand / Multi-Brand) routes
	api.Get("/company/brands", middleware.Protected(), middleware.RequireVendor(), handlers.GetCompanyBrands)
	api.Post("/company/brands", middleware.Protected(), middleware.RequireRole("owner", "admin"), handlers.CreateCompanyBrand)
	api.Put("/company/brands/:id", middleware.Protected(), middleware.RequireRole("owner", "admin"), handlers.UpdateCompanyBrand)
	api.Delete("/company/brands/:id", middleware.Protected(), middleware.RequireRole("owner", "admin"), handlers.DeleteCompanyBrand)

	// Company Role Permissions routes (Pengaturan Hak Akses Role oleh Owner)
	api.Get("/company/role-permissions", middleware.Protected(), middleware.RequireVendor(), handlers.GetCompanyRolePermissions)
	api.Put("/company/role-permissions", middleware.Protected(), middleware.RequireRole("owner"), handlers.UpdateCompanyRolePermissions)
	api.Post("/company/role-permissions/reset", middleware.Protected(), middleware.RequireRole("owner"), handlers.ResetCompanyRolePermissions)

	// Company Client Form Config routes (Pengaturan Daftar Isian Link Client per Sub-Brand & Jenis Acara)
	api.Get("/company/form-config/templates", middleware.Protected(), middleware.RequireVendor(), handlers.GetCompanyFormTemplates)
	api.Get("/company/form-config", middleware.Protected(), middleware.RequireVendor(), handlers.GetCompanyFormConfig)
	api.Put("/company/form-config", middleware.Protected(), middleware.RequireRole("owner", "admin"), handlers.UpdateCompanyFormConfig)
	api.Post("/company/form-config/reset", middleware.Protected(), middleware.RequireRole("owner", "admin"), handlers.ResetCompanyFormConfig)
	api.Delete("/company/form-config", middleware.Protected(), middleware.RequireRole("owner", "admin"), handlers.DeleteCompanyFormConfigTemplate)

	// User routes
	api.Get("/users", middleware.Protected(), middleware.RequireVendor(), handlers.GetAllUsers)
	api.Post("/users", middleware.Protected(), middleware.RequireVendor(), handlers.CreateUser)
	api.Put("/users/:id", middleware.Protected(), middleware.RequireVendor(), handlers.UpdateUser)
	api.Delete("/users/:id", middleware.Protected(), middleware.RequireVendor(), handlers.DeleteUser)

	// Dashboard summary route
	api.Get("/dashboard/summary", middleware.Protected(), middleware.RequireVendor(), handlers.GetDashboardSummary)

	// Agenda routes
	api.Get("/agendas", middleware.Protected(), middleware.RequireVendor(), handlers.GetAgendas)
	api.Get("/agendas/pending-links", middleware.Protected(), middleware.RequireVendor(), handlers.GetPendingClientLinks)
	api.Post("/agendas/generate-link", middleware.Protected(), middleware.RequireVendor(), handlers.GenerateClientFormLink)
	api.Post("/agendas", middleware.Protected(), middleware.RequireVendor(), handlers.CreateAgenda)
	api.Put("/agendas/:id", middleware.Protected(), middleware.RequireVendor(), handlers.UpdateAgenda)
	api.Post("/agendas/:id/unlock-form", middleware.Protected(), middleware.RequireVendor(), handlers.UnlockAgendaForm)
	api.Put("/agendas/:id/schedule", middleware.Protected(), middleware.RequireVendor(), handlers.UpdateAgendaSchedule)
	api.Put("/agendas/:id/location", middleware.Protected(), middleware.RequireVendor(), handlers.UpdateAgendaLocation)
	api.Put("/agendas/:id/dp", middleware.Protected(), middleware.RequireVendor(), handlers.UpdateAgendaDp)
	api.Put("/agendas/:id/transport", middleware.Protected(), middleware.RequireVendor(), handlers.UpdateAgendaTransport)
	api.Put("/agendas/:id/package", middleware.Protected(), middleware.RequireVendor(), handlers.UpdateAgendaPackage)
	api.Put("/agendas/:id/model", middleware.Protected(), middleware.RequireVendor(), handlers.UpdateAgendaModel)
	api.Put("/agendas/:id/notes", middleware.Protected(), middleware.RequireVendor(), handlers.UpdateAgendaNotes)
	api.Put("/agendas/:id/payment-status", middleware.Protected(), middleware.RequireVendor(), handlers.UpdateAgendaPaymentStatus)
	api.Put("/agendas/:id/status", middleware.Protected(), middleware.RequireVendor(), handlers.UpdateAgendaStatus)
	api.Put("/agendas/:id/brand", middleware.Protected(), middleware.RequireVendor(), handlers.UpdateAgendaBrand)
	api.Delete("/agendas/:id", middleware.Protected(), middleware.RequireVendor(), handlers.DeleteAgenda)

	// Expense routes
	api.Get("/expenses", middleware.Protected(), middleware.RequireVendor(), handlers.GetExpenses)
	api.Post("/expenses", middleware.Protected(), middleware.RequireVendor(), handlers.CreateExpense)
	api.Delete("/expenses/:id", middleware.Protected(), middleware.RequireVendor(), handlers.DeleteExpense)

	// AI routes (Smart OCR, Gemini Assistant & Decor Color Simulation)
	api.Post("/ai/scan-receipt", middleware.Protected(), middleware.RequireVendor(), handlers.ScanReceiptAI)
	api.Post("/ai/assistant", middleware.Protected(), middleware.RequireVendor(), handlers.HandleAIAssistant)
	api.Post("/ai/simulate-decor", middleware.Protected(), middleware.RequireVendor(), handlers.SimulateDecorAI)

	// Master Package routes
	api.Get("/packages", middleware.Protected(), middleware.RequireVendor(), handlers.GetPackages)
	api.Post("/packages", middleware.Protected(), middleware.RequireVendor(), handlers.CreatePackage)
	api.Put("/packages/reorder", middleware.Protected(), middleware.RequireVendor(), handlers.ReorderPackages)
	api.Put("/packages/:id", middleware.Protected(), middleware.RequireVendor(), handlers.UpdatePackage)
	api.Delete("/packages/:id", middleware.Protected(), middleware.RequireVendor(), handlers.DeletePackage)

	// Master Addon routes
	api.Get("/addons", middleware.Protected(), middleware.RequireVendor(), handlers.GetAddons)
	api.Post("/addons", middleware.Protected(), middleware.RequireVendor(), handlers.CreateAddon)
	api.Put("/addons/reorder", middleware.Protected(), middleware.RequireVendor(), handlers.ReorderAddons)
	api.Put("/addons/:id", middleware.Protected(), middleware.RequireVendor(), handlers.UpdateAddon)
	api.Delete("/addons/:id", middleware.Protected(), middleware.RequireVendor(), handlers.DeleteAddon)

	// Master Decoration Model routes
	api.Get("/decoration-models", middleware.Protected(), middleware.RequireVendor(), handlers.GetDecorationModels)
	api.Post("/decoration-models", middleware.Protected(), middleware.RequireVendor(), handlers.CreateDecorationModel)
	api.Put("/decoration-models/reorder", middleware.Protected(), middleware.RequireVendor(), handlers.ReorderDecorationModels)
	api.Put("/decoration-models/:id", middleware.Protected(), middleware.RequireVendor(), handlers.UpdateDecorationModel)
	api.Delete("/decoration-models/:id", middleware.Protected(), middleware.RequireVendor(), handlers.DeleteDecorationModel)

	// Master Properties & Tools routes (Inventaris Properti Dekorasi & Alat Kerja)
	api.Get("/properties", middleware.Protected(), middleware.RequireVendor(), handlers.GetProperties)
	api.Get("/properties/availability", middleware.Protected(), middleware.RequireVendor(), handlers.GetPropertiesAvailability)
	api.Post("/properties", middleware.Protected(), middleware.RequireVendor(), handlers.CreateProperty)
	api.Put("/properties/:id", middleware.Protected(), middleware.RequireVendor(), handlers.UpdateProperty)
	api.Delete("/properties/:id", middleware.Protected(), middleware.RequireVendor(), handlers.DeleteProperty)

	// Agenda Pick List routes (Paperless Checklist Muat & Bongkar)
	api.Get("/agendas/:id/properties", middleware.Protected(), middleware.RequireVendor(), handlers.GetAgendaProperties)
	api.Post("/agendas/:id/properties/populate", middleware.Protected(), middleware.RequireVendor(), handlers.PopulateAgendaProperties)
	api.Post("/agendas/:id/properties/batch", middleware.Protected(), middleware.RequireVendor(), handlers.BatchAddAgendaProperties)
	api.Delete("/agendas/:id/properties/clear", middleware.Protected(), middleware.RequireVendor(), handlers.ClearAgendaProperties)
	api.Post("/agendas/:id/properties", middleware.Protected(), middleware.RequireVendor(), handlers.AddAgendaProperty)
	api.Put("/agendas/:id/properties/:item_id", middleware.Protected(), middleware.RequireVendor(), handlers.UpdateAgendaPropertyItem)
	api.Patch("/agendas/:id/properties/:item_id/pack", middleware.Protected(), middleware.RequireVendor(), handlers.ToggleAgendaPropertyPacked)
	api.Patch("/agendas/:id/properties/:item_id/return", middleware.Protected(), middleware.RequireVendor(), handlers.ToggleAgendaPropertyReturned)
	api.Delete("/agendas/:id/properties/:item_id", middleware.Protected(), middleware.RequireVendor(), handlers.DeleteAgendaPropertyItem)

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

	// Web Superadmin Dashboard & PWA routes
	registerSuperadminWebRoutes(app, "./web")

	// Client Web Form routes (dengan rate limit untuk mencegah brute force token)
	app.Get("/form/:token", formLimiter, handlers.ShowClientForm)                        // Public HTML form / summary view
	api.Post("/client-form/:token", formLimiter, handlers.SubmitClientForm)              // Submit form API
	api.Post("/client-form/:token/upload", formLimiter, handlers.UploadClientFormMedia)  // Public image upload for client form
	api.Get("/agendas/:id/client-form", middleware.Protected(), middleware.RequireVendor(), handlers.GetClientFormByAgendaID) // Fetch submitted form for Flutter app

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
	api.Get("/invitations", middleware.Protected(), middleware.RequireVendor(), handlers.GetInvitations)
	api.Post("/invitations", middleware.Protected(), middleware.RequireVendor(), handlers.CreateInvitation)
	api.Get("/invitations/:id", middleware.Protected(), middleware.RequireVendor(), handlers.GetInvitationByID)
	api.Put("/invitations/:id", middleware.Protected(), middleware.RequireVendor(), handlers.UpdateInvitation)
	api.Post("/invitations/:id/reset-access", middleware.Protected(), middleware.RequireVendor(), handlers.ResetInvitationAccess)
	api.Delete("/invitations/:id", middleware.Protected(), middleware.RequireVendor(), handlers.DeleteInvitation)
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

func registerSuperadminWebRoutes(app *fiber.App, webDir string) {
	app.Get("/superadmin", func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		return c.SendFile(filepath.Join(webDir, "templates", "superadmin.html"))
	})

	app.Get("/superadmin/manifest.json", func(c *fiber.Ctx) error {
		if err := c.SendFile(filepath.Join(webDir, "assets", "superadmin-manifest.json")); err != nil {
			return err
		}
		c.Set("Content-Type", "application/manifest+json; charset=utf-8")
		c.Set("Cache-Control", "no-cache, must-revalidate")
		return nil
	})

	app.Get("/superadmin/sw.js", func(c *fiber.Ctx) error {
		if err := c.SendFile(filepath.Join(webDir, "assets", "superadmin-sw.js")); err != nil {
			return err
		}
		c.Set("Content-Type", "application/javascript; charset=utf-8")
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Set("Pragma", "no-cache")
		c.Set("Service-Worker-Allowed", "/superadmin")
		return nil
	})
}

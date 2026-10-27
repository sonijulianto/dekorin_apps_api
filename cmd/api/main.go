package main

import (
	"log"
	"os"
	"time"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/handlers"
	"dekorin_apps_api/internal/middleware"
	"dekorin_apps_api/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
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

	// Group routes
	api := app.Group("/api")
	auth := api.Group("/auth")

	// Routes
	auth.Post("/login", handlers.Login)
	auth.Post("/change-password", middleware.Protected(), handlers.ChangePassword)

	// Superadmin / Web Admin routes (dapat diakses superadmin, owner, dan admin)
	superadmin := api.Group("/superadmin")
	superadmin.Post("/login", handlers.SuperadminLogin)
	superadmin.Get("/me", middleware.Protected(), handlers.GetCurrentUserProfile)
	superadmin.Get("/dashboard-stats", middleware.Protected(), middleware.RequireRole("superadmin", "owner", "admin"), handlers.GetSuperadminDashboardStats)
	superadmin.Get("/companies", middleware.Protected(), middleware.RequireRole("superadmin", "owner", "admin"), handlers.GetAllCompanies)
	superadmin.Post("/companies", middleware.Protected(), middleware.RequireSuperadmin(), handlers.CreateCompanyWithInitialOwner)
	superadmin.Put("/companies/:id", middleware.Protected(), middleware.RequireRole("superadmin", "owner", "admin"), handlers.UpdateCompany)
	superadmin.Delete("/companies/:id", middleware.Protected(), middleware.RequireSuperadmin(), handlers.DeleteCompany)

	// Company profile routes (Data profil badan usaha masing-masing vendor dekorasi)
	api.Get("/company/profile", middleware.Protected(), handlers.GetMyCompanyProfile)
	api.Put("/company/profile", middleware.Protected(), middleware.RequireRole("owner", "admin"), handlers.UpdateMyCompanyProfile)

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
	api.Put("/agendas/:id/dp", middleware.Protected(), handlers.UpdateAgendaDp)
	api.Put("/agendas/:id/transport", middleware.Protected(), handlers.UpdateAgendaTransport)
	api.Put("/agendas/:id/package", middleware.Protected(), handlers.UpdateAgendaPackage)
	api.Put("/agendas/:id/model", middleware.Protected(), handlers.UpdateAgendaModel)
	api.Put("/agendas/:id/notes", middleware.Protected(), handlers.UpdateAgendaNotes)
	api.Put("/agendas/:id/payment-status", middleware.Protected(), handlers.UpdateAgendaPaymentStatus)
	api.Put("/agendas/:id/status", middleware.Protected(), handlers.UpdateAgendaStatus)
	api.Delete("/agendas/:id", middleware.Protected(), handlers.DeleteAgenda)

	// Expense routes
	api.Get("/expenses", middleware.Protected(), handlers.GetExpenses)
	api.Post("/expenses", middleware.Protected(), handlers.CreateExpense)
	api.Delete("/expenses/:id", middleware.Protected(), handlers.DeleteExpense)

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

	// API Logs routes (sesuai Screenshot)
	api.Get("/apilog/get", middleware.OptionalProtected(), handlers.GetApiLogs)
	api.Get("/apilog/:id", middleware.OptionalProtected(), handlers.GetApiLogByID)
	api.Get("/apilogs", middleware.OptionalProtected(), handlers.GetApiLogs)

	// App Version & In-App Update routes (Hanya Superadmin yang berhak upload & atur versi)
	api.Get("/app/version", handlers.GetLatestAppVersion)
	api.Get("/apk/check-version", handlers.GetLatestAppVersion)
	api.Get("/app/download-latest", handlers.DownloadLatestApk)
	app.Get("/download/apk", handlers.DownloadLatestApk)
	api.Get("/app/version-history", middleware.Protected(), middleware.RequireSuperadmin(), handlers.GetAppVersionHistory)
	api.Post("/app/version", middleware.Protected(), middleware.RequireSuperadmin(), handlers.UpdateAppVersion)
	api.Post("/app/upload-apk", middleware.Protected(), middleware.RequireSuperadmin(), handlers.UploadApkFile)
	api.Delete("/app/version/:id", middleware.Protected(), middleware.RequireSuperadmin(), handlers.DeleteAppVersion)

	// Serve static upload directory
	app.Static("/uploads", "./uploads")
	app.Static("/assets", "./web/assets")
	app.Static("/images", "./web/assets/images")

	// Upload route
	api.Post("/upload", handlers.UploadFile)

	// Landing Page route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("./web/templates/landing.html")
	})

	// Web Superadmin Dashboard route
	app.Get("/superadmin", func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		return c.SendFile("./web/templates/superadmin.html")
	})

	// Client Web Form routes
	app.Get("/form/:token", handlers.ShowClientForm)             // Public HTML form view
	api.Post("/client-form/:token", handlers.SubmitClientForm)   // Submit form API
	api.Get("/agendas/:id/client-form", handlers.GetClientFormByAgendaID) // Fetch submitted form for Flutter app

	// Digital Invitation routes
	app.Get("/invitation/:slug", handlers.ShowPublicInvitation) // Public HTML wedding invitation
	api.Get("/invitations", middleware.Protected(), handlers.GetInvitations)
	api.Post("/invitations", middleware.Protected(), handlers.CreateInvitation)
	api.Get("/invitations/:id", middleware.Protected(), handlers.GetInvitationByID)
	api.Put("/invitations/:id", middleware.Protected(), handlers.UpdateInvitation)
	api.Delete("/invitations/:id", middleware.Protected(), handlers.DeleteInvitation)
	api.Get("/invitations/:id/wishes", handlers.GetInvitationWishes)
	api.Post("/invitations/:slug/wishes", handlers.SubmitInvitationWish)

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

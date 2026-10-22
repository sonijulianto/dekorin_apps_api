package main

import (
	"log"
	"os"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/handlers"
	"dekorin_apps_api/internal/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	// Koneksi ke Database PostgreSQL
	config.ConnectDB()

	// Initialize Fiber app dengan limit body 30MB untuk upload file
	app := fiber.New(fiber.Config{
		BodyLimit: 30 * 1024 * 1024, // 30 MB
	})

	// Middleware
	app.Use(logger.New()) // Log every request
	app.Use(cors.New())   // Enable CORS for all origins

	// Group routes
	api := app.Group("/api")
	auth := api.Group("/auth")

	// Routes
	auth.Post("/login", handlers.Login)

	// Superadmin routes
	superadmin := api.Group("/superadmin")
	superadmin.Post("/login", handlers.SuperadminLogin)
	superadmin.Get("/companies", middleware.Protected(), middleware.RequireSuperadmin(), handlers.GetAllCompanies)
	superadmin.Post("/companies", middleware.Protected(), middleware.RequireSuperadmin(), handlers.CreateCompanyWithInitialOwner)
	superadmin.Put("/companies/:id", middleware.Protected(), middleware.RequireSuperadmin(), handlers.UpdateCompany)
	superadmin.Delete("/companies/:id", middleware.Protected(), middleware.RequireSuperadmin(), handlers.DeleteCompany)

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
	api.Put("/agendas/:id/payment-status", middleware.Protected(), handlers.UpdateAgendaPaymentStatus)
	api.Put("/agendas/:id/status", middleware.Protected(), handlers.UpdateAgendaStatus)

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

	// Serve static upload directory
	app.Static("/uploads", "./uploads")
	app.Static("/assets", "./web/assets")

	// Upload route
	api.Post("/upload", handlers.UploadFile)

	// Landing Page route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("./web/templates/landing.html")
	})

	// Web Superadmin Dashboard route
	app.Get("/superadmin", func(c *fiber.Ctx) error {
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

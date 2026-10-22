package main

import (
	"log"
	"os"

	"dekorin_apps_api/internal/config"
	"dekorin_apps_api/internal/handlers"

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

	// User routes
	api.Get("/users", handlers.GetAllUsers)

	// Dashboard summary route
	api.Get("/dashboard/summary", handlers.GetDashboardSummary)

	// Agenda routes
	api.Get("/agendas", handlers.GetAgendas)
	api.Post("/agendas", handlers.CreateAgenda)
	api.Put("/agendas/:id/dp", handlers.UpdateAgendaDp)
	api.Put("/agendas/:id/transport", handlers.UpdateAgendaTransport)
	api.Put("/agendas/:id/package", handlers.UpdateAgendaPackage)
	api.Put("/agendas/:id/payment-status", handlers.UpdateAgendaPaymentStatus)
	api.Put("/agendas/:id/status", handlers.UpdateAgendaStatus)

	// Expense routes
	api.Get("/expenses", handlers.GetExpenses)
	api.Post("/expenses", handlers.CreateExpense)
	api.Delete("/expenses/:id", handlers.DeleteExpense)

	// Master Package routes
	api.Get("/packages", handlers.GetPackages)
	api.Post("/packages", handlers.CreatePackage)
	api.Put("/packages/reorder", handlers.ReorderPackages)
	api.Put("/packages/:id", handlers.UpdatePackage)
	api.Delete("/packages/:id", handlers.DeletePackage)

	// Master Addon routes
	api.Get("/addons", handlers.GetAddons)
	api.Post("/addons", handlers.CreateAddon)
	api.Put("/addons/reorder", handlers.ReorderAddons)
	api.Put("/addons/:id", handlers.UpdateAddon)
	api.Delete("/addons/:id", handlers.DeleteAddon)

	// Serve static upload directory
	app.Static("/uploads", "./uploads")
	app.Static("/assets", "./web/assets")

	// Upload route
	api.Post("/upload", handlers.UploadFile)

	// Landing Page route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("./web/templates/landing.html")
	})

	// Client Web Form routes
	app.Get("/form/:token", handlers.ShowClientForm)             // Public HTML form view
	api.Post("/client-form/:token", handlers.SubmitClientForm)   // Submit form API
	api.Get("/agendas/:id/client-form", handlers.GetClientFormByAgendaID) // Fetch submitted form for Flutter app

	// Digital Invitation routes
	app.Get("/invitation/:slug", handlers.ShowPublicInvitation) // Public HTML wedding invitation
	api.Get("/invitations", handlers.GetInvitations)
	api.Post("/invitations", handlers.CreateInvitation)
	api.Get("/invitations/:id", handlers.GetInvitationByID)
	api.Put("/invitations/:id", handlers.UpdateInvitation)
	api.Delete("/invitations/:id", handlers.DeleteInvitation)
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

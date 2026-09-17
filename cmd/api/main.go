package main

import (
	"log"

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

	// Agenda & Package routes
	api.Get("/agendas", handlers.GetAgendas)
	api.Post("/agendas", handlers.CreateAgenda)

	// Master Package routes
	api.Get("/packages", handlers.GetPackages)
	api.Post("/packages", handlers.CreatePackage)
	api.Put("/packages/:id", handlers.UpdatePackage)
	api.Delete("/packages/:id", handlers.DeletePackage)

	// Master Addon routes
	api.Get("/addons", handlers.GetAddons)
	api.Post("/addons", handlers.CreateAddon)
	api.Put("/addons/:id", handlers.UpdateAddon)
	api.Delete("/addons/:id", handlers.DeleteAddon)

	// Serve static upload directory
	app.Static("/uploads", "./uploads")

	// Upload route
	api.Post("/upload", handlers.UploadFile)

	// Client Web Form routes
	app.Get("/form/:token", handlers.ShowClientForm)             // Public HTML form view
	api.Post("/client-form/:token", handlers.SubmitClientForm)   // Submit form API
	api.Get("/agendas/:id/client-form", handlers.GetClientFormByAgendaID) // Fetch submitted form for Flutter app

	// Start server on port 3000
	log.Println("Server started at http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
}

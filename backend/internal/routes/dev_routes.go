package routes

import (
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"faha.local/backend/internal/handlers"
)

func SetupDevRoutes(api fiber.Router, db *gorm.DB) {
	devHandler := handlers.NewDevZoneHandler(db)
	
	devGroup := api.Group("/dev")
	
	devGroup.Post("/zones", devHandler.CreateZone)
	devGroup.Get("/zones", devHandler.GetZones)
	
	// این دو خط جدید هستند:
	devGroup.Put("/zones/:id", devHandler.UpdateZone)
	devGroup.Delete("/zones/:id", devHandler.DeleteZone)
}
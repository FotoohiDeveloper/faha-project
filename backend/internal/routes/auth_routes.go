package routes

import (
	"github.com/gofiber/fiber/v3"
	"faha.local/backend/internal/handlers"
)

func SetupAuthRoutes(api fiber.Router, authHandler *handlers.AuthHandler) {
	authGroup := api.Group("/auth")

	authGroup.Post("/setup/begin", authHandler.SetupBegin)
	authGroup.Post("/setup/finish", authHandler.SetupFinish)

	authGroup.Post("/login/begin", authHandler.LoginBegin)
	authGroup.Post("/login/finish", authHandler.LoginFinish)
}
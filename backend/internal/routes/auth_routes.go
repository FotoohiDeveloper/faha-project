package routes

import (
	"github.com/gofiber/fiber/v3"
	"faha.local/backend/internal/handlers"
)

func SetupAuthRoutes(api fiber.Router, authHandler *handlers.AuthHandler) {
	authGroup := api.Group("/auth")

	// این دو مسیر برای فلوی Two-Man Rule و ستاپ اولیه است که با هم نوشتیم
	authGroup.Post("/setup/begin", authHandler.SetupBegin)
	authGroup.Post("/setup/finish", authHandler.SetupFinish)

	// (مسیرهای ایجاد کاربر توسط فرمانده و لاگین اصلی هم بعدا اینجا اضافه میشن)
}
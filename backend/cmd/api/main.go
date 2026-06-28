package main

import (
	"log"
	"os"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"faha.local/backend/internal/database"
	"faha.local/backend/internal/handlers"
	"faha.local/backend/internal/routes"
	"faha.local/backend/internal/service"
)

func main() {
	// ۱. اتصال به دیتابیس و ردیس
	database.Connect()

	// ۲. تنظیمات WebAuthn (FIDO2)
	wConfig := &webauthn.Config{
		RPDisplayName: os.Getenv("RP_DISPLAY_NAME"),
		RPID:          os.Getenv("RP_ID"),
		RPOrigins:     []string{os.Getenv("RP_ORIGIN")},

		AuthenticatorSelection: protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.AuthenticatorAttachment(""),
			UserVerification:        protocol.VerificationRequired,
		},
	}

	webAuthnApp, err := webauthn.New(wConfig)
	if err != nil {
		log.Fatal("❌ Failed to initialize WebAuthn: ", err)
	}
	log.Println("✅ WebAuthn Initialized")

	// ۳. تزریق وابستگی‌ها (Dependency Injection)
	authService := service.NewAuthService(database.DB, database.Redis, webAuthnApp)

	// ساخت هندلرها
	authHandler := handlers.NewAuthHandler(authService, database.Redis)
	// commanderHandler := handlers.NewCommanderHandler(authService) // هندلر فرماندهی اضافه شد

	// ۴. راه‌اندازی Fiber
	app := fiber.New(fiber.Config{
		AppName:       "FAHA C4ISR Backend v1.0",
		CaseSensitive: true,
		StrictRouting: true,
	})

	// ۵. میان‌افزارها
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	// سرو کردن پنل نقشه بدون مشکل CORS
	app.Get("/dev/map", func(c fiber.Ctx) error {
		return c.SendFile("./map.html")
	})

	// ۶. ثبت روت‌ها در گروه api/v1
	api := app.Group("/api/v1")

	// روت‌های احراز هویت
	routes.SetupAuthRoutes(api, authHandler)

	// روت‌های فرماندهی (محافظت شده)
	// routes.SetupCommanderRoutes(api, database.DB, database.Redis, commanderHandler)

	// روت‌های توسعه‌دهنده (نقشه)
	routes.SetupDevRoutes(api, database.DB)

	// ۷. اجرای سرور
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("🚀 Server is running on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatal("❌ Server failed to start: ", err)
	}
}

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
	database.Connect()

	// اصلاح استفاده از پکیج protocol
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

	authService := service.NewAuthService(database.DB, database.Redis, webAuthnApp)
	authHandler := handlers.NewAuthHandler(authService)

	app := fiber.New(fiber.Config{
		AppName:       "FAHA C4ISR Backend v1.0",
		CaseSensitive: true,
		StrictRouting: true,
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, 
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	api := app.Group("/api/v1")
	routes.SetupAuthRoutes(api, authHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("🚀 Server is running on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatal("❌ Server failed to start: ", err)
	}
}
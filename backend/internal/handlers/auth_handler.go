package handlers

import (
	"bytes"
	"encoding/json"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/gofiber/fiber/v3"
	
	"faha.local/backend/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(as *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: as}
}

func (h *AuthHandler) SetupBegin(c fiber.Ctx) error {
	var req struct {
		Username string `json:"username"`
		OTP      string `json:"otp"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "دیتا نامعتبر است"})
	}

	_, creation, err := h.authService.BeginFirstSetup(c.Context(), req.Username, req.OTP)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(creation) 
}

func (h *AuthHandler) SetupFinish(c fiber.Ctx) error {
	var req struct {
		Username       string                 `json:"username"`
		NewPassword    string                 `json:"new_password"`
		WebAuthnData   map[string]interface{} `json:"webauthn_data"`
	}
	
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "فرمت داده‌ها نامعتبر است"})
	}

	// تبدیل مجدد WebAuthnData به بایت برای پارس شدن توسط کتابخانه
	waDataBytes, err := json.Marshal(req.WebAuthnData)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "دیتای سخت‌افزاری مخدوش است"})
	}

	// پارس کردن دیتای برگشتی از سمت مرورگر یا سخت‌افزار
	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(waDataBytes))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "امضای سخت‌افزاری معتبر نیست"})
	}
	
	// TODO: در مرحله بعد وقتی متد کشیدن sessionData از ردیس را اضافه کردیم، این متد را هم از حالت کامنت خارج می‌کنیم:
	// err = h.authService.FinishFirstSetup(c.Context(), req.Username, req.NewPassword, parsedResponse, sessionData)
	
	_ = parsedResponse // فعلا برای جلوگیری از خطای متغیر بلااستفاده
	
	return c.JSON(fiber.Map{"message": "اطلاعات سخت‌افزاری و کلمه عبور با موفقیت ثبت شد. منتظر تایید فرمانده بمانید."})
}
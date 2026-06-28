package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"

	"faha.local/backend/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
	redis       *redis.Client
}

func NewAuthHandler(as *service.AuthService, rdb *redis.Client) *AuthHandler {
	return &AuthHandler{authService: as, redis: rdb}
}

// SetupBegin — اپراتور با username + OTP وارد می‌شه و چالش WebAuthn دریافت می‌کنه
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

// SetupFinish — اپراتور امضای Passkey و رمز جدید رو می‌فرسته
func (h *AuthHandler) SetupFinish(c fiber.Ctx) error {
	var req struct {
		Username     string                 `json:"username"`
		NewPassword  string                 `json:"new_password"`
		WebAuthnData map[string]interface{} `json:"webauthn_data"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "فرمت داده‌ها نامعتبر است"})
	}

	// ۱. خواندن sessionData از Redis
	sessionKey := fmt.Sprintf("webauthn_reg_session:%s", req.Username)
	sessionBytes, err := h.redis.Get(c.Context(), sessionKey).Bytes()
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "نشست ثبت کلید منقضی شده یا یافت نشد. دوباره از ابتدا شروع کنید."})
	}

	var sessionData webauthn.SessionData
	if err := json.Unmarshal(sessionBytes, &sessionData); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "خطا در بازیابی داده‌های نشست"})
	}

	// ۲. پارس کردن پاسخ WebAuthn ارسال‌شده از کلاینت
	waDataBytes, err := json.Marshal(req.WebAuthnData)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "دیتای سخت‌افزاری مخدوش است"})
	}

	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(waDataBytes))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "امضای سخت‌افزاری معتبر نیست: " + err.Error()})
	}

	// ۳. تکمیل فرآیند ثبت (ذخیره Passkey + رمز عبور + تغییر وضعیت به PENDING_APPROVAL)
	if err := h.authService.FinishFirstSetup(c.Context(), req.Username, req.NewPassword, parsedResponse, sessionData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// ۴. حذف sessionData از Redis پس از موفقیت (یک‌بار مصرف)
	h.redis.Del(c.Context(), sessionKey)

	return c.JSON(fiber.Map{"message": "اطلاعات سخت‌افزاری و کلمه عبور با موفقیت ثبت شد. منتظر تایید فرمانده بمانید."})
}

// LoginBegin — کاربر username می‌ده و چالش WebAuthn دریافت می‌کنه
func (h *AuthHandler) LoginBegin(c fiber.Ctx) error {
	var req struct {
		Username string `json:"username"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "دیتا نامعتبر است"})
	}

	_, assertion, err := h.authService.BeginLogin(c.Context(), req.Username)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(assertion)
}

// LoginFinish — کاربر امضای Passkey رو می‌فرسته و توکن سشن دریافت می‌کنه
func (h *AuthHandler) LoginFinish(c fiber.Ctx) error {
	var req struct {
		Username     string                 `json:"username"`
		WebAuthnData map[string]interface{} `json:"webauthn_data"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "فرمت داده‌ها نامعتبر است"})
	}

	// ۱. خواندن sessionData از Redis
	sessionKey := fmt.Sprintf("webauthn_login_session:%s", req.Username)
	sessionBytes, err := h.redis.Get(c.Context(), sessionKey).Bytes()
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "نشست ورود منقضی شده یا یافت نشد. دوباره از ابتدا ورود کنید."})
	}

	var sessionData webauthn.SessionData
	if err := json.Unmarshal(sessionBytes, &sessionData); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "خطا در بازیابی داده‌های نشست"})
	}

	// ۲. پارس کردن پاسخ WebAuthn ارسال‌شده از کلاینت
	waDataBytes, err := json.Marshal(req.WebAuthnData)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "دیتای سخت‌افزاری مخدوش است"})
	}

	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(waDataBytes))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "امضای سخت‌افزاری معتبر نیست: " + err.Error()})
	}

	// ۳. تکمیل فرآیند ورود (تایید امضا + تولید توکن)
	sessionToken, err := h.authService.FinishLogin(c.Context(), req.Username, parsedResponse, sessionData)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	// ۴. حذف sessionData از Redis پس از موفقیت (یک‌بار مصرف)
	h.redis.Del(c.Context(), sessionKey)

	// ۵. تنظیم کوکی امن برای مرورگر
	c.Cookie(&fiber.Cookie{
		Name:     "faha_session",
		Value:    sessionToken,
		Expires:  time.Now().Add(12 * time.Hour),
		HTTPOnly: true,
		Secure:   false, // در محیط production به true تغییر دهید (HTTPS)
		SameSite: "Lax",
	})

	return c.JSON(fiber.Map{"message": "ورود با موفقیت انجام شد", "token": sessionToken})
}

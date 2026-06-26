package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"faha.local/backend/internal/models"
	"faha.local/backend/internal/service"
)

type CommanderHandler struct {
	authService *service.AuthService
}

func NewCommanderHandler(as *service.AuthService) *CommanderHandler {
	return &CommanderHandler{authService: as}
}

// ایجاد اپراتور جدید توسط فرمانده و دریافت رمز موقت
func (h *CommanderHandler) CreateOperator(c fiber.Ctx) error {
	var req struct {
		Username string          `json:"username"`
		Rank     models.UserRank `json:"rank"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "دیتا نامعتبر است"})
	}

	commanderID := c.Locals("userID").(uuid.UUID)

	otpCode, err := h.authService.CreateOperatorByCommander(c.Context(), commanderID, req.Username, req.Rank)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// کد OTP به فرمانده نمایش داده می‌شود تا به صورت فیزیکی به اپراتور تحویل دهد
	return c.JSON(fiber.Map{
		"message":  "اپراتور با موفقیت ایجاد شد و در وضعیت تعلیق است.",
		"username": req.Username,
		"otp_code": otpCode,
	})
}

// تایید نهایی اپراتوری که Passkey و رمز خود را ست کرده است
func (h *CommanderHandler) ApproveOperator(c fiber.Ctx) error {
	targetUserIDStr := c.Params("id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "شناسه کاربر نامعتبر است"})
	}

	commanderID := c.Locals("userID").(uuid.UUID)

	if err := h.authService.ApproveOperator(c.Context(), commanderID, targetUserID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "اپراتور تایید و حساب کاربری او عملیاتی (ACTIVE) شد.",
	})
}
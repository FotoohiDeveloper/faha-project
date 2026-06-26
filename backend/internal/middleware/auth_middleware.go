package middleware

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"faha.local/backend/internal/models"
)

// Protected یک میان‌افزار برای حفاظت از روت‌های سامانه است
func Protected(db *gorm.DB, rdb *redis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		// ۱. واکشی توکن از کوکی (یا هدر Authorization برای کلاینت‌های موبایل)
		token := c.Cookies("faha_session")
		if token == "" {
			authHeader := c.Get("Authorization")
			if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token = authHeader[7:]
			}
		}

		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "احراز هویت الزامی است. لطفاً دانگل خود را متصل کنید."})
		}

		// ۲. بررسی وجود سشن در Redis (جلوگیری از دسترسی در صورت Revoke شدن)
		ctx := context.Background()
		userIDStr, err := rdb.Get(ctx, fmt.Sprintf("auth_session:%s", token)).Result()
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "نشست کاربری نامعتبر یا منقضی شده است"})
		}

		// ۳. واکشی اطلاعات کاربر از دیتابیس
		var user models.User
		if err := db.First(&user, "id = ?", userIDStr).Error; err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "کاربر یافت نشد"})
		}

		// ۴. بررسی وضعیت امنیتی کاربر
		if user.Status == models.StatusBlocked {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "دسترسی مسدود شده است. با فرماندهی تماس بگیرید."})
		}
		if user.Status != models.StatusActive {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "حساب کاربری شما هنوز به تایید نهایی نرسیده است."})
		}

		// ۵. تزریق اطلاعات کاربر به کانتکست (برای استفاده در مراحل بعدی)
		c.Locals("user", user)
		c.Locals("userID", user.ID)
		c.Locals("userRank", user.Rank)

		return c.Next()
	}
}

// RequireRank بررسی می‌کند که کاربر حتماً یکی از درجه‌های (Rank) مجاز را داشته باشد
func RequireRank(allowedRanks ...models.UserRank) fiber.Handler {
	return func(c fiber.Ctx) error {
		userRank, ok := c.Locals("userRank").(models.UserRank)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "عدم دسترسی سطح دسترسی"})
		}

		for _, rank := range allowedRanks {
			if userRank == rank {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "درجه سازمانی شما برای انجام این عملیات مجاز نیست"})
	}
}
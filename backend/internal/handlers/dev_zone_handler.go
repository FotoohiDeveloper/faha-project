package handlers

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"faha.local/backend/internal/models"
)

type DevZoneHandler struct {
	db *gorm.DB
}

func NewDevZoneHandler(db *gorm.DB) *DevZoneHandler {
	return &DevZoneHandler{db: db}
}

// ساختار دیتای دریافتی از فرانت‌اند
type CreateZoneReq struct {
	Name        string      `json:"name"`
	Coordinates [][]float64 `json:"coordinates"` // آرایه‌ای از [طول جغرافیایی، عرض جغرافیایی]
}

// ذخیره محدوده جدید در دیتابیس
func (h *DevZoneHandler) CreateZone(c fiber.Ctx) error {
	var req CreateZoneReq
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "فرمت درخواست نامعتبر است"})
	}

	if len(req.Coordinates) < 3 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "پلی‌گان باید حداقل ۳ نقطه داشته باشد"})
	}

	// PostGIS اجبار می‌کند که نقطه اول و آخر پلی‌گان دقیقاً یکی باشند تا حلقه بسته شود
	firstPt := req.Coordinates[0]
	lastPt := req.Coordinates[len(req.Coordinates)-1]
	if firstPt[0] != lastPt[0] || firstPt[1] != lastPt[1] {
		req.Coordinates = append(req.Coordinates, firstPt)
	}

	// تبدیل مختصات به فرمت WKT (Well-Known Text) برای ذخیره در PostGIS
	var pointsStr []string
	for _, pt := range req.Coordinates {
		pointsStr = append(pointsStr, fmt.Sprintf("%f %f", pt[0], pt[1])) // دقت کن: اول Longitude بعد Latitude
	}
	wkt := fmt.Sprintf("POLYGON((%s))", strings.Join(pointsStr, ", "))

	// استفاده از gorm.Expr برای اعمال تابع ST_GeomFromText دیتابیس
	err := h.db.Model(&models.Zone{}).Create(map[string]interface{}{
		"name":    req.Name,
		"polygon": gorm.Expr("ST_GeomFromText(?, 4326)", wkt), // 4326 استاندارد GPS است
	}).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "خطا در دیتابیس: " + err.Error()})
	}

	return c.JSON(fiber.Map{"message": "محدوده با موفقیت ذخیره شد", "name": req.Name})
}

// واکشی تمام محدوده‌ها برای نمایش روی نقشه
func (h *DevZoneHandler) GetZones(c fiber.Ctx) error {
	type ZoneRes struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		GeoJSON string `json:"geojson"`
	}
	var zones []ZoneRes
	
	// تبدیل هندسه دیتابیس به فرمت استاندارد GeoJSON برای فرانت‌اند
	err := h.db.Raw("SELECT id, name, ST_AsGeoJSON(polygon) as geo_json FROM zones").Scan(&zones).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(zones)
}

// ویرایش محدوده (تغییر نام یا تغییر نقاط پلی‌گان)
func (h *DevZoneHandler) UpdateZone(c fiber.Ctx) error {
	id := c.Params("id")
	var req CreateZoneReq
	
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "فرمت درخواست نامعتبر است"})
	}

	updates := map[string]interface{}{}

	// اگر نام جدید ارسال شده بود
	if req.Name != "" {
		updates["name"] = req.Name
	}

	// اگر مختصات جدید ارسال شده بود، پلی‌گان جدید را می‌سازیم
	if len(req.Coordinates) >= 3 {
		firstPt := req.Coordinates[0]
		lastPt := req.Coordinates[len(req.Coordinates)-1]
		if firstPt[0] != lastPt[0] || firstPt[1] != lastPt[1] {
			req.Coordinates = append(req.Coordinates, firstPt)
		}

		var pointsStr []string
		for _, pt := range req.Coordinates {
			pointsStr = append(pointsStr, fmt.Sprintf("%f %f", pt[0], pt[1]))
		}
		wkt := fmt.Sprintf("POLYGON((%s))", strings.Join(pointsStr, ", "))
		updates["polygon"] = gorm.Expr("ST_GeomFromText(?, 4326)", wkt)
	}

	// اعمال تغییرات در دیتابیس
	err := h.db.Model(&models.Zone{}).Where("id = ?", id).Updates(updates).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "خطا در بروزرسانی: " + err.Error()})
	}

	return c.JSON(fiber.Map{"message": "محدوده با موفقیت بروزرسانی شد"})
}

// حذف کامل یک محدوده
func (h *DevZoneHandler) DeleteZone(c fiber.Ctx) error {
	id := c.Params("id")
	
	err := h.db.Where("id = ?", id).Delete(&models.Zone{}).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "خطا در حذف: " + err.Error()})
	}

	return c.JSON(fiber.Map{"message": "محدوده با موفقیت حذف شد"})
}
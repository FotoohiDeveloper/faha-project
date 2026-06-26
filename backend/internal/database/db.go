package database

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"faha.local/backend/internal/models"
)

var (
	DB    *gorm.DB
	Redis *redis.Client
)

func Connect() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Tehran",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("❌ Failed to connect to PostgreSQL: ", err)
	}
	log.Println("✅ Connected to PostgreSQL (PostGIS)")

	err = DB.AutoMigrate(
		&models.Zone{},
		&models.User{},
		&models.PassHistory{},
		&models.WebAuthnCred{},
	)
	if err != nil {
		log.Fatal("❌ Failed to run migrations: ", err)
	}
	log.Println("✅ Database Migrations Applied")

	Redis = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	if err := Redis.Ping(context.Background()).Err(); err != nil {
		log.Fatal("❌ Failed to connect to Redis: ", err)
	}
	log.Println("✅ Connected to Redis")
}
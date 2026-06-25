package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"faha.local/backend/internal/models"
	"faha.local/backend/internal/utils"
)

type AuthService struct {
	db       *gorm.DB
	redis    *redis.Client
	webAuthn *webauthn.WebAuthn
}

func NewAuthService(db *gorm.DB, rdb *redis.Client, wa *webauthn.WebAuthn) *AuthService {
	return &AuthService{db: db, redis: rdb, webAuthn: wa}
}

func (s *AuthService) CreateOperatorByCommander(ctx context.Context, commanderID uuid.UUID, username string, rank models.UserRank) (string, error) {
	user := models.User{
		Username:     username,
		Rank:         rank,
		Status:       models.StatusSuspended,
		CreatedByID:  &commanderID,
		PasswordHash: "PENDING_SETUP",
	}

	if err := s.db.Create(&user).Error; err != nil {
		return "", err
	}

	otpCode := utils.GenerateOTP()
	redisKey := fmt.Sprintf("setup_otp:%s", username)
	if err := s.redis.Set(ctx, redisKey, otpCode, 24*time.Hour).Err(); err != nil {
		return "", err
	}

	return otpCode, nil
}

// اصلاح CredentialCreation از پکیج protocol
func (s *AuthService) BeginFirstSetup(ctx context.Context, username, otp string) (*models.User, *protocol.CredentialCreation, error) {
	redisKey := fmt.Sprintf("setup_otp:%s", username)
	val, err := s.redis.Get(ctx, redisKey).Result()
	if err != nil || val != otp {
		return nil, nil, errors.New("کد راه‌اندازی نامعتبر یا منقضی شده است")
	}

	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, nil, err
	}

	if user.Status != models.StatusSuspended {
		return nil, nil, errors.New("وضعیت کاربر برای راه‌اندازی اولیه معتبر نیست")
	}

	creation, sessionData, err := s.webAuthn.BeginRegistration(&user)
	if err != nil {
		return nil, nil, err
	}

	sessionKey := fmt.Sprintf("webauthn_session:%s", username)
	s.redis.Set(ctx, sessionKey, sessionData, 5*time.Minute)

	return &user, creation, nil
}

// اصلاح دیتای ورودی به ParsedCredentialCreationData
func (s *AuthService) FinishFirstSetup(ctx context.Context, username, newPassword string, response *protocol.ParsedCredentialCreationData, sessionData webauthn.SessionData) error {
	if err := utils.ValidatePasswordComplexity(newPassword); err != nil {
		return err
	}

	var user models.User
	if err := s.db.Preload("PasswordHistory").Where("username = ?", username).First(&user).Error; err != nil {
		return err
	}

	for _, history := range user.PasswordHistory {
		if utils.CheckPasswordHash(newPassword, history.PasswordHash) {
			return errors.New("شما قبلاً از این کلمه عبور استفاده کرده‌اید. لطفاً کلمه عبور جدیدی وارد کنید")
		}
	}

	// استفاده از CreateCredential برای دیتای از پیش پارس شده در فریم‌ورک Fiber
	credential, err := s.webAuthn.CreateCredential(&user, sessionData, response)
	if err != nil {
		return errors.New("تایید کلید سخت‌افزاری با خطا مواجه شد: " + err.Error())
	}

	newHash, _ := utils.HashPassword(newPassword)

	return s.db.Transaction(func(tx *gorm.DB) error {
		user.PasswordHash = newHash
		user.Status = models.StatusPendingApproval
		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		tx.Create(&models.PassHistory{UserID: user.ID, PasswordHash: newHash})

		webAuthnCred := models.WebAuthnCred{
			UserID:          user.ID,
			CredentialID:    credential.ID,
			PublicKey:       credential.PublicKey,
			AttestationType: credential.AttestationType,
			SignCount:       credential.Authenticator.SignCount,
		}
		if err := tx.Create(&webAuthnCred).Error; err != nil {
			return err
		}

		s.redis.Del(ctx, fmt.Sprintf("setup_otp:%s", username))

		return nil
	})
}

func (s *AuthService) ApproveOperator(ctx context.Context, commanderID uuid.UUID, targetUserID uuid.UUID) error {
	var user models.User
	if err := s.db.First(&user, targetUserID).Error; err != nil {
		return err
	}

	if user.Status != models.StatusPendingApproval {
		return errors.New("کاربر در وضعیت انتظار تایید قرار ندارد")
	}

	if user.CreatedByID == nil || *user.CreatedByID != commanderID {
		return errors.New("شما مجوز تایید این کاربر را ندارید")
	}

	user.Status = models.StatusActive
	return s.db.Save(&user).Error
}
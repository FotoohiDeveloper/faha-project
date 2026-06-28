package service

import (
	"context"
	"encoding/json"
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

	// سریالایز sessionData به JSON برای ذخیره در Redis
	sessionBytes, err := json.Marshal(sessionData)
	if err != nil {
		return nil, nil, errors.New("خطا در پردازش داده‌های سشن WebAuthn")
	}

	sessionKey := fmt.Sprintf("webauthn_reg_session:%s", username)
	if err := s.redis.Set(ctx, sessionKey, sessionBytes, 5*time.Minute).Err(); err != nil {
		return nil, nil, errors.New("خطا در ذخیره سشن موقت")
	}

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

// ۵. شروع فرآیند لاگین با Passkey
func (s *AuthService) BeginLogin(ctx context.Context, username string) (*models.User, *protocol.CredentialAssertion, error) {
	var user models.User
	// همراه کاربر، کلیدهای ثبت شده‌اش رو هم واکشی می‌کنیم
	if err := s.db.Preload("Credentials").Where("username = ?", username).First(&user).Error; err != nil {
		return nil, nil, errors.New("کاربر یافت نشد")
	}

	if user.Status != models.StatusActive {
		return nil, nil, errors.New("حساب کاربری شما فعال نیست (شاید در انتظار تایید فرمانده است)")
	}

	// تولید Challenge برای تایید هویت
	assertion, sessionData, err := s.webAuthn.BeginLogin(&user)
	if err != nil {
		return nil, nil, err
	}

	// سریالایز sessionData به JSON برای ذخیره در Redis
	sessionBytes, err := json.Marshal(sessionData)
	if err != nil {
		return nil, nil, errors.New("خطا در پردازش داده‌های سشن WebAuthn")
	}

	// ذخیره SessionData موقت در ردیس برای 5 دقیقه
	sessionKey := fmt.Sprintf("webauthn_login_session:%s", username)
	if err := s.redis.Set(ctx, sessionKey, sessionBytes, 5*time.Minute).Err(); err != nil {
		return nil, nil, errors.New("خطا در ذخیره سشن موقت")
	}

	return &user, assertion, nil
}

// ۶. پایان فرآیند لاگین (بررسی امضای سخت‌افزاری و تولید توکن/سشن)
func (s *AuthService) FinishLogin(ctx context.Context, username string, response *protocol.ParsedCredentialAssertionData, sessionData webauthn.SessionData) (string, error) {
	var user models.User
	if err := s.db.Preload("Credentials").Where("username = ?", username).First(&user).Error; err != nil {
		return "", errors.New("کاربر یافت نشد")
	}

	// بررسی امضای ارسال شده از دانگل با اطلاعات دیتابیس
	credential, err := s.webAuthn.ValidateLogin(&user, sessionData, response)
	if err != nil {
		return "", errors.New("تایید کلید سخت‌افزاری ناموفق بود: " + err.Error())
	}

	// آپدیت کردن SignCount برای جلوگیری از حملات Replay
	s.db.Model(&models.WebAuthnCred{}).Where("credential_id = ?", credential.ID).Update("sign_count", credential.Authenticator.SignCount)

	// تولید شناسه سشن (Session Token) تصادفی امن
	sessionToken := uuid.New().String()

	// ذخیره سشن در Redis با انقضای 12 ساعت
	redisSessionKey := fmt.Sprintf("auth_session:%s", sessionToken)
	s.redis.Set(ctx, redisSessionKey, user.ID.String(), 12*time.Hour)

	return sessionToken, nil
}

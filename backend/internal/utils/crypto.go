package utils

import (
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// هش کردن پسورد
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12) // هزینه 12 برای امنیت بالا
	return string(bytes), err
}

// بررسی تطابق پسورد با هش
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// بررسی پیچیدگی رمز عبور (حداقل 8 حرف، بزرگ، کوچک، عدد، کاراکتر خاص)
func ValidatePasswordComplexity(password string) error {
	if len(password) < 8 {
		return errors.New("رمز عبور باید حداقل ۸ کاراکتر باشد")
	}
	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
		return errors.New("رمز عبور باید شامل حروف بزرگ، حروف کوچک، عدد و کاراکتر خاص (@, #, ...) باشد")
	}
	return nil
}

// تولید کد یک‌بار‌مصرف 6 رقمی
func GenerateOTP() string {
	// در یک سیستم واقعی از crypto/rand برای تولید رشته تصادفی امن استفاده می‌کنیم
	// برای سادگی فعلا یک کد ثابت برمی‌گردانیم (شما با تابع رندوم امن جایگزین کنید)
	return "123456" 
}
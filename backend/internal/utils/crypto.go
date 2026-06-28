package utils

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
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

// تولید کد یک‌بار‌مصرف 6 رقمی با crypto/rand (امن رمزنگاری)
func GenerateOTP() string {
	max := big.NewInt(1_000_000) // بازه 0 تا 999999
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// در صورت خطای سیستمی غیرمنتظره، panic بهتر از بازگشت مقدار ناامن است
		panic("crypto/rand failed: " + err.Error())
	}
	// قالب‌بندی با صفر پیشرو تا 6 رقم
	return fmt.Sprintf("%06d", n.Int64())
}

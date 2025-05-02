package models

import (
	"errors"
	"regexp"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User представляет собой модель пользователя
type User struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"unique;not null"`
	Password string `gorm:"not null"`
}

// BeforeSave хеширует пароль перед сохранением пользователя
func (u *User) BeforeSave(tx *gorm.DB) error {
	// Валидация пароля
	if err := ValidatePassword(u.Password); err != nil {
		return err
	}

	hashedPassword, err := HashPassword(u.Password)
	if err != nil {
		return err
	}
	u.Password = hashedPassword
	return nil
}

// HashPassword хеширует пароль
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// ComparePassword сравнивает хешированный пароль с введенным паролем
func ComparePassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// ValidatePassword проверяет, соответствует ли пароль заданным требованиям
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString
	hasLower := regexp.MustCompile(`[a-z]`).MatchString
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString
	hasSpecial := regexp.MustCompile(`[!@#\$%^&*(),.?":{}|<>]`).MatchString

	if !hasUpper(password) || !hasLower(password) || !hasNumber(password) || !hasSpecial(password) {
		return errors.New("password must contain at least one uppercase letter, one lowercase letter, one number, and one special character")
	}

	return nil
}

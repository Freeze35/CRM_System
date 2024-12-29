package transport_rest

import (
	"github.com/go-playground/validator/v10"
	"regexp"
)

// Регулярное выражение для проверки email
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Валидация email
func validateEmail(fl validator.FieldLevel) bool {
	email := fl.Field().String()
	if email == "" {
		return false // Пустое значение невалидно
	}
	return emailRegex.MatchString(email)
}

// Кастомный валидатор для phone
func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	if phone == "" {
		return true // Если поле пустое, оно считается валидным
	}

	// Простая проверка на длину или формат (например, только цифры)
	// Здесь можно добавить вашу логику проверки
	return len(phone) >= 10 && len(phone) <= 15
}

package transport_rest

import (
	"github.com/go-playground/validator/v10"
	"regexp"
	"strings"
)

// Кастомный валидатор для phone
func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()

	// Если поле пустое, оно считается валидным (можно изменить по необходимости)
	if phone == "" {
		return true
	}

	// Убираем символ '+' из телефона
	phone = strings.ReplaceAll(phone, "+", "")

	// Регулярное выражение для проверки, что строка состоит только из цифр и длина от 10 до 15
	re := regexp.MustCompile(`^[0-9]{10,15}$`)

	// Проверка по регулярному выражению
	return re.MatchString(phone)
}

func validatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	return len(password) >= 8 // Проверяем минимальную длину
}

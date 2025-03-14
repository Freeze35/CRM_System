package utils

import (
	"fmt"
	"net/http"
)

func GetFromCookies(r *http.Request, nameCookie string) (string, error) {
	cookie, err := r.Cookie(nameCookie)
	if err != nil {
		if err == http.ErrNoCookie {
			return "", fmt.Errorf(fmt.Sprintf("%s не найден", nameCookie))
		}
		// Обрабатываем другие ошибки
		return "", fmt.Errorf(fmt.Sprintf("ошибка создания %s", nameCookie))
	}

	return cookie.Value, nil
}

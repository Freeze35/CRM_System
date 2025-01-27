package utils

import (
	"fmt"
	"net/http"
)

func GetFromCookies(w http.ResponseWriter, r *http.Request, nameCookie string) string {
	cookie, err := r.Cookie(nameCookie)
	if err != nil {
		if err == http.ErrNoCookie {
			// Если cookie не найдена, возвращаем ошибку
			CreateError(w, http.StatusBadRequest, fmt.Sprintf("%s не найден", nameCookie), err)
			return ""
		}
		// Обрабатываем другие ошибки
		CreateError(w, http.StatusInternalServerError, fmt.Sprintf("Не удалось получить %s", nameCookie), err)
		return ""
	}

	return cookie.Value
}

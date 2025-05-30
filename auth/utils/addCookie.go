package utils

import (
	"log"
	"net/http"
)

func AddCookie(w http.ResponseWriter, name string, value string, maxTime ...int) {

	// 1 час
	time := 3600

	// Проверка на необязательный параметр time
	if len(maxTime) != 0 && time != maxTime[0] {
		time = maxTime[0]
	}

	// Устанавливаем HttpOnly Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // Только через HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   time,
	})

	log.Printf("Cookie %s added successfully", name) // Лог после установки
}

package utils

import (
	"fmt"
	"log"
	"net/http"
)

// RecoverMiddleware - обертка для обработки паники в HTTP-обработчиках
func RecoverMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				// Логируем панику
				log.Printf("Произошла паника: %v", rec)
				// Возвращаем клиенту сообщение об ошибке
				CreateError(w, http.StatusInternalServerError, "Произошла непредвиденная ошибка", fmt.Errorf("%v", rec))
			}
		}()
		// Вызываем оригинальный обработчик
		next(w, r)
	}
}

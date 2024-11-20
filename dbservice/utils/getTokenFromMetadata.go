package utils

import (
	"context"
	"fmt"
	"google.golang.org/grpc/metadata"
	"log"
)

func GetTokenFromMetadata(ctx context.Context) (string, error) {
	// Извлекаем токен из метаданных запроса
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Println("Нет метаданных в запросе Токен")
	}

	// Токен ожидается в заголовке Authorization
	token := ""
	if val, exists := md["authorization"]; exists {
		// Ожидаем, что токен будет в виде "Bearer <token>"
		token = val[0] // Считаем, что первый элемент в списке — это сам токен
		return token, nil
	} else {
		log.Println("Токен не найден в метаданных")
		return token, fmt.Errorf("Токен не найден в метаданных")
	}
}

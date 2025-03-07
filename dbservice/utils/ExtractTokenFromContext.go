package utils

import (
	"context"
	"fmt"

	"google.golang.org/grpc/metadata"
)

// Функция извлечения токена из контекста gRPC
func ExtractTokenFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("метаданные не найдены")
	}

	// Ищем ключ "authorization"
	authHeaders, exists := md["authorization"]
	if !exists || len(authHeaders) == 0 {
		return "", fmt.Errorf("токен отсутствует")
	}

	// Возвращаем сам токен без "Bearer "
	return authHeaders[0][7:], nil
}

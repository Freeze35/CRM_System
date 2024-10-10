package utils

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"os"
	"time"
)

var ctx = context.Background()

// Функция для сохранения данных в Redis
func saveToRedis(username string, token string) error {
	addRedis := os.Getenv("REDIS_ADDRESS")
	rdb := redis.NewClient(&redis.Options{
		Addr:     addRedis,
		Password: "", // если у вас установлен пароль для Redis, укажите его здесь
		DB:       0,  // использовать стандартную базу данных
	})

	// Сохраняем токен пользователя с TTL (время жизни) 1 час
	err := rdb.Set(ctx, username, token, 1*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("ошибка сохранения в Redis: %w", err)
	}

	fmt.Println("Токен успешно сохранен в Redis")
	return nil
}

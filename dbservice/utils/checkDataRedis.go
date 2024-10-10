package utils

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	"log"
	"os"
)

func checkDataRedis() string {
	// Подключение к Redis
	redisAddress := os.Getenv("REDIS_ADDRESS")

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: "",
		DB:       0,
	})

	// Ключ для хранения данных в Redis
	redisKey := "user:test_user"

	// Попытка получить данные из Redis
	cachedData, err := rdb.Get(ctx, redisKey).Result()

	//возвращает пустую строку в случае неудачи
	if err == redis.Nil {
		fmt.Println("Данные не найдены в Redis")
		return ""
	} else if err != nil {
		log.Fatalf("Ошибка при попытке получить данные из Redis: %v", err)
		return ""
	}

	return cachedData
}

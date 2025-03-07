package main

import (
	"context"
	pb "crmSystem/proto/redis"
	"crmSystem/utils"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type server struct {
	pb.UnimplementedRedisServiceServer
	RedisClient *redis.Client
}

func NewRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: "redis:" + os.Getenv("REDIS_PORT"), // Адрес Redis из docker-compose
	})
}

func (s *server) Save(ctx context.Context, req *pb.SaveRedisRequest) (*pb.SaveRedisResponse, error) {

	//Считываем указанное время существования кэша
	expiration := time.Duration(req.Expiration)

	//С помощью клиента для редис кэша сохраняем данные
	success, err := s.RedisClient.SetNX(ctx, req.Key, req.Value, expiration).Result()
	if err != nil {

		return &pb.SaveRedisResponse{
			Message: "Ошибка при сохранении: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}, err
	}

	//Сохранение прошло успешно?
	if !success {
		return &pb.SaveRedisResponse{
			Message: "Ключ уже существует для сохранения и не истёк",
			Status:  http.StatusConflict,
		}, nil
	}

	return &pb.SaveRedisResponse{
		Message: "Ключ сохранён успешно",
		Status:  http.StatusOK,
	}, nil
}

func (s *server) Get(ctx context.Context, req *pb.GetRedisRequest) (*pb.GetRedisResponse, error) {

	//Получаем значение по ключу
	value, err := s.RedisClient.Get(ctx, req.Key).Result()

	if err == redis.Nil {
		return &pb.GetRedisResponse{
			Message: "Ключ не найден",
			Status:  http.StatusNotFound,
		}, nil
	} else if err != nil {
		return &pb.GetRedisResponse{
			Message: "Ошибка при получении данных по ключу: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}, err
	}

	return &pb.GetRedisResponse{
		Message: value,
		Status:  http.StatusOK,
	}, nil
}

func main() {

	// Загружаем переменные из .env файла
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	//Создаём соединение с редис кэшем
	redisClient := NewRedisClient()
	defer func(redisClient *redis.Client) {
		err := redisClient.Close()
		if err != nil {
			log.Fatalf("Некорректное подключение к redis")
		}
	}(redisClient)

	//Создаём соединение tcp для прослушивания входящих Grpc запросов
	listener, err := net.Listen("tcp", ":"+os.Getenv("GRPC_PORT"))
	if err != nil {
		log.Fatalf("Ошибка подключения: %v", err)
	}

	//Подключаем ssl сертификацию для https
	var opts []grpc.ServerOption

	tlsCredentials, err := utils.LoadTLSCredentials()
	if err != nil {
		log.Fatalf("Невозможно загрузить учетные данные TLS: %s", err)
	}
	opts = []grpc.ServerOption{
		grpc.Creds(tlsCredentials), // Добавление TLS опций
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     5 * time.Minute,
			MaxConnectionAge:      15 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Minute,
			Time:                  5 * time.Second, // Таймаут на соединение
		}),
	}

	//Создаём Grpc Сервер
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterRedisServiceServer(grpcServer, &server{RedisClient: redisClient})

	log.Println("gRPC server for RedisService is running on port" + os.Getenv("GRPC_PORT"))
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Проблема в запуске сервера: %v", err)
	}
}

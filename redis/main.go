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

	expiration := time.Duration(req.Expiration) * time.Minute * 10
	success, err := s.RedisClient.SetNX(ctx, req.Key, req.Value, expiration).Result()
	if err != nil {
		return &pb.SaveRedisResponse{
			Message: "Failed to save key: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}, err
	}

	if !success {
		return &pb.SaveRedisResponse{
			Message: "Key already exists",
			Status:  http.StatusConflict,
		}, nil
	}

	return &pb.SaveRedisResponse{
		Message: "Key saved successfully",
		Status:  http.StatusOK,
	}, nil
}

func (s *server) Get(ctx context.Context, req *pb.GetRedisRequest) (*pb.GetRedisResponse, error) {

	value, err := s.RedisClient.Get(ctx, req.Key).Result()
	if err == redis.Nil {
		return &pb.GetRedisResponse{
			Message: "Key not found",
			Status:  http.StatusNotFound,
		}, nil
	} else if err != nil {
		return &pb.GetRedisResponse{
			Message: "Failed to get key: " + err.Error(),
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

	redisClient := NewRedisClient()
	defer redisClient.Close()

	listener, err := net.Listen("tcp", ":"+os.Getenv("GRPC_PORT"))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	//Подключаем ssl сертификацию для https
	var opts []grpc.ServerOption
	tlsCredentials, err := utils.LoadTLSCredentials()
	if err != nil {
		log.Fatalf("cannot load TLS credentials: %s", err)
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

	grpcServer := grpc.NewServer(opts...)
	pb.RegisterRedisServiceServer(grpcServer, &server{RedisClient: redisClient})

	log.Println("gRPC server for RedisService is running on port" + os.Getenv("GRPC_PORT"))
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

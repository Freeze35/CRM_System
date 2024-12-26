package main

import (
	"crmSystem/grpc_service"
	"crmSystem/proto/auth"
	"crmSystem/transport_rest"
	"crmSystem/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

func main() {
	grpcPort := os.Getenv("AUTH_SERVICE_GRPC_PORT")
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Не удалось запустить сервер: %v", err)
	}

	// Загрузка TLS-учетных данных для gRPC и HTTP
	tlsConfig, err := utils.LoadTLSCredentials()
	if err != nil {
		log.Fatalf("Невозможно загрузить учетные данные TLS: %s", err)
	}

	// Настройки для gRPC
	var opts []grpc.ServerOption
	opts = []grpc.ServerOption{
		grpc.Creds(credentials.NewTLS(tlsConfig)), // Добавление TLS опций для gRPC
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     5 * time.Minute,
			MaxConnectionAge:      15 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Minute,
			Time:                  5 * time.Second, // Таймаут на соединение
		}),
	}

	// Создаем gRPC сервер
	grpcServer := grpc.NewServer(opts...)
	grpcService := grpc_service.NewGRPCService()

	// Регистрируем наш AuthServiceServer
	auth.RegisterAuthServiceServer(grpcServer, grpcService)

	// Включаем отражение для gRPC
	reflection.Register(grpcServer)

	log.Printf("gRPC сервер запущен на %s с TLS", ":"+grpcPort)

	// HTTP сервер с обработчиком
	handler := transport_rest.NewHandler()

	httpPort := os.Getenv("AUTH_SERVICE_HTTP_PORT")

	// Создаем HTTP сервер с TLS
	httpServer := &http.Server{
		Addr:    ":" + httpPort,
		Handler: handler.InitRouter(),
	}

	// Запускаем HTTP сервер с TLS в отдельной горутине
	go func() {
		log.Println("HTTP SERVER STARTED WITH TLS ON PORT", httpPort)
		if err := httpServer.ListenAndServeTLS(utils.ServerCertFile, utils.ServerKeyFile); err != nil {
			log.Fatalf("Ошибка запуска HTTP сервера с TLS: %v", err)
		}
	}()

	// Запуск gRPC сервера
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Ошибка запуска gRPC сервера: %v", err)
	}
}

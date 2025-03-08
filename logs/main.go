package main

import (
	logsservice "crmSystem/service"
	"crmSystem/utils"
	"fmt"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"log"
	"net"
	"net/url"
	"os"
	"time"

	pb "crmSystem/proto/logs" // Замените на путь к вашему proto-пакету
	"github.com/grafana/loki-client-go/loki"
	"github.com/grafana/loki-client-go/pkg/urlutil"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

func main() {
	// Загружаем переменные из .env файла
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Настройка URL для Loki
	lokiURL := os.Getenv("LOKI_URL")

	// Parse the string into a *url.URL
	parsedURL, err := url.Parse(lokiURL)
	if err != nil {
		log.Fatal(err) // Handle error properly
	}

	CertClient := utils.LokiHttpCertClient()

	// Создаем клиента для отправки логов в Loki
	client, err := loki.New(loki.Config{
		URL:       urlutil.URLValue{URL: parsedURL}, // URL Loki
		BatchWait: 5 * time.Second,
		BatchSize: 1000,
		Timeout:   time.Duration(10) * time.Second,
		Client:    *CertClient,
	})

	if err != nil {
		log.Fatalf("Ошибка создания клиента Loki: %v", err)
	}
	defer client.Stop()

	grpcPort := os.Getenv("LOGS_SERVICE_HTTP_PORT")
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
		grpc.UnaryInterceptor(utils.RecoveryInterceptor),
	}

	// Создаем gRPC сервер
	grpcServer := grpc.NewServer(opts...)

	// Регистрируем AdminService с привязкой к общему переданному пулу соединений
	logsService := logsservice.NewGRPCDBLogsService(lokiURL)
	pb.RegisterLogsServiceServer(grpcServer, logsService)

	// Запуск сервера
	log.Println(fmt.Sprintf("gRPC сервер запускается на %s", os.Getenv("LOGS_SERVICE_HTTP_PORT")))
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Ошибка запуска gRPC сервера: %v", err)
	}
}

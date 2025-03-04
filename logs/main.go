package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"time"

	pb "crmSystem/proto/logs" // Замените на путь к вашему proto-пакету
	"github.com/grafana/loki-client-go/loki"
	"github.com/grafana/loki-client-go/pkg/urlutil"
	"github.com/joho/godotenv"
	"github.com/prometheus/common/model"
	"google.golang.org/grpc"
)

// GreeterServer реализует интерфейс gRPC-сервера
type GreeterServer struct {
	pb.UnimplementedLogsServiceServer
	lokiClient *loki.Client
}

func (s *GreeterServer) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	// Логирование в Loki
	labels := model.LabelSet{
		model.LabelName("job"):     model.LabelValue("go-microservice"),
		model.LabelName("level"):   model.LabelValue("info"),
		model.LabelName("handler"): model.LabelValue("say_hello"),
	}
	logMsg := fmt.Sprintf("Получен gRPC запрос от %s", req.GetName())
	err := s.lokiClient.Handle(labels, time.Now(), logMsg)
	if err != nil {
		return nil, err
	}

	// Ответ клиенту
	resp := &pb.HelloResponse{
		Message: "Привет! Это микросервис на Go.",
	}
	return resp, nil
}

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

	// Создаем клиента для отправки логов в Loki
	client, err := loki.New(loki.Config{
		URL:       urlutil.URLValue{URL: parsedURL}, // URL Loki
		BatchWait: 5 * time.Second,
		BatchSize: 1000,
	})
	if err != nil {
		log.Fatalf("Ошибка создания клиента Loki: %v", err)
	}
	defer client.Stop()

	// Создаем gRPC-сервер
	grpcServer := grpc.NewServer()
	server := &GreeterServer{lokiClient: client}

	// Регистрируем сервис
	pb.RegisterLogsServiceServer(grpcServer, server)

	// Запуск сервера на порту :8080
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", os.Getenv("LOGS_SERVICE_HTTP_PORT")))
	if err != nil {
		log.Fatalf("Ошибка запуска listener: %v", err)
	}
	log.Println("gRPC сервер запускается на :8080...")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Ошибка запуска gRPC сервера: %v", err)
	}
}

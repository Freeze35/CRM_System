package main

import (
	"crmSystem/internal/handler"
	"crmSystem/internal/service"
	pb "crmSystem/proto/email-service"
	"crmSystem/utils"
	"errors"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"log"
	"net"
	"os"
	"time"
)

// ConnectToRabbitMQ выполняет попытки подключения к RabbitMQ
func ConnectToRabbitMQ(rabbitMQURL string, retryCount int, retryInterval time.Duration) (*amqp.Connection, error) {
	var conn *amqp.Connection
	var err error

	for i := 0; i < retryCount; i++ {
		// Пытаемся подключиться
		conn, err = amqp.Dial(rabbitMQURL)
		if err == nil {
			log.Printf("Успешное подключение к RabbitMQ на попытке %d/%d", i+1, retryCount)
			return conn, nil
		}

		// Логируем ошибку и ждем перед следующей попыткой
		log.Printf("Не удалось подключиться к RabbitMQ, попытка %d/%d: %v", i+1, retryCount, err)
		time.Sleep(retryInterval)
	}

	// Если все попытки провалились, возвращаем ошибку
	return nil, errors.New("не удалось подключиться к RabbitMQ после нескольких попыток")
}

func main() {

	// Загружаем переменные из .env файла
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Подключение к RabbitMQ
	// Загружаем URL RabbitMQ из переменных окружения
	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	if rabbitMQURL == "" {
		log.Fatal("Переменная окружения RABBITMQ_URL не установлена")
	}

	// Параметры повторных попыток подключения
	retryCount := 10
	retryInterval := 4 * time.Second

	// Подключение к RabbitMQ
	conn, err := ConnectToRabbitMQ(rabbitMQURL, retryCount, retryInterval)
	if err != nil {
		log.Fatalf("Ошибка подключения к RabbitMQ: %v", err)
	} else {
		defer conn.Close()
	}

	// Открытие канала
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Не удалось открыть канал: %v", err)
	}
	defer ch.Close()

	// Объявляем очередь, из которой будем получать сообщения
	q, err := ch.QueueDeclare(
		"email_tasks", // Имя очереди
		true,          // Очередь устойчива к сбоям
		false,         // Очередь не будет удаляться после использования
		false,         // Очередь не будет блокирующей
		false,         // Не будем указывать дополнительные параметры
		nil,
	)
	if err != nil {
		log.Fatalf("Не удалось объявить очередь: %v", err)
	}

	// Запуск нескольких воркеров (потребителей)
	numWorkers := 5
	for i := 0; i < numWorkers; i++ {
		go handler.StartConsumer(ch, q.Name, service.NewEmailService(), i)
	}

	// Настроим gRPC сервер для общения с другими микросервисами
	grpcPort := os.Getenv("EMAIL_SERVICE_HTTP_PORT")
	listener, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Не удалось начать прослушивание на порту %s: %v", grpcPort, err)
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
	pb.RegisterEmailServiceServer(grpcServer, service.NewEmailService())

	log.Printf("EmailService работает на порту %s", grpcPort)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Ошибка при запуске gRPC: %v", err)
	}
}

package main

import (
	"crmSystem/internal/handler"
	"crmSystem/internal/service"
	pb "crmSystem/proto/email-service"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
)

func main() {

	// Загружаем переменные из .env файла
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Подключение к RabbitMQ
	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	log.Println(rabbitMQURL)
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		log.Fatalf("Не удалось подключиться к RabbitMQ: %v", err)
	}
	defer conn.Close()

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

	// Регистрация gRPC сервера
	server := grpc.NewServer()
	pb.RegisterEmailServiceServer(server, service.NewEmailService())

	log.Printf("EmailService работает на порту %s", grpcPort)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Ошибка при запуске gRPC: %v", err)
	}
}

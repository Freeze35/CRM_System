package handler

import (
	"context"
	"crmSystem/internal/service"
	pb "crmSystem/proto/email-service"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
)

// StartConsumer будет слушать очередь и обрабатывать сообщения
func StartConsumer(ch *amqp.Channel, queueName string, emailService *service.EmailService, workerID int) {
	msgs, err := ch.Consume(
		queueName, // Имя очереди
		"",        // Имя потребителя
		true,      // Автоматическое подтверждение
		false,     // Не эксклюзивное соединение
		false,     // Не заблокированное соединение
		false,     // Не ожидать сообщений
		nil,       // Дополнительные параметры
	)
	if err != nil {
		log.Fatalf("Не удалось начать потребление сообщений: %v", err)
	}

	for msg := range msgs {
		log.Printf("Worker %d: Получено сообщение: %s", workerID, string(msg.Body))

		// Создаем контекст для gRPC запроса (можно добавить тайм-ауты или отмену)
		ctx := context.Background()

		// Отправляем email
		resp, err := emailService.SendEmail(ctx, &pb.SendEmailRequest{
			Recipient: string(msg.Body), // Или другой параметр, в зависимости от структуры вашего сообщения
			Subject:   "Тема письма",
			Body:      "Текст письма",
		})

		if err != nil {
			log.Printf("Worker %d: Ошибка при отправке email: %v", workerID, err)
		} else {
			log.Printf("Worker %d: Email успешно отправлен: %v", workerID, resp.Message)
		}
	}
}

package main

import (
	"bufio"
	"crmSystem/utils/types"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/streadway/amqp"
	"log"
	"os"
	"strconv"
	"time"
)

func main() {
	// Загружаем переменные из файла .env
	err := godotenv.Load("/app/.env")
	if err != nil {
		log.Fatalf("Ошибка загрузки .env файла: %v", err)
	}

	chatID := flag.String("chat_id", "", "ID чата")
	userID := flag.String("user_id", "", "ID пользователя")
	flag.Parse()

	if *chatID == "" || *userID == "" {
		log.Fatal("Параметры chat_id и user_id обязательны")
	}

	parsedChatID, err := strconv.ParseInt(*chatID, 10, 64)
	if err != nil {
		log.Fatalf("Ошибка конвертации chat_id в int64: %v", err)
	}
	parsedUserID, err := strconv.ParseInt(*userID, 10, 64)
	if err != nil {
		log.Fatalf("Ошибка конвертации user_id в int64: %v", err)
	}

	RMQURL := os.Getenv("RABBITMQ_URL")

	conn, err := amqp.Dial(RMQURL)
	if err != nil {
		log.Fatalf("Не удалось подключиться к RabbitMQ: %v", err)
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Не удалось создать канал: %v", err)
	}
	defer channel.Close()

	exchangeName := "chat_exchange_" + strconv.FormatInt(parsedChatID, 10)

	// Создаем обменник fanout
	err = channel.ExchangeDeclare(
		exchangeName,
		"fanout", // Тип обменника
		true,     // Долговечность
		false,    // Автоудаление
		false,    // Внутренний
		false,    // No-wait
		nil,      // Аргументы
	)
	if err != nil {
		log.Fatalf("Ошибка создания обменника: %v", err)
	}

	// Создаем временную очередь
	queue, err := channel.QueueDeclare(
		"",    // Имя очереди (пустое имя создаст временную очередь)
		false, // Долговечность
		true,  // Автоудаление
		true,  // Эксклюзивность
		false, // No-wait
		nil,   // Аргументы
	)
	if err != nil {
		log.Fatalf("Ошибка создания очереди: %v", err)
	}

	// Привязываем очередь к обменнику
	err = channel.QueueBind(
		queue.Name,
		"",           // Routing key (игнорируется для fanout)
		exchangeName, // Имя обменника
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Ошибка привязки очереди к обменнику: %v", err)
	}

	// Запускаем получение сообщений
	go listenForMessages(channel, queue.Name)

	// Читаем и отправляем сообщения
	readAndSendMessages(channel, exchangeName, parsedUserID)
}

func listenForMessages(channel *amqp.Channel, queueName string) {
	messages, err := channel.Consume(
		queueName, // Имя очереди
		"",        // Тег потребителя
		true,      // Auto-ack
		false,     // Exclusive
		false,     // No-local
		false,     // No-wait
		nil,       // Аргументы
	)
	if err != nil {
		log.Fatalf("Ошибка при получении сообщений: %v", err)
	}

	fmt.Println("Ожидание сообщений...")

	for msg := range messages {
		var message types.ChatMessage
		err := json.Unmarshal(msg.Body, &message)
		if err != nil {
			log.Printf("Ошибка при разборе сообщения: %v", err)
			continue
		}
		fmt.Printf("[%s][User %d]: %s\n",
			message.Time.Format("2006-01-02 15:04:05"),
			message.UserID, message.Content)
	}
}

func readAndSendMessages(channel *amqp.Channel, exchangeName string, userID int64) {
	fmt.Println("Введите сообщение и нажмите Enter. Для выхода введите 'exit'.")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ") // Показываем приглашение ввода
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Ошибка при чтении ввода: %v", err)
			return
		}

		// Удаляем символ переноса строки
		message = message[:len(message)-1]

		// Если "exit" — выходим
		if message == "exit" {
			fmt.Println("Выход из чата...")
			break
		}

		// Удаляем ввод пользователя из консоли
		fmt.Print("\033[1A\033[K") // Поднимаемся на одну строку и очищаем её

		// Создаем сообщение
		chatMessage := types.ChatMessage{
			UserID:  userID,
			Content: message,
			Time:    time.Now(),
		}

		// Форматируем сообщение для вывода
		body, err := json.Marshal(chatMessage)
		if err != nil {
			log.Printf("Ошибка при сериализации сообщения: %v", err)
			continue
		}

		// Отправляем сообщение
		err = channel.Publish(
			exchangeName, // Имя обменника
			"",           // Routing key
			false,        // Mandatory
			false,        // Immediate
			amqp.Publishing{
				ContentType: "application/json",
				Body:        body,
			},
		)
		if err != nil {
			log.Printf("Ошибка при отправке сообщения: %v", err)
		}
	}
}

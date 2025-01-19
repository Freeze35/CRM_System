package main

import (
	"crmSystem/transport_rest"
	"crmSystem/utils"
	"errors"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

// ConnectToRabbitMQ выполняет попытки подключения к RabbitMQ
func ConnectToRabbitMQ(rabbitMQURL string, retryCount int, retryInterval time.Duration) (*amqp.Connection, error) {
	var conn *amqp.Connection
	var err error

	// Загрузите TLS-учетные данные, если необходимо
	tlsConfig, err := utils.LoadTLSCredentials() // Реализуйте эту функцию для загрузки сертификатов
	if err != nil {
		log.Fatalf("Ошибка загрузки TLS-настроек: %v", err)
	}

	// Создаем amqp.DialConfig с TLS
	for i := 0; i < retryCount; i++ {
		conn, err = amqp.DialConfig(rabbitMQURL, amqp.Config{
			TLSClientConfig: tlsConfig, // Используем конфигурацию TLS
		})
		if err == nil {
			log.Printf("Успешное подключение к RabbitMQ на попытке %d/%d", i+1, retryCount)
			return conn, nil
		}

		log.Printf("Не удалось подключиться к RabbitMQ, попытка %d/%d: %v", i+1, retryCount, err)
		time.Sleep(retryInterval)
	}

	return nil, errors.New("не удалось подключиться к RabbitMQ после нескольких попыток")
}

func main() {

	// Загружаем переменные из файла .env
	err := godotenv.Load("/app/.env")
	if err != nil {
		log.Fatalf("Ошибка загрузки .env файла: %v", err)
	}

	// Подключение к RabbitMQ
	// Загружаем URL RabbitMQ из переменных окружения
	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	if rabbitMQURL == "" {
		log.Fatal("Переменная окружения RABBITMQ_URL не установлена")
	}

	// Параметры повторных попыток подключения
	retryCount := 10
	retryInterval := 3 * time.Second

	// Подключение к RabbitMQ
	rabbitMQConn, err := ConnectToRabbitMQ(rabbitMQURL, retryCount, retryInterval)
	if err != nil {
		log.Fatalf("Ошибка подключения к RabbitMQ: %v", err)
	} else {
		defer rabbitMQConn.Close()
	}

	// Создаем HTTP обработчик с передачей RabbitMQ соединения
	handler := transport_rest.NewHandler(rabbitMQConn)

	// Получаем порт из переменных окружения
	httpPort := os.Getenv("CHAT_SERVICE_HTTP_PORT")

	// Создаем HTTP сервер
	httpServer := &http.Server{
		Addr:    ":" + httpPort,
		Handler: handler.InitRouter(),
	}

	// Используем WaitGroup для ожидания завершения сервера
	var wg sync.WaitGroup
	wg.Add(1)

	// Запускаем сервер с TLS в отдельной горутине
	go func() {
		log.Println("HTTP SERVER STARTED WITH TLS ON PORT", httpPort)
		if err := httpServer.ListenAndServeTLS(utils.ServerCertFile, utils.ServerKeyFile); err != nil {
			log.Fatalf("Ошибка запуска HTTP сервера с TLS: %v", err)
		}
		wg.Done() // Уменьшаем счетчик после завершения работы сервера
	}()

	// Ожидаем завершения работы HTTP сервера
	wg.Wait()
}

package main

import (
	"crmSystem/transport_rest"
	"crmSystem/utils"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"sync"
)

func main() {
	// Инициализируем TCP соединение для gRPC сервера

	// Загружаем переменные из файла .env
	err := godotenv.Load("/app/.env")
	if err != nil {
		log.Fatalf("Ошибка загрузки .env файла: %v", err)
	}

	port := os.Getenv("TIMER_SERVICE_PORT")

	// HTTP сервер с обработчиком
	handler := transport_rest.NewHandler()

	// Создаем HTTP сервер с TLS
	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: handler.InitRouter(),
	}

	// Используем WaitGroup для ожидания завершения сервера
	var wg sync.WaitGroup
	wg.Add(1)

	// Запускаем HTTP сервер с TLS в отдельной горутине
	go func() {
		log.Println("HTTP SERVER STARTED WITH TLS ON PORT", port)
		if err := httpServer.ListenAndServeTLS(utils.ServerCertFile, utils.ServerKeyFile); err != nil {
			log.Fatalf("Ошибка запуска HTTP сервера с TLS: %v", err)
		}
	}()

	// Ожидаем завершения работы HTTP сервера
	wg.Wait()
}

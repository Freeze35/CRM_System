package main

import (
	"crmSystem/transport_rest"
	"crmSystem/utils"
	"log"
	"net/http"
	"os"
)

func main() {

	// HTTP сервер с обработчиком
	handler := transport_rest.NewHandler()

	// Получаем порт из переменной окружения или устанавливаем значение по умолчанию
	httpPort := os.Getenv("ADMIN_SERVICE_HTTP_PORT")

	// Проверяем, что файлы сертификатов существуют
	if _, err := os.Stat(utils.ServerCertFile); os.IsNotExist(err) {
		log.Fatalf("Сертификат не найден: %v", err)
	}
	if _, err := os.Stat(utils.ServerKeyFile); os.IsNotExist(err) {
		log.Fatalf("Ключ не найден: %v", err)
	}

	// Создаем HTTP сервер с TLS
	httpServer := &http.Server{
		Addr:    ":" + httpPort,
		Handler: handler.InitRouter(),
	}

	// Запускаем HTTP сервер с TLS в основной горутине
	log.Println("HTTP SERVER STARTED WITH TLS ON PORT", httpPort)
	if err := httpServer.ListenAndServeTLS(utils.ServerCertFile, utils.ServerKeyFile); err != nil {
		log.Fatalf("Ошибка запуска HTTP сервера с TLS: %v", err)
	}

	// Программа продолжает работать, пока сервер работает
}

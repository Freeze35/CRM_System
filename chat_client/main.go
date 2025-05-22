package main

import (
	chat "crmSystem/internal"
	"crmSystem/utils"
	"flag"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {
	err := godotenv.Load("/app/.env")
	if err != nil {
		log.Fatalf("Ошибка загрузки .env файла: %v", err)
	}

	chatID := flag.String("chat_id", "", "ID чата")
	userID := flag.String("user_id", "", "ID пользователя")
	dbName := flag.String("db_name", "", "Имя Базы Данных")
	flag.Parse()

	if *chatID == "" || *userID == "" || *dbName == "" {
		log.Fatal("Параметры chat_id,user_id,dbName обязательны")
	}

	parsedChatID, err := strconv.ParseInt(*chatID, 10, 64)
	if err != nil {
		log.Fatalf("Ошибка конвертации chat_id в int64: %v", err)
	}
	parsedUserID, err := strconv.ParseInt(*userID, 10, 64)
	if err != nil {
		log.Fatalf("Ошибка конвертации user_id в int64: %v", err)
	}

	apiBaseURL := os.Getenv("CHATS_SERVICE_URL")
	if apiBaseURL == "" {
		apiBaseURL = "https://nginx:443"
	}

	tlsConfig, err := utils.LoadTLSCredentials()
	if err != nil {
		log.Fatalf("Ошибка настройки TLS: %v", err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	go chat.ListenForMessages(apiBaseURL, parsedChatID, parsedUserID, httpClient)
	go chat.ReadAndSendMessages(apiBaseURL, parsedChatID, parsedUserID, *dbName, httpClient)

	select {}
}

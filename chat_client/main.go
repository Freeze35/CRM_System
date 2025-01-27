package main

import (
	"bufio"
	"bytes"
	"crmSystem/utils"
	"crmSystem/utils/types"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/joho/godotenv"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	// Загружаем переменные из файла .env
	err := godotenv.Load("/app/.env")
	if err != nil {
		log.Fatalf("Ошибка загрузки .env файла: %v", err)
	}

	//Данные через командную строку при запуске программы
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

	// Настраиваем HTTPS-клиент
	tlsConfig, err := utils.LoadTLSCredentials()
	if err != nil {
		log.Fatalf("Ошибка настройки TLS: %v", err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// Запускаем чтение сообщений с сервера
	go listenForMessages(apiBaseURL, parsedChatID, parsedUserID, httpClient)

	// Читаем и отправляем сообщения
	go readAndSendMessages(apiBaseURL, parsedChatID, parsedUserID, *dbName, httpClient)

	// Чтобы главный поток не завершался
	select {}
}

func listenForMessages(apiBaseURL string, chatID int64, userID int64, httpClient *http.Client) {
	for {
		// Формируем URL для запроса
		reqURL := fmt.Sprintf("%s/chats/%d/messages", apiBaseURL, chatID)

		// Выполняем HTTP-запрос
		resp, err := httpClient.Get(reqURL)
		if err != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				resp.Body.Close() // Закрываем тело ответа, если есть
			}
			time.Sleep(500 * time.Millisecond) // Задержка перед повторным запросом
			continue
		}

		// Чтение тела ответа
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close() // Закрываем тело после чтения
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Разбор JSON
		var messages []types.ChatMessage
		if err := json.Unmarshal(body, &messages); err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Вывод сообщений
		for _, message := range messages {
			// Игнорируем сообщения от текущего пользователя
			if message.UserID == userID {
				continue
			}
			fmt.Printf("[%s][User %d]: %s\n",
				message.Time.Format("2006-01-02 15:04:05"),
				message.UserID, message.Content)
		}

		// Задержка перед следующим запросом
		time.Sleep(500 * time.Millisecond)
	}
}

func readAndSendMessages(apiBaseURL string, chatID, userID int64, dbName string, httpClient *http.Client) {
	fmt.Println("Введите сообщение и нажмите Enter. Для выхода введите 'exit'.")

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		message, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Ошибка при чтении ввода: %v", err)
			return
		}

		message = strings.TrimSpace(message)

		if message == "exit" {
			fmt.Println("Выход из чата...")
			break
		}

		chatMessage := types.ChatMessage{
			DBName:  dbName,
			ChatID:  chatID,
			UserID:  userID,
			Content: message,
			Time:    time.Now(),
		}
		body, err := json.Marshal(chatMessage)
		if err != nil {
			log.Printf("Ошибка при сериализации сообщения: %v", err)
			continue
		}

		// Формируем HTTPS-запрос для отправки сообщения
		req, err := http.NewRequest(
			"POST",
			fmt.Sprintf("%s/chats/%d/sendMessage", apiBaseURL, chatID),
			bytes.NewBuffer(body),
		)
		if err != nil {
			log.Printf("Ошибка при создании запроса: %v", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		// Выполняем HTTPS-запрос
		resp, err := httpClient.Do(req)
		if err != nil {
			log.Printf("Ошибка при отправке сообщения: %v", err)
			continue
		}
		defer resp.Body.Close()

		// Read the response body
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Ошибка при чтении тела ответа: %v", err)
			continue
		}

		// Check for server error
		if resp.StatusCode != http.StatusOK {
			log.Printf("Ошибка от сервера: %s, Статус: %d", respBody, resp.StatusCode)
			continue
		}

		// Форматированный вывод отправленного сообщения
		fmt.Printf("[You][%s]: %s\n",
			time.Now().Format("2006-01-02 15:04:05"), message)
	}
}

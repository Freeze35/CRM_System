package chat

import (
	"bufio"
	"bytes"
	"crmSystem/utils/types"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// ListenForMessages периодически запрашивает сообщения чата и выводит их в консоль.
func ListenForMessages(apiBaseURL string, chatID int64, userID int64, httpClient *http.Client) {
	for {
		reqURL := fmt.Sprintf("%s/chats/%d/messages", apiBaseURL, chatID)
		resp, err := httpClient.Get(reqURL)
		if err != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				resp.Body.Close()
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		var messages []types.ChatMessage
		if err := json.Unmarshal(body, &messages); err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		for _, message := range messages {
			if message.UserID == userID {
				continue
			}
			fmt.Printf("[%s][User %d]: %s\n",
				message.Time.Format("2006-01-02 15:04:05"),
				message.UserID, message.Content)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// ReadAndSendMessages читает сообщения из ввода и отправляет их через HTTP API.
func ReadAndSendMessages(apiBaseURL string, chatID, userID int64, dbName string, httpClient *http.Client) {
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
		resp, err := httpClient.Do(req)
		if err != nil {
			log.Printf("Ошибка при отправке сообщения: %v", err)
			continue
		}
		defer resp.Body.Close()
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Ошибка при чтении тела ответа: %v", err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			log.Printf("Ошибка от сервера: %s, Статус: %d", respBody, resp.StatusCode)
			continue
		}
		fmt.Printf("[You][%s]: %s\n",
			time.Now().Format("2006-01-02 15:04:05"), message)
	}
}

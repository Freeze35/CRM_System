package tests

import (
	"bytes"
	"context"
	chat "crmSystem/internal"
	"crmSystem/utils"
	"crmSystem/utils/types"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

// captureOutput перехватывает вывод в консоль
func captureOutput(f func()) string {
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestListenForMessages(t *testing.T) {
	// Настраиваем тестовый HTTPS-сервер
	server := httptest.NewTLSServer(nil)
	defer server.Close()

	// Настраиваем TLS-клиент с использованием utils.LoadTLSCredentials
	tlsConfig, err := utils.LoadTLSCredentials()
	if err != nil {
		t.Fatalf("Failed to load TLS credentials: %v", err)
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	chatID := int64(123)
	userID := int64(456)

	tests := []struct {
		name           string
		serverHandler  func(w http.ResponseWriter, r *http.Request)
		expectedOutput string
	}{
		{
			name: "Success",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != fmt.Sprintf("/chats/%d/messages", chatID) {
					http.Error(w, "Invalid request", http.StatusBadRequest)
					return
				}
				messages := []types.ChatMessage{
					{
						ChatID:  chatID,
						UserID:  789, // Другой пользователь
						Content: "Hello from user 789",
						Time:    time.Now().Truncate(time.Second),
					},
					{
						ChatID:  chatID,
						UserID:  userID, // Текущий пользователь
						Content: "Own message",
						Time:    time.Now().Truncate(time.Second),
					},
				}
				_ = json.NewEncoder(w).Encode(messages)
			},
			expectedOutput: fmt.Sprintf("[%s][User 789]: Hello from user 789\n", time.Now().Truncate(time.Second).Format("2006-01-02 15:04:05")),
		},
		{
			name: "Server Error",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Server error", http.StatusInternalServerError)
			},
			expectedOutput: "", // Нет вывода, так как запрос неуспешен
		},
		{
			name: "Invalid JSON",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("invalid json"))
			},
			expectedOutput: "", // Нет вывода, так как JSON некорректен
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем обработчик для сервера
			server.Config.Handler = http.HandlerFunc(tt.serverHandler)

			// Создаем контекст с таймаутом для ограничения времени теста
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			// Перехватываем вывод
			var output string
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				output = captureOutput(func() {
					chat.ListenForMessages(server.URL, chatID, userID, httpClient)
				})
			}()

			// Ждем завершения контекста
			<-ctx.Done()
			wg.Wait()

			// Проверяем вывод
			if tt.expectedOutput == "" {
				assert.Empty(t, output, "Expected no output")
			} else {
				assert.Contains(t, output, tt.expectedOutput, "Expected output to contain message")
			}
		})
	}
}

func TestReadAndSendMessages(t *testing.T) {
	// Настраиваем тестовый HTTPS-сервер
	server := httptest.NewTLSServer(nil)
	defer server.Close()

	// Настраиваем TLS-клиент
	tlsConfig, err := utils.LoadTLSCredentials()
	if err != nil {
		t.Fatalf("Failed to load TLS credentials: %v", err)
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	chatID := int64(123)
	userID := int64(456)
	dbName := "test_db"

	tests := []struct {
		name           string
		input          string
		serverHandler  func(w http.ResponseWriter, r *http.Request)
		expectedOutput string
		expectedStatus int
	}{
		{
			name:  "Success",
			input: "Hello, world!\n",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != fmt.Sprintf("/chats/%d/sendMessage", chatID) {
					http.Error(w, "Invalid request", http.StatusBadRequest)
					return
				}
				var msg types.ChatMessage
				if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
					http.Error(w, "Invalid JSON", http.StatusBadRequest)
					return
				}
				assert.Equal(t, chatID, msg.ChatID)
				assert.Equal(t, userID, msg.UserID)
				assert.Equal(t, dbName, msg.DBName)
				assert.Equal(t, "Hello, world!", msg.Content)
				w.WriteHeader(http.StatusOK)
			},
			expectedOutput: fmt.Sprintf("Введите сообщение и нажмите Enter. Для выхода введите 'exit'.\n> [You][%s]: Hello, world!\n", time.Now().Truncate(time.Second).Format("2006-01-02 15:04:05")),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Exit",
			input:          "exit\n",
			serverHandler:  func(w http.ResponseWriter, r *http.Request) {},
			expectedOutput: "Введите сообщение и нажмите Enter. Для выхода введите 'exit'.\n> Выход из чата...\n",
			expectedStatus: 0, // Не проверяем статус, так как запрос не отправляется
		},
		{
			name:  "Server Error",
			input: "Test message\n",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Server error", http.StatusInternalServerError)
			},
			expectedOutput: "Введите сообщение и нажмите Enter. Для выхода введите 'exit'.\n> Ошибка от сервера: Server error, Статус: 500\n",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем обработчик для сервера
			server.Config.Handler = http.HandlerFunc(tt.serverHandler)

			// Перехватываем вывод
			var output string
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				output = captureOutput(func() {
					// Временно заменяем os.Stdin для теста
					origStdin := os.Stdin
					defer func() { os.Stdin = origStdin }()
					os.Stdin, _ = os.CreateTemp("", "test-stdin")
					_, _ = os.Stdin.WriteString(tt.input)
					_ = os.Stdin.Sync()
					os.Stdin.Seek(0, 0)

					chat.ReadAndSendMessages(server.URL, chatID, userID, dbName, httpClient)
				})
			}()

			// Ждем завершения или таймаута
			time.Sleep(500 * time.Millisecond)
			wg.Wait()

			// Проверяем вывод
			assert.Contains(t, output, tt.expectedOutput, "Expected output to contain message")
		})
	}
}

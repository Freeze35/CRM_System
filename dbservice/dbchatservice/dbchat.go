package dbchatservice

import (
	"context"
	"crmSystem/proto/dbchat"
	"crmSystem/utils"
	"fmt"
	"log"
	"net/http"
	"os"
)

type ChatServiceServer struct {
	dbchat.UnsafeDbChatServiceServer
	connectionsMap *utils.MapConnectionsDB // Используем указатель
}

func NewGRPCDBChatService(mapConnect *utils.MapConnectionsDB) *ChatServiceServer {
	return &ChatServiceServer{
		connectionsMap: mapConnect,
	}
}

// SaveMessage сохраняет сообщение в базу данных.
func (s *ChatServiceServer) SaveMessage(ctx context.Context, req *dbchat.SaveMessageRequest) (*dbchat.SaveMessageResponse, error) {
	// Получаем строку подключения к базе данных из переменной окружения с именем базы данных.
	dsn := utils.DsnString(os.Getenv(req.DbName))

	// Получаем соединение с базой данных.
	db, err := s.connectionsMap.GetDb(dsn)
	if err != nil {
		// Если произошла ошибка подключения, логируем её и возвращаем ответ с ошибкой.
		log.Printf("Ошибка подключения к базе данных: %s", err)
		return &dbchat.SaveMessageResponse{
			Response: fmt.Sprintf("Ошибка подключения к базе данных: %s.", err), // Сообщение об ошибке.
			Status:   http.StatusInternalServerError,                            // Статус внутренней ошибки.
		}, err
	}

	// SQL-запрос для сохранения сообщения.
	query := `
        INSERT INTO messages (chat_id, user_id, message, created_at)
        VALUES ($1, $2, $3, to_timestamp($4)) RETURNING id;
    `

	// Переменная для ID сохраненного сообщения.
	var messageID int

	// Выполняем запрос с параметрами из запроса.
	err = db.QueryRowContext(ctx, query, req.ChatId, req.UserId, req.Message, req.CreatedAt).Scan(&messageID)
	if err != nil {
		// Если произошла ошибка при выполнении запроса, логируем её и возвращаем ответ с ошибкой.
		log.Printf("Ошибка сохранения сообщения в базу данных: %s", err)
		return &dbchat.SaveMessageResponse{
			Response: fmt.Sprintf("Ошибка сохранения сообщения в базу данных: %s.", err), // Сообщение об ошибке.
			Status:   http.StatusInternalServerError,                                     // Статус внутренней ошибки.
		}, err
	}

	// Возвращаем успешный ответ с ID сохраненного сообщения.
	return &dbchat.SaveMessageResponse{
		Response: fmt.Sprintf("Сообщение успешно сохранено с ID: %d", messageID), // Сообщение о успешном сохранении.
		Status:   http.StatusOK,                                                  // Статус успешного выполнения.
	}, nil
}

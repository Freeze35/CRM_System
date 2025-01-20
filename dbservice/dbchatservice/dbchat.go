package dbchatservice

import (
	"context"
	"crmSystem/proto/dbchat"
	"crmSystem/utils"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
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

// Метод для создания чата и добавления пользователя с транзакцией
func (s *ChatServiceServer) CreateChat(ctx context.Context, req *dbchat.CreateChatRequest) (*dbchat.CreateChatResponse, error) {
	log.Printf("CreateChat: %s", "CreateChat")
	// Получаем строку подключения к базе данных
	dsn := utils.DsnString(req.DbName)
	dbConnCompany, err := s.connectionsMap.GetDb(dsn)
	if err != nil || dbConnCompany == nil {
		log.Printf("Ошибка подключения к базе данных: %s", err)
		return &dbchat.CreateChatResponse{
			DbName:    req.DbName,
			Message:   fmt.Sprintf("Ошибка подключения к базе данных: %s", err),
			Status:    http.StatusInternalServerError,
			CreatedAt: time.Now().Unix(),
		}, err
	}

	// Начинаем транзакцию
	tx, err := dbConnCompany.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Ошибка начала транзакции: %s", err)
		return &dbchat.CreateChatResponse{
			DbName:    req.DbName,
			Message:   fmt.Sprintf("Ошибка начала транзакции: %s", err),
			Status:    http.StatusInternalServerError,
			CreatedAt: time.Now().Unix(),
		}, err
	}

	// Создаём новый чат
	createChatQuery := `INSERT INTO chats (chat_name) VALUES ($1) RETURNING id;`
	var chatID int64
	err = tx.QueryRowContext(ctx, createChatQuery, req.ChatName).Scan(&chatID)
	if err != nil {
		tx.Rollback()
		log.Printf("Ошибка создания чата: %s", err)
		return &dbchat.CreateChatResponse{
			DbName:    req.DbName,
			Message:   fmt.Sprintf("Ошибка создания чата: %s", err),
			Status:    http.StatusInternalServerError,
			CreatedAt: time.Now().Unix(),
		}, err
	}

	// Завершаем транзакцию на уровне создания чата
	err = tx.Commit()
	if err != nil {
		log.Printf("Ошибка при коммите транзакции создания чата: %s", err)
		return &dbchat.CreateChatResponse{
			DbName:    req.DbName,
			Message:   fmt.Sprintf("Ошибка при коммите создания чата: %s", err),
			Status:    http.StatusInternalServerError,
			CreatedAt: time.Now().Unix(),
		}, err
	}

	// Создаём запрос для добавления пользователей
	addUsersReq := &dbchat.AddUsersToChatRequest{
		ChatId:  chatID,
		DbName:  req.DbName,
		UsersId: req.UsersId,
	}

	// Вызываем метод AddUsersToChat
	addUsersResp, err := s.AddUsersToChat(ctx, addUsersReq)
	if err != nil {
		log.Printf("Ошибка добавления пользователей в чат: %v", err)
		return &dbchat.CreateChatResponse{
			ChatId:    chatID,
			DbName:    req.DbName,
			Message:   addUsersResp.Message,
			Status:    http.StatusInternalServerError,
			CreatedAt: time.Now().Unix(),
		}, err
	}

	// Успешный ответ
	return &dbchat.CreateChatResponse{
		ChatId:    chatID,
		DbName:    req.DbName,
		Message:   fmt.Sprintf("Чат '%s' успешно создан. %s", req.ChatName, addUsersResp.Message),
		Status:    http.StatusOK,
		CreatedAt: time.Now().Unix(),
	}, nil
}

func (s *ChatServiceServer) AddUsersToChat(ctx context.Context, req *dbchat.AddUsersToChatRequest) (*dbchat.AddUsersToChatResponse, error) {
	// Получаем строку подключения к базе данных
	dsn := utils.DsnString(req.DbName)
	dbConnCompany, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Printf("Ошибка подключения к базе данных: %s", err)
		return &dbchat.AddUsersToChatResponse{
			Message: fmt.Sprintf("Ошибка подключения к базе данных: %s", err),
			Status:  http.StatusInternalServerError,
		}, err
	}
	defer dbConnCompany.Close()

	// Начало транзакции
	tx, err := dbConnCompany.Begin()
	if err != nil {
		log.Printf("Ошибка начала транзакции: %s", err)
		return &dbchat.AddUsersToChatResponse{
			Message: fmt.Sprintf("Ошибка начала транзакции: %s", err),
			Status:  http.StatusInternalServerError,
		}, err
	}

	// Формируем данные для батчевого запроса
	var values []interface{}
	var placeholders []string
	for i, user := range req.UsersId {
		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		values = append(values, user.UserId, req.ChatId)
	}

	// Добавляем пользователей в таблицу chat_users с использованием батч-запроса
	addUserQuery := fmt.Sprintf(
		`INSERT INTO chat_users (user_id, chat_id) VALUES %s ON CONFLICT DO NOTHING;`,
		strings.Join(placeholders, ","),
	)

	// Выполняем батч-запрос
	_, err = tx.ExecContext(ctx, addUserQuery, values...)
	if err != nil {
		tx.Rollback() // Откат транзакции при ошибке
		log.Printf("Ошибка добавления пользователей в чат: %v", err)
		return &dbchat.AddUsersToChatResponse{
			Message: fmt.Sprintf("Ошибка добавления пользователей в чат %s", req.ChatId),
			Status:  http.StatusInternalServerError,
		}, err
	}

	// Подтверждаем транзакцию
	err = tx.Commit()
	if err != nil {
		log.Printf("Ошибка подтверждения транзакции: %s", err)
		return &dbchat.AddUsersToChatResponse{
			Message: fmt.Sprintf("Ошибка подтверждения транзакции: %s", err),
			Status:  http.StatusInternalServerError,
		}, err
	}

	// Успешный ответ
	return &dbchat.AddUsersToChatResponse{
		Message: fmt.Sprintf("Пользователи успешно добавлены в чат %s", req.ChatId),
		Status:  http.StatusOK,
	}, nil
}

func (s *ChatServiceServer) SaveMessage(ctx context.Context, req *dbchat.SaveMessageRequest) (*dbchat.SaveMessageResponse, error) {
	// Получаем строку подключения к базе данных
	dsn := utils.DsnString(req.DbName)
	dbConnCompany, err := s.connectionsMap.GetDb(dsn)
	if err != nil || dbConnCompany == nil {
		log.Printf("Ошибка подключения к базе данных: %s", err)
		return &dbchat.SaveMessageResponse{
			Message: fmt.Sprintf("Ошибка подключения к базе данных: %s", err),
			Status:  http.StatusInternalServerError,
		}, err
	}

	// SQL-запрос для вставки сообщения
	insertMessageQuery := `
        INSERT INTO messages (chat_id, user_id, message)
        VALUES ($1, $2, $3)
        RETURNING id, created_at;
    `

	// Переменные для получения возвращаемых значений
	var messageID int64
	var createdAt time.Time

	// Выполняем запрос
	err = dbConnCompany.QueryRowContext(ctx, insertMessageQuery, req.ChatId, req.UserId, req.Content).
		Scan(&messageID, &createdAt)
	if err != nil {
		log.Printf("Ошибка при сохранении сообщения: %s", err)
		return &dbchat.SaveMessageResponse{
			Message: fmt.Sprintf("Ошибка при сохранении сообщения: %s", err),
			Status:  http.StatusInternalServerError,
		}, err
	}

	// Успешный ответ
	return &dbchat.SaveMessageResponse{
		MessageId: messageID,
		ChatId:    req.ChatId,
		UserId:    req.UserId,
		Message:   req.Content,
		CreatedAt: createdAt.Unix(),
		Status:    http.StatusOK,
	}, nil
}

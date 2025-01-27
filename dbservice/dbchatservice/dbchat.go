package dbchatservice

import (
	"context"
	"crmSystem/proto/dbchat"
	"crmSystem/utils"
	"database/sql"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log"
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

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "Не удалось получить метаданные из контекста")
	}

	// Извлекаем DatabaseName из метаданных
	database := md["database"][0] // токен передается как "auth-token"
	if len(database) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "database не найдена в метаданных")
	}

	// Извлекаем userId из метаданных
	userId := md["user-id"][0] // токен передается как "auth-token"
	if len(userId) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "userId не найден в метаданных")
	}

	log.Printf("CreateChat: %s", "CreateChat")
	// Получаем строку подключения к базе данных
	dsn := utils.DsnString(database)
	dbConnCompany, err := s.connectionsMap.GetDb(dsn)
	if err != nil || dbConnCompany == nil {
		log.Printf("Ошибка подключения к базе данных: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка подключения к базе данных: %v", err))
	}

	// Начинаем транзакцию
	tx, err := dbConnCompany.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Ошибка начала транзакции: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка начала транзакции: %v", err))
	}

	// Создаём новый чат
	createChatQuery := `INSERT INTO chats (chat_name) VALUES ($1) RETURNING id;`
	var chatID int64
	err = tx.QueryRowContext(ctx, createChatQuery, req.ChatName).Scan(&chatID)
	if err != nil {
		tx.Rollback()
		log.Printf("Ошибка создания чата: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка создания чата: %v", err))
	}

	// Завершаем транзакцию на уровне создания чата
	err = tx.Commit()
	if err != nil {
		log.Printf("Ошибка при коммите транзакции создания чата: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка при коммите создания чата: %v", err))
	}

	// Создаём запрос для добавления пользователей
	addUsersReq := &dbchat.AddUsersToChatRequest{
		ChatId:  chatID,
		UsersId: req.UsersId,
	}

	// Вызываем метод AddUsersToChat
	addUsersResp, err := s.AddUsersToChat(ctx, addUsersReq)
	if err != nil {
		log.Printf("Ошибка добавления пользователей в чат: %v", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка добавления пользователей в чат: %v", err))
	}

	// Успешный ответ
	return &dbchat.CreateChatResponse{
		ChatId:    chatID,
		Message:   fmt.Sprintf("Чат '%s' успешно создан. %s", req.ChatName, addUsersResp.Message),
		CreatedAt: time.Now().Unix(),
	}, nil
}

func (s *ChatServiceServer) AddUsersToChat(ctx context.Context, req *dbchat.AddUsersToChatRequest) (*dbchat.AddUsersToChatResponse, error) {

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "Не удалось получить метаданные из контекста")
	}

	// Извлекаем DatabaseName из метаданных
	database := md["database"][0] // токен передается как "auth-token"
	if len(database) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "database не найдена в метаданных")
	}

	// Получаем строку подключения к базе данных
	dsn := utils.DsnString(database)
	dbConnCompany, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Printf("Ошибка подключения к базе данных: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка подключения к базе данных: %v", err))
	}
	defer dbConnCompany.Close()

	// Начало транзакции
	tx, err := dbConnCompany.Begin()
	if err != nil {
		log.Printf("Ошибка начала транзакции: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка начала транзакции: %v", err))
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
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка добавления пользователей в чат %v", req.ChatId))
	}

	// Подтверждаем транзакцию
	err = tx.Commit()
	if err != nil {
		log.Printf("Ошибка подтверждения транзакции: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка подтверждения транзакции: %s", err))
	}

	// Успешный ответ
	return &dbchat.AddUsersToChatResponse{
		Message: fmt.Sprintf("Пользователи успешно добавлены в чат %v", req.ChatId),
	}, nil
}

func (s *ChatServiceServer) SaveMessage(ctx context.Context, req *dbchat.SaveMessageRequest) (*dbchat.SaveMessageResponse, error) {

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "Не удалось получить метаданные из контекста")
	}

	// Извлекаем DatabaseName из метаданных
	database := md["database"][0] // токен передается как "auth-token"
	if len(database) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "database не найдена в метаданных")
	}

	// Извлекаем UserId из метаданных
	userId := md["user-id"][0] // токен передается как "auth-token"
	if len(userId) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "userId не найдена в метаданных")
	}

	// Получаем строку подключения к базе данных
	dsn := utils.DsnString(database)
	dbConnCompany, err := s.connectionsMap.GetDb(dsn)
	if err != nil || dbConnCompany == nil {
		log.Printf("Ошибка подключения к базе данных: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка подключения к базе данных: %s", err))
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
	err = dbConnCompany.QueryRowContext(ctx, insertMessageQuery, req.ChatId, userId, req.Content).
		Scan(&messageID, &createdAt)
	if err != nil {
		log.Printf("Ошибка при сохранении сообщения: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка при сохранении сообщения: %s", err))
	}

	// Успешный ответ
	return &dbchat.SaveMessageResponse{
		MessageId: messageID,
		ChatId:    req.ChatId,
		Message:   req.Content,
		CreatedAt: createdAt.Unix(),
	}, nil
}

package dbchatservice

import (
	"context"
	"crmSystem/proto/dbchat"
	"crmSystem/proto/logs"
	"crmSystem/utils"
	"database/sql"
	"fmt"
	"google.golang.org/grpc"
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

	token, err := utils.ExtractTokenFromContext(ctx)
	if err != nil {
		log.Printf("Не удалось извлечь токен для логирования: %v", err)
		return nil, status.Errorf(codes.Unauthenticated, "Не удалось извлечь токен для логирования")
	}

	// Устанавливаем соединение с gRPC сервером Logs
	clientLogs, err, conn := utils.GRPCServiceConnector(token, logs.NewLogsServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		return nil, status.Errorf(codes.Unauthenticated, "Не удалось создать соединение с сервером Logs")
	} else {
		defer func(conn *grpc.ClientConn) {
			err := conn.Close()
			if err != nil {
				log.Printf("Ошибка закрытия соединения: %v", err)
				errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
				if errLogs != nil {
					log.Printf("Ошибка закрытия соединения: %v", err)
				}
			}
		}(conn)
	}

	// Извлекаем DatabaseName из метаданных
	database := md["database"][0] // токен передается как "auth-token"
	if len(database) == 0 {
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", "database не найдена в метаданных")
		if errLogs != nil {
			log.Printf("database не найдена в метаданных: %v", err)
		}
		return nil, status.Errorf(codes.Unauthenticated, "database не найдена в метаданных")
	}

	// Извлекаем userId из метаданных
	userId := md["user-id"][0] // токен передается как "auth-token"
	if len(userId) == 0 {
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", "userId не найден в метаданных")
		if errLogs != nil {
			log.Printf("userId не найден в метаданных: %v", err)
		}
		return nil, status.Errorf(codes.Unauthenticated, "userId не найден в метаданных")
	}

	log.Printf("CreateChat: %s", "CreateChat")
	// Получаем строку подключения к базе данных
	dsn := utils.DsnString(database)
	dbConnCompany, err := s.connectionsMap.GetDb(dsn)
	if err != nil || dbConnCompany == nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, fmt.Sprintf("Ошибка подключения к базе данных: %v", database))
		if errLogs != nil {
			log.Printf("Ошибка подключения к базе данных: %v", err)
		}
		log.Printf("Ошибка подключения к базе данных: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка подключения к базе данных: %v", err))
	}

	// Начинаем транзакцию
	tx, err := dbConnCompany.BeginTx(ctx, nil)
	if err != nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
		if errLogs != nil {
			log.Printf("Ошибка начала транзакции: %v", err)
		}
		log.Printf("Ошибка начала транзакции: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка начала транзакции: %v", err))
	}

	// Создаём новый чат
	createChatQuery := `INSERT INTO chats (chat_name) VALUES ($1) RETURNING id;`
	var chatID int64
	err = tx.QueryRowContext(ctx, createChatQuery, req.ChatName).Scan(&chatID)
	if err != nil {
		err := tx.Rollback()
		if err != nil {
			return nil, err
		}
		errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, fmt.Sprintf("Ошибка создания чата"))
		if errLogs != nil {
			log.Printf("Ошибка создания чата: %v", err)
		}
		log.Printf("Ошибка создания чата: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка создания чата: %v", err))
	}

	// Завершаем транзакцию на уровне создания чата
	err = tx.Commit()
	if err != nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка при коммите транзакции создания чата: %v", err)
		}
		log.Printf("Ошибка при коммите транзакции создания чата: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка при коммите создания чата: %v", err))
	}

	// Создаём запрос для добавления пользователей
	addUsersReq := &dbchat.AddUsersToChatRequest{
		ChatId:  chatID,
		UsersId: req.UsersId,
	}

	// Вызываем метод AddUsersToChat
	addUsersResp, err := s.AddUsersToChat(ctx, addUsersReq, clientLogs, userId)
	if err != nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка добавления пользователей в чат: %v", err)
		}
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

func (s *ChatServiceServer) AddUsersToChat(ctx context.Context, req *dbchat.AddUsersToChatRequest, clientLogs logs.LogsServiceClient, userId string) (*dbchat.AddUsersToChatResponse, error) {

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
		errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка подключения к базе данных: %v", err)
		}
		log.Printf("Ошибка подключения к базе данных: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка подключения к базе данных: %v", err))
	}
	defer func(dbConnCompany *sql.DB) {
		err := dbConnCompany.Close()
		if err != nil {
			errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, err.Error())
			if errLogs != nil {
				log.Printf("Ошибка закрытия соединения: %v", err)
			}
		}
	}(dbConnCompany)

	// Начало транзакции
	tx, err := dbConnCompany.Begin()
	if err != nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка начала транзакции: %v", err)
		}
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
		err := tx.Rollback()
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка закрытия канала %v", err.Error()))
		} // Откат транзакции при ошибке
		errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, "Ошибка добавления пользователей в чат")
		if errLogs != nil {
			log.Printf("Откат транзакции при ошибке: %v", err)
		}
		log.Printf("Ошибка добавления пользователей в чат: %v", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка добавления пользователей в чат %v", req.ChatId))
	}

	// Подтверждаем транзакцию
	err = tx.Commit()
	if err != nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка подтверждения транзакции: %v", err)
		}
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

	token, err := utils.ExtractTokenFromContext(ctx)
	if err != nil {
		log.Printf("Не удалось извлечь токен для логирования: %v", err)
		return nil, status.Errorf(codes.Unauthenticated, "Не удалось извлечь токен для логирования")
	}

	// Устанавливаем соединение с gRPC сервером Logs
	clientLogs, err, conn := utils.GRPCServiceConnector(token, logs.NewLogsServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		return nil, status.Errorf(codes.Unauthenticated, "Не удалось создать соединение с сервером Logs")
	} else {
		defer func(conn *grpc.ClientConn) {
			err := conn.Close()
			if err != nil {
				log.Printf("Ошибка закрытия соединения: %v", err)
				errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
				if errLogs != nil {
					log.Printf("Ошибка закрытия соединения: %v", err)
				}
			}
		}(conn)
	}

	// Извлекаем DatabaseName из метаданных
	database := md["database"][0] // токен передается как "auth-token"
	if len(database) == 0 {
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", "database не найдена в метаданных")
		if errLogs != nil {
			log.Printf("database не найдена в метаданных: %v", err)
		}
		return nil, status.Errorf(codes.Unauthenticated, "database не найдена в метаданных")
	}

	// Извлекаем UserId из метаданных
	userId := md["user-id"][0] // токен передается как "auth-token"
	if len(userId) == 0 {
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", "userId не найдена в метаданных")
		if errLogs != nil {
			log.Printf("userId не найдена в метаданных: %v", err)
		}
		return nil, status.Errorf(codes.Unauthenticated, "userId не найдена в метаданных")
	}

	// Получаем строку подключения к базе данных
	dsn := utils.DsnString(database)
	dbConnCompany, err := s.connectionsMap.GetDb(dsn)
	if err != nil || dbConnCompany == nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, "Ошибка подключения к базе данных")
		if errLogs != nil {
			log.Printf("Ошибка подключения к базе данных: %v", err)
		}
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
		errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка при сохранении сообщения: %v", err)
		}
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

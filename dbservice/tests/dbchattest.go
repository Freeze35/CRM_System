package tests

import (
	"context"
	"crmSystem/dbchatservice"
	"crmSystem/proto/dbchat"
	"crmSystem/utils"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"testing"
	"time"
)

// TestCreateChat тестирует метод CreateChat структуры ChatServiceServer.
func TestCreateChat(t *testing.T) {
	// Инициализируем MapConnectionsDB
	serverPool := utils.NewMapConnectionsDB()

	// Создаём мок базы данных для компании
	companyDB, companyMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Не удалось создать мок базы данных компании: %v", err)
	}
	defer companyDB.Close()

	// Добавляем мок базы данных в пул соединений
	serverPool.MapDB["test_company_db"] = companyDB

	// Создаём экземпляр ChatServiceServer
	chatService := dbchatservice.NewGRPCDBChatService(serverPool)

	// Тестовые случаи
	tests := []struct {
		name           string
		req            *dbchat.CreateChatRequest
		ctx            context.Context
		prepareMocks   func(companyMock sqlmock.Sqlmock)
		expectedResp   *dbchat.CreateChatResponse
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name: "Успешное создание чата",
			req: &dbchat.CreateChatRequest{
				ChatName: "Test Chat",
				UsersId: []*dbchat.UserId{
					{UserId: 1, RoleId: 1},
					{UserId: 2, RoleId: 1},
				},
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "admin1",
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(companyMock sqlmock.Sqlmock) {
				// Мок для создания чата
				companyMock.ExpectBegin()
				companyMock.ExpectQuery(`INSERT INTO chats \(chat_name\) VALUES \(\$1\) RETURNING id`).
					WithArgs("Test Chat").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				companyMock.ExpectCommit()

				// Мок для добавления пользователей в чат
				companyMock.ExpectBegin()
				companyMock.ExpectExec(`INSERT INTO chat_users \(user_id, chat_id\) VALUES \(\$1, \$2\),\(\$3, \$4\)`).
					WithArgs(int64(1), int64(1), int64(2), int64(1)).
					WillReturnResult(sqlmock.NewResult(0, 2))
				companyMock.ExpectCommit()
			},
			expectedResp: &dbchat.CreateChatResponse{
				ChatId:    1,
				Message:   "Чат 'Test Chat' успешно создан. Пользователи успешно добавлены в чат 1",
				CreatedAt: time.Now().Unix(),
			},
			expectedErr: false,
		},
		{
			name: "Отсутствует метаданные database",
			req: &dbchat.CreateChatRequest{
				ChatName: "Test Chat",
				UsersId:  []*dbchat.UserId{{UserId: 1, RoleId: 1}},
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"user-id", "admin1",
				"authorization", "Bearer test_token",
			)),
			prepareMocks:   func(companyMock sqlmock.Sqlmock) {},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "database не найдена в метаданных",
		},
		{
			name: "Отсутствует метаданные user-id",
			req: &dbchat.CreateChatRequest{
				ChatName: "Test Chat",
				UsersId:  []*dbchat.UserId{{UserId: 1, RoleId: 1}},
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"authorization", "Bearer test_token",
			)),
			prepareMocks:   func(companyMock sqlmock.Sqlmock) {},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "userId не найдена в метаданных",
		},
		{
			name: "Отсутствует токен авторизации",
			req: &dbchat.CreateChatRequest{
				ChatName: "Test Chat",
				UsersId:  []*dbchat.UserId{{UserId: 1, RoleId: 1}},
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "admin1",
			)),
			prepareMocks:   func(companyMock sqlmock.Sqlmock) {},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "токен отсутствует",
		},
		{
			name: "Ошибка создания чата",
			req: &dbchat.CreateChatRequest{
				ChatName: "Test Chat",
				UsersId:  []*dbchat.UserId{{UserId: 1, RoleId: 1}},
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "admin1",
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(companyMock sqlmock.Sqlmock) {
				companyMock.ExpectBegin()
				companyMock.ExpectQuery(`INSERT INTO chats \(chat_name\) VALUES \(\$1\) RETURNING id`).
					WithArgs("Test Chat").
					WillReturnError(fmt.Errorf("database error"))
				companyMock.ExpectRollback()
			},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "Ошибка создания чата: database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Подготавливаем моки для тестового случая
			tt.prepareMocks(companyMock)

			// Вызываем метод
			resp, err := chatService.CreateChat(tt.ctx, tt.req)

			// Проверяем ожидаемую ошибку
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				// Учитываем возможное небольшое расхождение во времени
				if resp != nil {
					resp.CreatedAt = tt.expectedResp.CreatedAt
				}
				assert.Equal(t, tt.expectedResp, resp)
			}

			// Проверяем, что все ожидания мока выполнены
			assert.NoError(t, companyMock.ExpectationsWereMet())
		})
	}
}

// TestAddUsersToChat тестирует метод AddUsersToChat структуры ChatServiceServer.
func TestAddUsersToChat(t *testing.T) {
	// Инициализируем MapConnectionsDB
	serverPool := utils.NewMapConnectionsDB()

	// Создаём мок базы данных для компании
	companyDB, companyMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Не удалось создать мок базы данных компании: %v", err)
	}
	defer companyDB.Close()

	// Добавляем мок базы данных в пул соединений
	serverPool.MapDB["test_company_db"] = companyDB

	// Создаём экземпляр ChatServiceServer
	chatService := dbchatservice.NewGRPCDBChatService(serverPool)

	// Тестовые случаи
	tests := []struct {
		name           string
		req            *dbchat.AddUsersToChatRequest
		ctx            context.Context
		prepareMocks   func(companyMock sqlmock.Sqlmock)
		expectedResp   *dbchat.AddUsersToChatResponse
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name: "Успешное добавление пользователей",
			req: &dbchat.AddUsersToChatRequest{
				ChatId: 1,
				UsersId: []*dbchat.UserId{
					{UserId: 1, RoleId: 1},
					{UserId: 2, RoleId: 1},
				},
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "admin1",
			)),
			prepareMocks: func(companyMock sqlmock.Sqlmock) {
				companyMock.ExpectBegin()
				companyMock.ExpectExec(`INSERT INTO chat_users \(user_id, chat_id\) VALUES \(\$1, \$2\),\(\$3, \$4\)`).
					WithArgs(int64(1), int64(1), int64(2), int64(1)).
					WillReturnResult(sqlmock.NewResult(0, 2))
				companyMock.ExpectCommit()
			},
			expectedResp: &dbchat.AddUsersToChatResponse{
				Message: "Пользователи успешно добавлены в чат 1",
			},
			expectedErr: false,
		},
		{
			name: "Отсутствует метаданные database",
			req: &dbchat.AddUsersToChatRequest{
				ChatId:  1,
				UsersId: []*dbchat.UserId{{UserId: 1, RoleId: 1}},
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"user-id", "admin1",
			)),
			prepareMocks:   func(companyMock sqlmock.Sqlmock) {},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "database не найдена в метаданных",
		},
		{
			name: "Ошибка добавления пользователей",
			req: &dbchat.AddUsersToChatRequest{
				ChatId:  1,
				UsersId: []*dbchat.UserId{{UserId: 1, RoleId: 1}},
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "admin1",
			)),
			prepareMocks: func(companyMock sqlmock.Sqlmock) {
				companyMock.ExpectBegin()
				companyMock.ExpectExec(`INSERT INTO chat_users \(user_id, chat_id\) VALUES \(\$1, \$2\)`).
					WithArgs(int64(1), int64(1)).
					WillReturnError(fmt.Errorf("database error"))
				companyMock.ExpectRollback()
			},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "Ошибка добавления пользователей в чат 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Подготавливаем моки для тестового случая
			tt.prepareMocks(companyMock)

			// Вызываем метод
			resp, err := chatService.AddUsersToChat(tt.ctx, tt.req, nil, "admin1")

			// Проверяем ожидаемую ошибку
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}

			// Проверяем, что все ожидания мока выполнены
			assert.NoError(t, companyMock.ExpectationsWereMet())
		})
	}
}

// TestSaveMessage тестирует метод SaveMessage структуры ChatServiceServer.
func TestSaveMessage(t *testing.T) {
	// Инициализируем MapConnectionsDB
	serverPool := utils.NewMapConnectionsDB()

	// Создаём мок базы данных для компании
	companyDB, companyMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Не удалось создать мок базы данных компании: %v", err)
	}
	defer companyDB.Close()

	// Добавляем мок базы данных в пул соединений
	serverPool.MapDB["test_company_db"] = companyDB

	// Создаём экземпляр ChatServiceServer
	chatService := dbchatservice.NewGRPCDBChatService(serverPool)

	// Тестовые случаи
	tests := []struct {
		name           string
		req            *dbchat.SaveMessageRequest
		ctx            context.Context
		prepareMocks   func(companyMock sqlmock.Sqlmock)
		expectedResp   *dbchat.SaveMessageResponse
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name: "Успешное сохранение сообщения",
			req: &dbchat.SaveMessageRequest{
				ChatId:  1,
				Content: "Hello, world!",
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "user1",
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(companyMock sqlmock.Sqlmock) {
				companyMock.ExpectQuery(`INSERT INTO messages \(chat_id, user_id, message\) VALUES \(\$1, \$2, \$3\) RETURNING id, created_at`).
					WithArgs(int64(1), "user1", "Hello, world!").
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(1, time.Now()))
			},
			expectedResp: &dbchat.SaveMessageResponse{
				MessageId: 1,
				ChatId:    1,
				Message:   "Hello, world!",
				CreatedAt: time.Now().Unix(),
			},
			expectedErr: false,
		},
		{
			name: "Отсутствует метаданные database",
			req: &dbchat.SaveMessageRequest{
				ChatId:  1,
				Content: "Hello, world!",
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"user-id", "user1",
				"authorization", "Bearer test_token",
			)),
			prepareMocks:   func(companyMock sqlmock.Sqlmock) {},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "database не найдена в метаданных",
		},
		{
			name: "Отсутствует метаданные user-id",
			req: &dbchat.SaveMessageRequest{
				ChatId:  1,
				Content: "Hello, world!",
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"authorization", "Bearer test_token",
			)),
			prepareMocks:   func(companyMock sqlmock.Sqlmock) {},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "userId не найдена в метаданных",
		},
		{
			name: "Отсутствует токен авторизации",
			req: &dbchat.SaveMessageRequest{
				ChatId:  1,
				Content: "Hello, world!",
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "user1",
			)),
			prepareMocks:   func(companyMock sqlmock.Sqlmock) {},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "токен отсутствует",
		},
		{
			name: "Ошибка сохранения сообщения",
			req: &dbchat.SaveMessageRequest{
				ChatId:  1,
				Content: "Hello, world!",
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "user1",
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(CompanyMock sqlmock.Sqlmock) {
				companyMock.ExpectQuery(`INSERT INTO messages \(chat_id, user_id, message\) VALUES \(\$1, \$2, \$3\) RETURNING id, created_at`).
					WithArgs(int64(1), "user1", "Hello, world!").
					WillReturnError(fmt.Errorf("database error"))
			},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "Ошибка при сохранении сообщения: database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Подготавливаем моки для тестового случая
			tt.prepareMocks(companyMock)

			// Вызываем метод
			resp, err := chatService.SaveMessage(tt.ctx, tt.req)

			// Проверяем ожидаемую ошибку
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				// Учитываем возможное небольшое расхождение во времени
				if resp != nil {
					resp.CreatedAt = tt.expectedResp.CreatedAt
				}
				assert.Equal(t, tt.expectedResp, resp)
			}

			// Проверяем, что все ожидания мока выполнены
			assert.NoError(t, companyMock.ExpectationsWereMet())
		})
	}
}

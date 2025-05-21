package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"crmSystem/proto/dbadmin"
	"crmSystem/proto/email-service"
	"crmSystem/proto/logs"
	"crmSystem/tests/mocks"
	"crmSystem/transport_rest"
	"crmSystem/transport_rest/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

// testHandler — обертка для transport_rest.Handler с замоканными зависимостями
type testHandler struct {
	*transport_rest.Handler
	grpcServiceConnector func(token string, client interface{}) (interface{}, error, *grpc.ClientConn)
	saveLogsError        func(ctx context.Context, client logs.LogsServiceClient, database, userId, errorMessage string) error
}

func TestAddUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDbAdmin := mocks.NewMockDbAdminServiceClient(ctrl)
	mockLogs := mocks.NewMockLogsServiceClient(ctrl)
	mockEmail := mocks.NewMockEmailServiceClient(ctrl)

	h := &testHandler{
		Handler: &transport_rest.Handler{},
		grpcServiceConnector: func(token string, client interface{}) (interface{}, error, *grpc.ClientConn) {
			conn, _ := grpc.Dial("fake", grpc.WithInsecure())
			switch client.(type) {
			case func(grpc.ClientConnInterface) dbadmin.DbAdminServiceClient:
				return mockDbAdmin, nil, conn
			case func(grpc.ClientConnInterface) logs.LogsServiceClient:
				return mockLogs, nil, conn
			case func(grpc.ClientConnInterface) email.EmailServiceClient:
				return mockEmail, nil, conn
			}
			return nil, fmt.Errorf("unknown client"), nil
		},
		saveLogsError: func(ctx context.Context, client logs.LogsServiceClient, database, userId, errorMessage string) error {
			if client == nil {
				return fmt.Errorf("logs client is nil")
			}
			return nil // Мок успешного сохранения логов
		},
	}

	tests := []struct {
		name           string
		cookies        []*http.Cookie
		body           interface{}
		mockSetup      func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			cookies: []*http.Cookie{
				{Name: "access_token", Value: "valid-token"},
				{Name: "user-id", Value: "test-user"},
				{Name: "database", Value: "test-db"},
				{Name: "DbName", Value: "test-database"}, // Изменено на test-database
			},
			body: types.RegisterUsersRequest{
				CompanyId: "company-123",
				Users: []*types.User{
					{Email: "user1@example.com", Phone: "1234567890", RoleId: int64(1)},
				},
			},
			mockSetup: func() {
				mockDbAdmin.EXPECT().RegisterUsersInCompany(gomock.Any(), &dbadmin.RegisterUsersRequest{
					CompanyId: "company-123",
					Users: []*dbadmin.User{
						{Email: "user1@example.com", Phone: "1234567890", RoleId: int64(1)},
					},
				}).Return(&dbadmin.RegisterUsersResponse{
					Message: "Users registered",
					Users: []*dbadmin.UserResponse{
						{Email: "user1@example.com", Phone: "1234567890", RoleId: int64(1), Password: "pass123"},
					},
				}, nil)

				mockEmail.EXPECT().SendEmail(gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, req *email.SendEmailRequest) (*email.SendEmailResponse, error) {
						assert.Equal(t, "user1@example.com", req.Email)
						assert.Equal(t, "Welcome to our service! FROM PETR", req.Message)
						assert.Contains(t, req.Body, "user1@example.com")
						assert.Contains(t, req.Body, "pass123")
						return &email.SendEmailResponse{Message: "Email sent"}, nil
					},
				)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Successfully sent to all 1 users.","failures":""}`,
		},
		{
			name: "Missing token",
			cookies: []*http.Cookie{
				{Name: "user-id", Value: "test-user"},
				{Name: "database", Value: "test-db"},
				{Name: "DbName", Value: "test-database"},
			},
			body:           types.RegisterUsersRequest{},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"access_token не найден http: named cookie not present"}`,
		},
		{
			name: "Invalid JSON",
			cookies: []*http.Cookie{
				{Name: "access_token", Value: "valid-token"},
				{Name: "user-id", Value: "test-user"},
				{Name: "database", Value: "test-db"},
				{Name: "DbName", Value: "test-database"},
			},
			body: "invalid json",
			mockSetup: func() {
				mockLogs.EXPECT().SaveLogs(gomock.Any(), gomock.Any()).Return(&logs.LogResponse{}, nil).AnyTimes()
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Ошибка при декодировании данных"}`,
		},
		{
			name: "Validation Error",
			cookies: []*http.Cookie{
				{Name: "access_token", Value: "valid-token"},
				{Name: "user-id", Value: "test-user"},
				{Name: "database", Value: "test-db"},
				{Name: "DbName", Value: "test-database"},
			},
			body: types.RegisterUsersRequest{
				CompanyId: "",
				Users: []*types.User{
					{Email: "invalid", Phone: "", RoleId: int64(0)},
				},
			},
			mockSetup: func() {
				mockLogs.EXPECT().SaveLogs(gomock.Any(), gomock.Any()).Return(&logs.LogResponse{}, nil).AnyTimes()
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Ошибка валидации Поле 'CompanyId' не прошло валидацию"}`,
		},
		{
			name: "gRPC RegisterUsers Failure",
			cookies: []*http.Cookie{
				{Name: "access_token", Value: "valid-token"},
				{Name: "user-id", Value: "test-user"},
				{Name: "database", Value: "test-db"},
				{Name: "DbName", Value: "test-database"},
			},
			body: types.RegisterUsersRequest{
				CompanyId: "company-123",
				Users: []*types.User{
					{Email: "user1@example.com", Phone: "1234567890", RoleId: int64(1)},
				},
			},
			mockSetup: func() {
				mockDbAdmin.EXPECT().RegisterUsersInCompany(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("database error"))
				mockLogs.EXPECT().SaveLogs(gomock.Any(), gomock.Any()).Return(&logs.LogResponse{}, nil).AnyTimes()
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"Ошибка валидации Поле 'DbName' не прошло валидацию"}`,
		},
		{
			name: "Partial Email Failure",
			cookies: []*http.Cookie{
				{Name: "access_token", Value: "valid-token"},
				{Name: "user-id", Value: "test-user"},
				{Name: "database", Value: "test-db"},
				{Name: "DbName", Value: "test-database"},
			},
			body: types.RegisterUsersRequest{
				CompanyId: "company-123",
				Users: []*types.User{
					{Email: "user1@example.com", Phone: "1234567890", RoleId: int64(1)},
					{Email: "user2@example.com", Phone: "0987654321", RoleId: int64(2)},
				},
			},
			mockSetup: func() {
				mockDbAdmin.EXPECT().RegisterUsersInCompany(gomock.Any(), &dbadmin.RegisterUsersRequest{
					CompanyId: "company-123",
					Users: []*dbadmin.User{
						{Email: "user1@example.com", Phone: "1234567890", RoleId: int64(1)},
						{Email: "user2@example.com", Phone: "0987654321", RoleId: int64(2)},
					},
				}).Return(&dbadmin.RegisterUsersResponse{
					Message: "Users registered",
					Users: []*dbadmin.UserResponse{
						{Email: "user1@example.com", Phone: "1234567890", RoleId: int64(1), Password: "pass123"},
						{Email: "user2@example.com", Phone: "0987654321", RoleId: int64(2), Password: "pass456"},
					},
				}, nil)

				mockEmail.EXPECT().SendEmail(gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, req *email.SendEmailRequest) (*email.SendEmailResponse, error) {
						assert.Equal(t, "user1@example.com", req.Email)
						assert.Equal(t, "Welcome to our service! FROM PETR", req.Message)
						assert.Contains(t, req.Body, "user1@example.com")
						assert.Contains(t, req.Body, "pass123")
						return &email.SendEmailResponse{Message: "Email sent"}, nil
					},
				)

				mockEmail.EXPECT().SendEmail(gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, req *email.SendEmailRequest) (*email.SendEmailResponse, error) {
						assert.Equal(t, "user2@example.com", req.Email)
						assert.Equal(t, "Welcome to our service! FROM PETR", req.Message)
						assert.Contains(t, req.Body, "user2@example.com")
						assert.Contains(t, req.Body, "pass456")
						return nil, fmt.Errorf("email service error")
					},
				)

				mockLogs.EXPECT().SaveLogs(gomock.Any(), gomock.Any()).Return(&logs.LogResponse{}, nil).AnyTimes()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Ошибка валидации Поле 'DbName' не прошло валидацию"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/admin/addusers", bytes.NewReader(bodyBytes))
			for _, cookie := range tt.cookies {
				req.AddCookie(cookie)
			}
			w := httptest.NewRecorder()

			t.Logf("Cookies sent: %+v", req.Cookies()) // Отладка кук
			tt.mockSetup()

			h.AddUsers(w, req)

			t.Logf("Response body: %s", w.Body.String()) // Отладка ответа
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestTransformUsersConcurrently(t *testing.T) {
	users := []*types.User{
		{Email: "user1@example.com", Phone: "1234567890", RoleId: int64(1)},
		{Email: "user2@example.com", Phone: "0987654321", RoleId: int64(2)},
	}

	result := transport_rest.TransformUsersConcurrently(users)

	assert.Len(t, result, 2)
	for _, expected := range []*dbadmin.User{
		{Email: "user1@example.com", Phone: "1234567890", RoleId: int64(1)},
		{Email: "user2@example.com", Phone: "0987654321", RoleId: int64(2)},
	} {
		assert.Contains(t, result, expected)
	}
}

func TestCallAddUsers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDbAdmin := mocks.NewMockDbAdminServiceClient(ctrl)
	mockLogs := mocks.NewMockLogsServiceClient(ctrl)

	req := &types.RegisterUsersRequest{
		CompanyId: "company-123",
		Users: []*types.User{
			{Email: "user1@example.com", Phone: "1234567890", RoleId: int64(1)},
		},
	}

	ctx := context.Background()

	mockDbAdmin.EXPECT().RegisterUsersInCompany(gomock.Any(), &dbadmin.RegisterUsersRequest{
		CompanyId: "company-123",
		Users: []*dbadmin.User{
			{Email: "user1@example.com", Phone: "1234567890", RoleId: int64(1)},
		},
	}).Return(&dbadmin.RegisterUsersResponse{
		Message: "Users registered",
		Users: []*dbadmin.UserResponse{
			{Email: "user1@example.com", Phone: "1234567890", RoleId: int64(1)},
		},
	}, nil)

	response, err := transport_rest.CallAddUsers(ctx, mockDbAdmin, req, mockLogs, "test-db", "test-user")
	assert.NoError(t, err)
	assert.Equal(t, "Users registered", response.Message)
}

func TestSendToEmailUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEmail := mocks.NewMockEmailServiceClient(ctrl)

	req := &types.SendEmailRequest{
		Email:   "user@example.com",
		Message: "Welcome",
		Body:    "Hello",
	}

	mockEmail.EXPECT().SendEmail(gomock.Any(), &email.SendEmailRequest{
		Email:   "user@example.com",
		Message: "Welcome",
		Body:    "Hello",
	}).Return(&email.SendEmailResponse{Message: "Email sent"}, nil)

	response, err := transport_rest.SendToEmailUser(mockEmail, req)
	assert.NoError(t, err)
	assert.Equal(t, "Email sent", response.Message)
}

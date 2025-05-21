package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"crmSystem/proto/dbadmin"
	"crmSystem/proto/email-service"
	"crmSystem/proto/logs"
	"crmSystem/tests/mocks"
	"crmSystem/transport_rest"
	"crmSystem/transport_rest/types"
	"crmSystem/utils"

	"github.com/go-playground/validator/v10"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// testHandler — обертка для transport_rest.Handler с замоканными зависимостями
type testHandler struct {
	*transport_rest.Handler
	grpcServiceConnector func(token string, client interface{}) (interface{}, error, *grpc.ClientConn)
	saveLogsError        func(ctx context.Context, client logs.LogsServiceClient, database, userId, errorMessage string) error
}

// AddUsers — переопределенный метод для использования замоканных зависимостей
func (h *testHandler) AddUsers(w http.ResponseWriter, r *http.Request) {
	token := utils.GetFromCookies(w, r, "access_token")
	if token == "" {
		utils.CreateError(w, http.StatusBadRequest, "Токен не найден", fmt.Errorf(""))
		return
	}

	userId := utils.GetFromCookies(w, r, "user-id")
	if userId == "" {
		utils.CreateError(w, http.StatusBadRequest, "user-id не найден", fmt.Errorf(""))
		return
	}

	database := utils.GetFromCookies(w, r, "database")
	if database == "" {
		utils.CreateError(w, http.StatusBadRequest, "database не найдена", fmt.Errorf(""))
		return
	}

	md := metadata.Pairs("user-id", userId, "database", database)
	ctxWithMetadata := metadata.NewOutgoingContext(context.Background(), md)

	clientLogs, err, conn := h.grpcServiceConnector(token, logs.NewLogsServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusBadRequest, "Ошибка подключения", err)
		if clientLogs != nil {
			errLogs := h.saveLogsError(ctxWithMetadata, clientLogs.(logs.LogsServiceClient), database, userId, err.Error())
			if errLogs != nil {
				log.Printf("Ошибка логирования: %v", errLogs)
			}
		}
		return
	}
	defer conn.Close()

	var reqUsers types.RegisterUsersRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&reqUsers); err != nil {
		utils.CreateError(w, http.StatusBadRequest, "Ошибка при декодировании данных", err)
		errLogs := h.saveLogsError(ctxWithMetadata, clientLogs.(logs.LogsServiceClient), database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка логирования: %v", errLogs)
		}
		return
	}

	validate := validator.New()
	if err := validate.Struct(reqUsers); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		for _, e := range validationErrors {
			errorMessage := fmt.Sprintf("Поле '%s' не прошло валидацию", e.Field())
			utils.CreateError(w, http.StatusBadRequest, "Ошибка валидации", fmt.Errorf(errorMessage))
			errLogs := h.saveLogsError(ctxWithMetadata, clientLogs.(logs.LogsServiceClient), database, userId, err.Error())
			if errLogs != nil {
				log.Printf("Ошибка логирования: %v", errLogs)
			}
			return
		}
	}

	client, err, conn := h.grpcServiceConnector(token, dbadmin.NewDbAdminServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusBadRequest, "Ошибка подключения", err)
		errLogs := h.saveLogsError(ctxWithMetadata, clientLogs.(logs.LogsServiceClient), database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка логирования: %v", errLogs)
		}
		return
	}
	defer conn.Close()

	response, err := transport_rest.CallAddUsers(ctxWithMetadata, client.(dbadmin.DbAdminServiceClient), &reqUsers, clientLogs.(logs.LogsServiceClient), database, userId)
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не корректная ошибка на сервере.", err)
		errLogs := h.saveLogsError(ctxWithMetadata, clientLogs.(logs.LogsServiceClient), database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка логирования: %v", errLogs)
		}
		return
	}

	clientEmail, err, conn := h.grpcServiceConnector(token, email.NewEmailServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusBadRequest, "Ошибка подключения", err)
		errLogs := h.saveLogsError(ctxWithMetadata, clientLogs.(logs.LogsServiceClient), database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка логирования: %v", errLogs)
		}
		return
	}
	defer conn.Close()

	var successCount, failureCount int
	var failureMessages []string

	for _, user := range response.Users {
		mailRequest := types.SendEmailRequest{
			Email:   user.Email,
			Message: "Welcome to our service! FROM PETR",
			Body: fmt.Sprintf(
				`Hello %s,

				Thank you for signing up for our service! We are excited to have you on board.
				
				Here are your login details:
				- **Login**: %s
				- **Password**: %s
				
				If you have any questions, feel free to contact our support team.
				
				Best regards,
				The Team at Our Service`,
				user.Email, user.Email, user.Password),
		}

		_, err := transport_rest.SendToEmailUser(clientEmail.(email.EmailServiceClient), &mailRequest)
		if err != nil {
			failureCount++
			failureMessages = append(failureMessages, "Failed to send email to "+user.Email+": "+err.Error())
			errLogs := h.saveLogsError(ctxWithMetadata, clientLogs.(logs.LogsServiceClient), database, userId, err.Error())
			if errLogs != nil {
				log.Printf("Ошибка логирования: %v", errLogs)
			}
			continue
		}
		successCount++
	}

	var failuresString string
	if len(failureMessages) > 0 {
		failuresString = strings.Join(failureMessages, "\n")
	}

	responseMessage := fmt.Sprintf("Successfully sent to %d users, failed for %d users.", successCount, failureCount)
	if failureCount == 0 {
		responseMessage = fmt.Sprintf("Successfully sent to all %d users.", successCount)
	}

	sendMessageResponse := &types.SendEmailResponse{
		Message:  responseMessage,
		Failures: failuresString,
	}

	if err := utils.WriteJSON(w, http.StatusOK, sendMessageResponse); err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не корректная ошибка на сервере.", err)
		errLogs := h.saveLogsError(ctxWithMetadata, clientLogs.(logs.LogsServiceClient), database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка логирования: %v", errLogs)
		}
	}
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
			},
			body:           types.RegisterUsersRequest{},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Токен не найден"}`,
		},
		{
			name: "Invalid JSON",
			cookies: []*http.Cookie{
				{Name: "access_token", Value: "valid-token"},
				{Name: "user-id", Value: "test-user"},
				{Name: "database", Value: "test-db"},
			},
			body:           "invalid json",
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Ошибка при декодировании данных"}`,
		},
		{
			name: "Validation Error",
			cookies: []*http.Cookie{
				{Name: "access_token", Value: "valid-token"},
				{Name: "user-id", Value: "test-user"},
				{Name: "database", Value: "test-db"},
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
			expectedBody:   `{"error":"Поле 'CompanyId' не прошло валидацию"}`,
		},
		{
			name: "gRPC RegisterUsers Failure",
			cookies: []*http.Cookie{
				{Name: "access_token", Value: "valid-token"},
				{Name: "user-id", Value: "test-user"},
				{Name: "database", Value: "test-db"},
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
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"Не корректная ошибка на сервере."}`,
		},
		{
			name: "Partial Email Failure",
			cookies: []*http.Cookie{
				{Name: "access_token", Value: "valid-token"},
				{Name: "user-id", Value: "test-user"},
				{Name: "database", Value: "test-db"},
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
			expectedBody:   `{"message":"Successfully sent to 1 users, failed for 1 users.","failures":"Failed to send email to user2@example.com: email service error"}`,
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

			tt.mockSetup()

			h.AddUsers(w, req)

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

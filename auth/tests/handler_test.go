package tests

import (
	"bytes"
	"context"
	"crmSystem/proto/dbauth"
	"crmSystem/proto/logs"
	"crmSystem/tests/mocks"
	"crmSystem/transport_rest"
	"crmSystem/transport_rest/types"
	"encoding/json"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"net/http"
	"net/http/httptest"
	"testing"
)

// testHandler wraps transport_rest.Handler
type testHandler struct {
	*transport_rest.Handler
}

func TestLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDbAuth := mocks.NewMockDbAuthServiceClient(ctrl)
	mockLogs := mocks.NewMockLogsServiceClient(ctrl)

	h := &testHandler{
		Handler: transport_rest.NewHandler(),
	}

	tests := []struct {
		name           string
		body           interface{}
		cookies        []*http.Cookie
		mockSetup      func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			body: types.LoginAuthRequest{
				Email:    "user@example.com",
				Phone:    "+1234567890",
				Password: "ValidPass123",
			},
			cookies: nil,
			mockSetup: func() {
				mockLogs.EXPECT().SaveLogs(gomock.Any(), gomock.Any()).Return(&logs.LogResponse{}, nil).AnyTimes()
				mockDbAuth.EXPECT().LoginDB(gomock.Any(), &dbauth.LoginDBRequest{
					Email:    "user@example.com",
					Phone:    "+1234567890",
					Password: "ValidPass123",
				}, gomock.Any()).DoAndReturn(
					func(ctx context.Context, req *dbauth.LoginDBRequest, opts ...grpc.CallOption) (*dbauth.LoginDBResponse, error) {
						md := metadata.New(map[string]string{
							"database":   "test-db",
							"user-id":    "user-123",
							"company-id": "company-123",
						})
						for k, v := range md {
							grpc.SetHeader(ctx, metadata.Pairs(k, v[0]))
						}
						return &dbauth.LoginDBResponse{Message: "Login successful"}, nil
					},
				)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Login successful"}`,
		},
		{
			name:    "Invalid JSON",
			body:    "invalid json",
			cookies: nil,
			mockSetup: func() {
				mockLogs.EXPECT().SaveLogs(gomock.Any(), gomock.Any()).Return(&logs.LogResponse{}, nil).AnyTimes()
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Ошибка при декодировании данных"}`,
		},
		{
			name: "Validation Error",
			body: types.LoginAuthRequest{
				Email:    "invalid-email",
				Phone:    "123",
				Password: "short",
			},
			cookies: nil,
			mockSetup: func() {
				mockLogs.EXPECT().SaveLogs(gomock.Any(), gomock.Any()).Return(&logs.LogResponse{}, nil).AnyTimes()
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Ошибка валидации","message":"Поле 'Email' не прошло валидацию"}`,
		},
		{
			name: "gRPC Unauthenticated",
			body: types.LoginAuthRequest{
				Email:    "user@example.com",
				Phone:    "+1234567890",
				Password: "ValidPass123",
			},
			cookies: nil,
			mockSetup: func() {
				mockLogs.EXPECT().SaveLogs(gomock.Any(), gomock.Any()).Return(&logs.LogResponse{}, nil).AnyTimes()
				mockDbAuth.EXPECT().LoginDB(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, status.Error(codes.Unauthenticated, "invalid credentials"))
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"Ошибка на сервере","message":"неавторизированный запрос : invalid credentials"}`,
		},
		{
			name: "Missing Metadata",
			body: types.LoginAuthRequest{
				Email:    "user@example.com",
				Phone:    "+1234567890",
				Password: "ValidPass123",
			},
			cookies: nil,
			mockSetup: func() {
				mockLogs.EXPECT().SaveLogs(gomock.Any(), gomock.Any()).Return(&logs.LogResponse{}, nil).AnyTimes()
				mockDbAuth.EXPECT().LoginDB(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, req *dbauth.LoginDBRequest, opts ...grpc.CallOption) (*dbauth.LoginDBResponse, error) {
						grpc.SetHeader(ctx, metadata.Pairs("database", "test-db"))
						return &dbauth.LoginDBResponse{Message: "Login successful"}, nil
					},
				)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"Ошибка на сервере","message":"отсутствуют необходимые метаданные"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, err := json.Marshal(tt.body)
			if s, ok := tt.body.(string); ok {
				bodyBytes = []byte(s)
			} else if err != nil {
				t.Fatalf("Failed to marshal body: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(bodyBytes))
			for _, cookie := range tt.cookies {
				req.AddCookie(cookie)
			}
			w := httptest.NewRecorder()

			tt.mockSetup()

			h.Handler.Login(w, req)

			t.Logf("Response body: %s", w.Body.String())
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDbAuth := mocks.NewMockDbAuthServiceClient(ctrl)
	mockLogs := mocks.NewMockLogsServiceClient(ctrl)

	h := &testHandler{
		Handler: transport_rest.NewHandler(),
	}

	tests := []struct {
		name           string
		body           interface{}
		cookies        []*http.Cookie
		mockSetup      func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			body: types.RegisterAuthRequest{
				Email:       "company@example.com",
				Phone:       "+1234567890",
				Password:    "ValidPass123",
				NameCompany: "Test Company",
				Address:     "123 Test St",
				CompanyDb:   "test_company_db",
			},
			cookies: nil,
			mockSetup: func() {
				mockLogs.EXPECT().SaveLogs(gomock.Any(), gomock.Any()).Return(&logs.LogResponse{}, nil).AnyTimes()
				mockDbAuth.EXPECT().RegisterCompany(gomock.Any(), &dbauth.RegisterCompanyRequest{
					NameCompany: "Test Company",
					Address:     "123 Test St",
					Email:       "company@example.com",
					Phone:       "+1234567890",
					Password:    "ValidPass123",
				}, gomock.Any()).DoAndReturn(
					func(ctx context.Context, req *dbauth.RegisterCompanyRequest, opts ...grpc.CallOption) (*dbauth.RegisterCompanyResponse, error) {
						md := metadata.New(map[string]string{
							"database":   "test-db",
							"user-id":    "user-123",
							"company-id": "company-123",
						})
						for k, v := range md {
							grpc.SetHeader(ctx, metadata.Pairs(k, v[0]))
						}
						return &dbauth.RegisterCompanyResponse{Message: "Registration successful"}, nil
					},
				)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Registration successful"}`,
		},
		{
			name:    "Invalid JSON",
			body:    "invalid json",
			cookies: nil,
			mockSetup: func() {
				mockLogs.EXPECT().SaveLogs(gomock.Any(), gomock.Any()).Return(&logs.LogResponse{}, nil).AnyTimes()
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Ошибка при декодировании данных"}`,
		},
		{
			name: "Validation Error",
			body: types.RegisterAuthRequest{
				Email:       "invalid-email",
				Phone:       "123",
				Password:    "short",
				NameCompany: "",
				Address:     "",
				CompanyDb:   "",
			},
			cookies: nil,
			mockSetup: func() {
				mockLogs.EXPECT().SaveLogs(gomock.Any(), gomock.Any()).Return(&logs.LogResponse{}, nil).AnyTimes()
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Ошибка валидации","message":"Поле 'NameCompany' не прошло валидацию"}`,
		},
		{
			name: "gRPC AlreadyExists",
			body: types.RegisterAuthRequest{
				Email:       "company@example.com",
				Phone:       "+1234567890",
				Password:    "ValidPass123",
				NameCompany: "Test Company",
				Address:     "123 Test St",
			},
			cookies: nil,
			mockSetup: func() {
				mockLogs.EXPECT().SaveLogs(gomock.Any(), gomock.Any()).Return(&logs.LogResponse{}, nil).AnyTimes()
				mockDbAuth.EXPECT().RegisterCompany(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, status.Error(codes.AlreadyExists, "company already exists"))
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   `{"error":"Ошибка регистрации компании","message":"company already exists"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, err := json.Marshal(tt.body)
			if s, ok := tt.body.(string); ok {
				bodyBytes = []byte(s)
			} else if err != nil {
				t.Fatalf("Failed to marshal body: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(bodyBytes))
			for _, cookie := range tt.cookies {
				req.AddCookie(cookie)
			}
			w := httptest.NewRecorder()

			tt.mockSetup()

			h.Handler.Register(w, req)

			t.Logf("Response body: %s", w.Body.String())
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestRefreshToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogs := mocks.NewMockLogsServiceClient(ctrl)

	h := &testHandler{
		Handler: transport_rest.NewHandler(),
	}

	tests := []struct {
		name           string
		cookies        []*http.Cookie
		mockSetup      func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			cookies: []*http.Cookie{
				{Name: "refresh_token", Value: "valid-refresh"},
				{Name: "access_token", Value: "invalid-access"},
			},
			mockSetup: func() {
				mockLogs.EXPECT().SaveLogs(gomock.Any(), gomock.Any()).Return(&logs.LogResponse{}, nil).AnyTimes()
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Токен успешно обновлён"}`,
		},
		{
			name: "Invalid Refresh Token",
			cookies: []*http.Cookie{
				{Name: "refresh_token", Value: "invalid-refresh"},
				{Name: "access_token", Value: "invalid-access"},
			},
			mockSetup: func() {
				mockLogs.EXPECT().SaveLogs(gomock.Any(), gomock.Any()).Return(&logs.LogResponse{}, nil).AnyTimes()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"Не удалось создать токен","message":"invalid refresh token"}`,
		},
		{
			name: "Missing Refresh Token",
			cookies: []*http.Cookie{
				{Name: "access_token", Value: "invalid-access"},
			},
			mockSetup: func() {
				mockLogs.EXPECT().SaveLogs(gomock.Any(), gomock.Any()).Return(&logs.LogResponse{}, nil).AnyTimes()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"Не удалось создать токен","message":"cookie refresh_token not found"}`,
		},
		{
			name: "Missing Access Token",
			cookies: []*http.Cookie{
				{Name: "refresh_token", Value: "valid-refresh"},
			},
			mockSetup: func() {
				mockLogs.EXPECT().SaveLogs(gomock.Any(), gomock.Any()).Return(&logs.LogResponse{}, nil).AnyTimes()
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"Не удалось создать токен","message":"cookie access_token not found"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
			for _, cookie := range tt.cookies {
				req.AddCookie(cookie)
			}
			w := httptest.NewRecorder()

			tt.mockSetup()

			h.Handler.RefreshToken(w, req)

			t.Logf("Response body: %s", w.Body.String())
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestCheckAuth(t *testing.T) {
	h := transport_rest.NewHandler()

	req := httptest.NewRequest(http.MethodPost, "/auth/check", nil)
	w := httptest.NewRecorder()

	h.CheckAuth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"message":"Проверка пройдена"}`, w.Body.String())
}

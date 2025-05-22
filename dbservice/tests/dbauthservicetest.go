package tests

import (
	"context"
	"crmSystem/dbauthservice"
	"crmSystem/proto/dbauth"
	"crmSystem/proto/redis"
	"crmSystem/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"net/http"
	"os"
	"testing"
)

// MockRedisClient is a mock implementation of the RedisServiceClient.
type MockRedisClient struct {
	redis.RedisServiceClient
	getFunc  func(ctx context.Context, in *redis.GetRedisRequest, opts ...grpc.CallOption) (*redis.GetRedisResponse, error)
	saveFunc func(ctx context.Context, in *redis.SaveRedisRequest, opts ...grpc.CallOption) (*redis.SaveRedisResponse, error)
}

func (m *MockRedisClient) Get(ctx context.Context, in *redis.GetRedisRequest, opts ...grpc.CallOption) (*redis.GetRedisResponse, error) {
	return m.getFunc(ctx, in)
}

func (m *MockRedisClient) Save(ctx context.Context, in *redis.SaveRedisRequest, opts ...grpc.CallOption) (*redis.SaveRedisResponse, error) {
	return m.saveFunc(ctx, in)
}

// TestLoginDB tests the LoginDB method of AuthServiceServer.
func TestLoginDB(t *testing.T) {
	// Initialize the MapConnectionsDB
	serverPool := utils.NewMapConnectionsDB()

	// Set environment variable for auth database name
	err := os.Setenv("DB_AUTH_NAME", "auth_db")
	if err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}

	// Create mock database for auth
	authDB, authMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create auth mock database: %v", err)
	}
	defer authDB.Close()

	// Create mock database for company
	companyDB, companyMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create company mock database: %v", err)
	}
	defer companyDB.Close()

	// Manually add mock databases to the server pool
	serverPool.MapDB["auth_db"] = authDB
	serverPool.MapDB["test_company_db"] = companyDB

	// Create the AuthServiceServer instance
	authService := dbauthservice.NewGRPCDBAuthService(serverPool)

	// Test cases
	tests := []struct {
		name           string
		req            *dbauth.LoginDBRequest
		ctx            context.Context
		prepareMocks   func(authMock, companyMock sqlmock.Sqlmock, redisClient *MockRedisClient)
		expectedResp   *dbauth.LoginDBResponse
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name: "Successful login",
			req: &dbauth.LoginDBRequest{
				Email:    "user@example.com",
				Phone:    "1234567890",
				Password: "password123",
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(authMock, companyMock sqlmock.Sqlmock, redisClient *MockRedisClient) {
				// Mock Redis Get (cache miss)
				redisClient.getFunc = func(ctx context.Context, in *redis.GetRedisRequest, opts ...grpc.CallOption) (*redis.GetRedisResponse, error) {
					return &redis.GetRedisResponse{Status: http.StatusNotFound}, nil
				}
				redisClient.saveFunc = func(ctx context.Context, in *redis.SaveRedisRequest, opts ...grpc.CallOption) (*redis.SaveRedisResponse, error) {
					return &redis.SaveRedisResponse{Status: http.StatusOK}, nil
				}

				// Mock authusers table query
				authMock.ExpectQuery(`SELECT id, company_id FROM authusers WHERE \(email = \$1 OR phone = \$2\) AND password = \$3`).
					WithArgs("user@example.com", "1234567890", "password123").
					WillReturnRows(sqlmock.NewRows([]string{"id", "company_id"}).AddRow("1", "100"))

				// Mock companies table query
				authMock.ExpectQuery(`SELECT dbName FROM companies WHERE id = \$1`).
					WithArgs("100").
					WillReturnRows(sqlmock.NewRows([]string{"dbName"}).AddRow("test_company_db"))

				// Mock users table query in company DB
				companyMock.ExpectQuery(`SELECT id FROM users WHERE authId = \$1`).
					WithArgs("1").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user1"))
			},
			expectedResp: &dbauth.LoginDBResponse{
				Message: "Пользователь найден",
			},
			expectedErr: false,
		},
		{
			name: "User not found",
			req: &dbauth.LoginDBRequest{
				Email:    "user@example.com",
				Phone:    "1234567890",
				Password: "wrongpassword",
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(authMock, companyMock sqlmock.Sqlmock, redisClient *MockRedisClient) {
				// Mock Redis Get (cache miss)
				redisClient.getFunc = func(ctx context.Context, in *redis.GetRedisRequest, opts ...grpc.CallOption) (*redis.GetRedisResponse, error) {
					return &redis.GetRedisResponse{Status: http.StatusNotFound}, nil
				}

				// Mock authusers table query (no rows)
				authMock.ExpectQuery(`SELECT id, company_id FROM authusers WHERE \(email = \$1 OR phone = \$2\) AND password = \$3`).
					WithArgs("user@example.com", "1234567890", "wrongpassword").
					WillReturnError(sql.ErrNoRows)
			},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "Пользователь не найден",
		},
		{
			name: "Missing authorization token",
			req: &dbauth.LoginDBRequest{
				Email:    "user@example.com",
				Phone:    "1234567890",
				Password: "password123",
			},
			ctx:            context.Background(),
			prepareMocks:   func(authMock, companyMock sqlmock.Sqlmock, redisClient *MockRedisClient) {},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "токен отсутствует",
		},
		{
			name: "Redis cache hit",
			req: &dbauth.LoginDBRequest{
				Email:    "user@example.com",
				Phone:    "1234567890",
				Password: "password123",
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(authMock, companyMock sqlmock.Sqlmock, redisClient *MockRedisClient) {
				// Mock Redis Get (cache hit)
				redisClient.getFunc = func(ctx context.Context, in *redis.GetRedisRequest, opts ...grpc.CallOption) (*redis.GetRedisResponse, error) {
					data := struct {
						Database  string
						UserId    string
						CompanyId string
					}{
						Database:  "test_company_db",
						UserId:    "user1",
						CompanyId: "100",
					}
					jsonData, _ := json.Marshal(data)
					return &redis.GetRedisResponse{
						Status:  http.StatusOK,
						Message: string(jsonData),
					}, nil
				}
			},
			expectedResp: &dbauth.LoginDBResponse{
				Message: "Пользователь найден",
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock Redis client
			redisClient := &MockRedisClient{}
			// Override utils.RedisServiceConnector to return mock Redis client

			// Prepare mocks for the test case
			tt.prepareMocks(authMock, companyMock, redisClient)

			// Call the method
			resp, err := authService.LoginDB(tt.ctx, tt.req)

			// Check for expected error
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}

			// Verify that all expectations were met
			assert.NoError(t, authMock.ExpectationsWereMet())
			assert.NoError(t, companyMock.ExpectationsWereMet())
		})
	}
}

// TestRegisterCompany tests the RegisterCompany method of AuthServiceServer.
func TestRegisterCompany(t *testing.T) {
	// Initialize the MapConnectionsDB
	serverPool := utils.NewMapConnectionsDB()

	// Set environment variables
	err := os.Setenv("DB_AUTH_NAME", "auth_db")
	if err != nil {
		t.Fatalf("Failed to set DB_AUTH_NAME: %v", err)
	}
	err = os.Setenv("MIGRATION_COMPANYDB_PATH", "/migrations/company")
	if err != nil {
		t.Fatalf("Failed to set MIGRATION_COMPANYDB_PATH: %v", err)
	}
	err = os.Setenv("FIRST_ROLE", "admin")
	if err != nil {
		t.Fatalf("Failed to set FIRST_ROLE: %v", err)
	}

	// Create mock databases
	authDB, authMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create auth mock database: %v", err)
	}
	defer authDB.Close()

	companyDB, companyMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create company mock database: %v", err)
	}
	defer companyDB.Close()

	// Manually add mock databases to the server pool
	serverPool.MapDB["auth_db"] = authDB
	serverPool.MapDB["test_company_db"] = companyDB

	// Create the AuthServiceServer instance
	authService := dbauthservice.NewGRPCDBAuthService(serverPool)

	// Test cases
	tests := []struct {
		name           string
		req            *dbauth.RegisterCompanyRequest
		ctx            context.Context
		prepareMocks   func(authMock, companyMock sqlmock.Sqlmock, redisClient *MockRedisClient)
		expectedResp   *dbauth.RegisterCompanyResponse
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name: "Successful company registration",
			req: &dbauth.RegisterCompanyRequest{
				NameCompany: "Test Company",
				Address:     "123 Main St",
				Email:       "user@example.com",
				Phone:       "1234567890",
				Password:    "password123",
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(authMock, companyMock sqlmock.Sqlmock, redisClient *MockRedisClient) {
				// Mock Redis Get (cache miss)
				redisClient.getFunc = func(ctx context.Context, in *redis.GetRedisRequest, opts ...grpc.CallOption) (*redis.GetRedisResponse, error) {
					return &redis.GetRedisResponse{Status: http.StatusNotFound}, nil
				}
				redisClient.saveFunc = func(ctx context.Context, in *redis.SaveRedisRequest, opts ...grpc.CallOption) (*redis.SaveRedisResponse, error) {
					return &redis.SaveRedisResponse{Status: http.StatusOK}, nil
				}

				// Mock auth DB transaction
				authMock.ExpectBegin()
				// Mock companies table check (no existing company)
				authMock.ExpectQuery(`SELECT id FROM companies WHERE name = \$1 AND address = \$2`).
					WithArgs("test company", "123 main st").
					WillReturnError(sql.ErrNoRows)
				// Mock companies table insert
				authMock.ExpectQuery(`INSERT INTO companies \(name, address, dbname\) VALUES \(\$1, \$2, \$3\) RETURNING id`).
					WithArgs("test company", "123 main st", "test_company_db").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("100"))
				// Mock authusers table insert
				authMock.ExpectQuery(`INSERT INTO authusers \(email, phone, password, company_id\) VALUES \(\$1, \$2, \$3, \$4\) RETURNING id`).
					WithArgs("user@example.com", "1234567890", "password123", "100").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))
				authMock.ExpectCommit()

				// Mock company DB transaction
				companyMock.ExpectBegin()
				// Mock rights table insert
				companyMock.ExpectQuery(`INSERT INTO rights \(roles\) VALUES \(\$1\) RETURNING id`).
					WithArgs("admin").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				// Mock users table insert
				companyMock.ExpectQuery(`INSERT INTO users \(rightsId, authId\) VALUES \(\$1, \$2\) RETURNING id`).
					WithArgs(1, "1").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user1"))
				// Mock availableactions table insert
				companyMock.ExpectExec(`INSERT INTO availableactions \(roleId, createTasks, createChats, addWorkers\) VALUES \(\$1, \$2, \$3, \$4\)`).
					WithArgs(1, true, true, true).
					WillReturnResult(sqlmock.NewResult(0, 1))
				companyMock.ExpectCommit()
			},
			expectedResp: &dbauth.RegisterCompanyResponse{
				Message: "Регистрация успешна",
			},
			expectedErr: false,
		},
		{
			name: "Existing company",
			req: &dbauth.RegisterCompanyRequest{
				NameCompany: "Test Company",
				Address:     "123 Main St",
				Email:       "user@example.com",
				Phone:       "1234567890",
				Password:    "password123",
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(authMock, companyMock sqlmock.Sqlmock, redisClient *MockRedisClient) {
				// Mock Redis Get (cache miss)
				redisClient.getFunc = func(ctx context.Context, in *redis.GetRedisRequest, opts ...grpc.CallOption) (*redis.GetRedisResponse, error) {
					return &redis.GetRedisResponse{Status: http.StatusNotFound}, nil
				}

				// Mock auth DB transaction
				authMock.ExpectBegin()
				// Mock companies table check (existing company)
				authMock.ExpectQuery(`SELECT id FROM companies WHERE name = \$1 AND address = \$2`).
					WithArgs("test company", "123 main st").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("100"))
			},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "компания с таким именем и адресом уже существует",
		},
		{
			name: "Duplicate email",
			req: &dbauth.RegisterCompanyRequest{
				NameCompany: "Test Company",
				Address:     "123 Main St",
				Email:       "user@example.com",
				Phone:       "1234567890",
				Password:    "password123",
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(authMock, companyMock sqlmock.Sqlmock, redisClient *MockRedisClient) {
				// Mock Redis Get (cache miss)
				redisClient.getFunc = func(ctx context.Context, in *redis.GetRedisRequest, opts ...grpc.CallOption) (*redis.GetRedisResponse, error) {
					return &redis.GetRedisResponse{Status: http.StatusNotFound}, nil
				}

				// Mock auth DB transaction
				authMock.ExpectBegin()
				// Mock companies table check (no existing company)
				authMock.ExpectQuery(`SELECT id FROM companies WHERE name = \$1 AND address = \$2`).
					WithArgs("test company", "123 main st").
					WillReturnError(sql.ErrNoRows)
				// Mock companies table insert
				authMock.ExpectQuery(`INSERT INTO companies \(name, address, dbname\) VALUES \(\$1, \$2, \$3\) RETURNING id`).
					WithArgs("test company", "123 main st", "test_company_db").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("100"))
				// Mock authusers table insert (duplicate email)
				authMock.ExpectQuery(`INSERT INTO authusers \(email, phone, password, company_id\) VALUES \(\$1, \$2, \$3, \$4\) RETURNING id`).
					WithArgs("user@example.com", "1234567890", "password123", "100").
					WillReturnError(fmt.Errorf("authusers_email_key"))
			},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "дубликат почты",
		},
		{
			name: "Redis cache hit",
			req: &dbauth.RegisterCompanyRequest{
				NameCompany: "Test Company",
				Address:     "123 Main St",
				Email:       "user@example.com",
				Phone:       "1234567890",
				Password:    "password123",
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(authMock, companyMock sqlmock.Sqlmock, redisClient *MockRedisClient) {
				// Mock Redis Get (cache hit)
				redisClient.getFunc = func(ctx context.Context, in *redis.GetRedisRequest, opts ...grpc.CallOption) (*redis.GetRedisResponse, error) {
					data := struct {
						Message   string
						Database  string
						CompanyId string
						UserId    string
					}{
						Message:   "test company",
						Database:  "test_company_db",
						CompanyId: "100",
						UserId:    "user1",
					}
					jsonData, _ := json.Marshal(data)
					return &redis.GetRedisResponse{
						Status:  http.StatusOK,
						Message: string(jsonData),
					}, nil
				}
			},
			expectedResp: &dbauth.RegisterCompanyResponse{
				Message: "Регистрация успешна",
			},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock Redis client
			redisClient := &MockRedisClient{}
			// Override utils.RedisServiceConnector to return mock Redis client

			// Prepare mocks for the test case
			tt.prepareMocks(authMock, companyMock, redisClient)

			// Call the method
			resp, err := authService.RegisterCompany(tt.ctx, tt.req)

			// Check for expected error
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResp, resp)
			}

			// Verify that all expectations were met
			assert.NoError(t, authMock.ExpectationsWereMet())
			assert.NoError(t, companyMock.ExpectationsWereMet())
		})
	}
}

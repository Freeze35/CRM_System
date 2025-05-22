package tests

import (
	"context"
	"crmSystem/dbadminservice"
	pbAdmin "crmSystem/proto/dbadmin"
	"crmSystem/utils"
	"database/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"os"
	"testing"
)

// TestRegisterUsersInCompany tests the RegisterUsersInCompany method of AdminServiceServer.
func TestRegisterUsersInCompany(t *testing.T) {
	// Initialize the MapConnectionsDB
	serverPool := utils.NewMapConnectionsDB()

	// Set environment variable for auth database name
	err := os.Setenv("DB_AUTH_NAME", "auth_db")
	if err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}

	// Create mock databases for auth and company
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

	// Create the AdminServiceServer instance
	adminService := dbadminservice.NewGRPCDBAdminService(serverPool)

	// Test cases
	tests := []struct {
		name           string
		req            *pbAdmin.RegisterUsersRequest
		ctx            context.Context
		prepareMocks   func(authMock, companyMock sqlmock.Sqlmock)
		expectedResp   *pbAdmin.RegisterUsersResponse
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name: "Successful registration of new users",
			req: &pbAdmin.RegisterUsersRequest{
				CompanyId: "1",
				Users: []*pbAdmin.User{
					{Email: "user1@example.com", Phone: "1234567890", RoleId: 1},
					{Email: "user2@example.com", Phone: "0987654321", RoleId: 2},
				},
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "admin1",
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(authMock, companyMock sqlmock.Sqlmock) {
				// Mock authusers table checks (no existing users)
				authMock.ExpectBegin()
				authMock.ExpectQuery(`SELECT id FROM authusers WHERE email = \$1 OR phone = \$2`).
					WithArgs("user1@example.com", "1234567890").
					WillReturnError(sql.ErrNoRows)
				authMock.ExpectQuery(`SELECT id FROM authusers WHERE email = \$1 OR phone = \$2`).
					WithArgs("user2@example.com", "0987654321").
					WillReturnError(sql.ErrNoRows)

				// Mock authusers table inserts
				authMock.ExpectPrepare(`INSERT INTO authusers \(email, phone, password, company_id\) VALUES \(\$1, \$2, \$3, \$4\) RETURNING id`)
				authMock.ExpectQuery(`INSERT INTO authusers`).
					WithArgs("user1@example.com", "1234567890", "default_password", int64(1)).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				authMock.ExpectQuery(`INSERT INTO authusers`).
					WithArgs("user2@example.com", "0987654321", "default_password", int64(1)).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
				authMock.ExpectCommit()

				// Mock company users table inserts
				companyMock.ExpectBegin()
				companyMock.ExpectQuery(`INSERT INTO users \(rightsId, authId\) VALUES \(\$1, \$2\) RETURNING id`).
					WithArgs(int64(1), 1).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user1_id"))
				companyMock.ExpectQuery(`INSERT INTO users \(rightsId, authId\) VALUES \(\$1, \$2\) RETURNING id`).
					WithArgs(int64(2), 2).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user2_id"))
				companyMock.ExpectCommit()
			},
			expectedResp: &pbAdmin.RegisterUsersResponse{
				Users: []*pbAdmin.UserResponse{
					{Email: "user1@example.com", Phone: "1234567890", RoleId: 1, Password: "default_password"},
					{Email: "user2@example.com", Phone: "0987654321", RoleId: 2, Password: "default_password"},
				},
				Message: "Пользователи успешно добавлены",
			},
			expectedErr: false,
		},
		{
			name: "Existing user in authusers",
			req: &pbAdmin.RegisterUsersRequest{
				CompanyId: "1",
				Users: []*pbAdmin.User{
					{Email: "user1@example.com", Phone: "1234567890", RoleId: 1},
				},
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "admin1",
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(authMock, companyMock sqlmock.Sqlmock) {
				// Mock authusers table check (existing user)
				authMock.ExpectBegin()
				authMock.ExpectQuery(`SELECT id FROM authusers WHERE email = \$1 OR phone = \$2`).
					WithArgs("user1@example.com", "1234567890").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				authMock.ExpectCommit()

				// Mock company users table check (no user in company)
				companyMock.ExpectBegin()
				companyMock.ExpectQuery(`SELECT id FROM users WHERE authId = \$1`).
					WithArgs(1).
					WillReturnError(sql.ErrNoRows)

				// Mock company users table insert
				companyMock.ExpectQuery(`INSERT INTO users \(rightsId, authId\) VALUES \(\$1, \$2\) RETURNING id`).
					WithArgs(int64(1), 1).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user1_id"))
				companyMock.ExpectCommit()
			},
			expectedResp: &pbAdmin.RegisterUsersResponse{
				Users: []*pbAdmin.UserResponse{
					{Email: "user1@example.com", Phone: "1234567890", RoleId: 1, Password: "default_password"},
				},
				Message: "Пользователи успешно добавлены",
			},
			expectedErr: false,
		},
		{
			name: "Missing database metadata",
			req: &pbAdmin.RegisterUsersRequest{
				CompanyId: "1",
				Users:     []*pbAdmin.User{{Email: "user1@example.com", Phone: "1234567890", RoleId: 1}},
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"user-id", "admin1",
				"authorization", "Bearer test_token",
			)),
			prepareMocks:   func(authMock, companyMock sqlmock.Sqlmock) {},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "database не найдена в метаданных",
		},
		{
			name: "Missing user-id metadata",
			req: &pbAdmin.RegisterUsersRequest{
				CompanyId: "1",
				Users:     []*pbAdmin.User{{Email: "user1@example.com", Phone: "1234567890", RoleId: 1}},
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"authorization", "Bearer test_token",
			)),
			prepareMocks:   func(authMock, companyMock sqlmock.Sqlmock) {},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "userId не найдена в метаданных",
		},
		{
			name: "Missing authorization token",
			req: &pbAdmin.RegisterUsersRequest{
				CompanyId: "1",
				Users:     []*pbAdmin.User{{Email: "user1@example.com", Phone: "1234567890", RoleId: 1}},
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "admin1",
			)),
			prepareMocks:   func(authMock, companyMock sqlmock.Sqlmock) {},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "токен отсутствует",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare mocks for the test case
			tt.prepareMocks(authMock, companyMock)

			// Call the method
			resp, err := adminService.RegisterUsersInCompany(tt.ctx, tt.req)

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

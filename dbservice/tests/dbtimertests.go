package tests

import (
	"context"
	"crmSystem/dbtimerservice"
	"crmSystem/proto/dbtimer"
	"crmSystem/utils"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"testing"
	"time"
)

// TestChangeTimerDB tests the ChangeTimerDB method of TimerServiceServer.
func TestChangeTimerDB(t *testing.T) {
	serverPool := utils.NewMapConnectionsDB()
	companyDB, companyMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer companyDB.Close()
	serverPool.MapDB["test_company_db"] = companyDB
	timerService := dbtimerservice.NewGRPCDBTimerService(serverPool)

	tests := []struct {
		name           string
		req            *dbtimer.ChangeTimerRequestDB
		ctx            context.Context
		prepareMocks   func(companyMock sqlmock.Sqlmock)
		expectedResp   *dbtimer.ChangeTimerResponseDB
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name: "Successful timer change",
			req:  &dbtimer.ChangeTimerRequestDB{},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "user1",
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(companyMock sqlmock.Sqlmock) {
				startTime := time.Now()
				companyMock.ExpectQuery(`UPDATE user_timers SET start_time = CASE WHEN \$1 IS NOT NULL AND \$1 != '' THEN \$1 ELSE start_time END, end_time = CASE WHEN \$2 IS NOT NULL AND \$2 != '' THEN \$2 ELSE end_time END, is_active = CASE WHEN \$3 IS NOT NULL AND \$3 != '' THEN \$3 ELSE is_active END, description = CASE WHEN \$4 IS NOT NULL AND \$4 != '' THEN \$4 ELSE description END WHERE user_id = \$5 RETURNING start_time, end_time, id, duration, description, is_active`).
					WithArgs("", "", "", "", "user1").
					WillReturnRows(sqlmock.NewRows([]string{"start_time", "end_time", "id", "duration", "description", "is_active"}).
						AddRow(startTime, nil, uint64(1), "", "test timer", true))
			},
			expectedResp: &dbtimer.ChangeTimerResponseDB{
				TimerId:     1,
				StartTime:   time.Now().UTC().Format(time.RFC3339),
				EndTime:     "",
				Duration:    "",
				Description: "test timer",
				Active:      true,
				Message:     "Таймер изменён",
			},
			expectedErr: false,
		},
		{
			name: "Missing database metadata",
			req:  &dbtimer.ChangeTimerRequestDB{},
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
			name: "Database error",
			req:  &dbtimer.ChangeTimerRequestDB{},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "user1",
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(companyMock sqlmock.Sqlmock) {
				companyMock.ExpectQuery(`UPDATE user_timers SET start_time = CASE WHEN \$1 IS NOT NULL AND \$1 != '' THEN \$1 ELSE start_time END, end_time = CASE WHEN \$2 IS NOT NULL AND \$2 != '' THEN \$2 ELSE end_time END, is_active = CASE WHEN \$3 IS NOT NULL AND \$3 != '' THEN \$3 ELSE is_active END, description = CASE WHEN \$4 IS NOT NULL AND \$4 != '' THEN \$4 ELSE description END WHERE user_id = \$5 RETURNING start_time, end_time, id, duration, description, is_active`).
					WithArgs("", "", "", "", "user1").
					WillReturnError(fmt.Errorf("database error"))
			},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "Ошибка при закрытии старого таймера: database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepareMocks(companyMock)
			resp, err := timerService.ChangeTimerDB(tt.ctx, tt.req)
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				if resp != nil {
					resp.StartTime = tt.expectedResp.StartTime
				}
				assert.Equal(t, tt.expectedResp, resp)
			}
			assert.NoError(t, companyMock.ExpectationsWereMet())
		})
	}
}

// TestStartTimerDB tests the StartTimerDB method of TimerServiceServer.
func TestStartTimerDB(t *testing.T) {
	serverPool := utils.NewMapConnectionsDB()
	companyDB, companyMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer companyDB.Close()
	serverPool.MapDB["test_company_db"] = companyDB
	timerService := dbtimerservice.NewGRPCDBTimerService(serverPool)

	tests := []struct {
		name           string
		req            *dbtimer.StartEndTimerRequestDB
		ctx            context.Context
		prepareMocks   func(companyMock sqlmock.Sqlmock)
		expectedResp   *dbtimer.StartEndTimerResponseDB
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name: "Successful start new timer",
			req: &dbtimer.StartEndTimerRequestDB{
				Description: "test timer",
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "user1",
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(companyMock sqlmock.Sqlmock) {
				companyMock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM user_timers WHERE user_id = \$1 AND is_active = TRUE\)`).
					WithArgs("user1").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
				startTime := time.Now()
				companyMock.ExpectQuery(`INSERT INTO user_timers \(user_id, start_time, description, is_active\) VALUES \(\$1, NOW\(\), \$2, TRUE\) RETURNING start_time, end_time, id`).
					WithArgs("user1", "test timer").
					WillReturnRows(sqlmock.NewRows([]string{"start_time", "end_time", "id"}).
						AddRow(startTime, nil, uint64(1)))
			},
			expectedResp: &dbtimer.StartEndTimerResponseDB{
				TimerId:   1,
				StartTime: time.Now().UTC().Format(time.RFC3339),
				EndTime:   "",
				Message:   "Таймер запушен",
			},
			expectedErr: false,
		},
		{
			name: "Start timer with existing active timer",
			req: &dbtimer.StartEndTimerRequestDB{
				Description: "test timer",
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "user1",
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(companyMock sqlmock.Sqlmock) {
				companyMock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM user_timers WHERE user_id = \$1 AND is_active = TRUE\)`).
					WithArgs("user1").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				companyMock.ExpectBegin()
				companyMock.ExpectExec(`UPDATE user_timers SET end_time = NOW\(\), is_active = FALSE WHERE user_id = \$1 AND is_active = TRUE`).
					WithArgs("user1").
					WillReturnResult(sqlmock.NewResult(0, 1))
				startTime := time.Now()
				companyMock.ExpectQuery(`INSERT INTO user_timers \(user_id, start_time, description, is_active\) VALUES \(\$1, NOW\(\), \$2, TRUE\) RETURNING start_time, end_time, id`).
					WithArgs("user1", "test timer").
					WillReturnRows(sqlmock.NewRows([]string{"start_time", "end_time", "id"}).
						AddRow(startTime, nil, uint64(1)))
				companyMock.ExpectCommit()
			},
			expectedResp: &dbtimer.StartEndTimerResponseDB{
				TimerId:   1,
				StartTime: time.Now().UTC().Format(time.RFC3339),
				EndTime:   "",
				Message:   "Таймер запушен",
			},
			expectedErr: false,
		},
		{
			name: "Missing user-id metadata",
			req:  &dbtimer.StartEndTimerRequestDB{},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"authorization", "Bearer test_token",
			)),
			prepareMocks:   func(companyMock sqlmock.Sqlmock) {},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "userId не найдена в метаданных",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepareMocks(companyMock)
			resp, err := timerService.StartTimerDB(tt.ctx, tt.req)
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				if resp != nil {
					resp.StartTime = tt.expectedResp.StartTime
				}
				assert.Equal(t, tt.expectedResp, resp)
			}
			assert.NoError(t, companyMock.ExpectationsWereMet())
		})
	}
}

// TestGetWorkingTimerDB tests the GetWorkingTimerDB method of TimerServiceServer.
func TestGetWorkingTimerDB(t *testing.T) {
	serverPool := utils.NewMapConnectionsDB()
	companyDB, companyMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer companyDB.Close()
	serverPool.MapDB["test_company_db"] = companyDB
	timerService := dbtimerservice.NewGRPCDBTimerService(serverPool)

	tests := []struct {
		name           string
		req            *dbtimer.WorkingTimerRequestDB
		ctx            context.Context
		prepareMocks   func(companyMock sqlmock.Sqlmock)
		expectedResp   *dbtimer.WorkingTimerResponseDB
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name: "Successful get working timer",
			req:  &dbtimer.WorkingTimerRequestDB{},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "user1",
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(companyMock sqlmock.Sqlmock) {
				startTime := time.Now()
				companyMock.ExpectQuery(`SELECT start_time, end_time, id FROM user_timers WHERE user_id = \$1 AND is_active = TRUE`).
					WithArgs("user1").
					WillReturnRows(sqlmock.NewRows([]string{"start_time", "end_time", "id"}).
						AddRow(startTime, nil, uint64(1)))
			},
			expectedResp: &dbtimer.WorkingTimerResponseDB{
				TimerId:   1,
				StartTime: time.Now().UTC().Format(time.RFC3339),
				EndTime:   "",
				Message:   "Найден незавершённый таймер",
			},
			expectedErr: false,
		},
		{
			name: "Missing authorization token",
			req:  &dbtimer.WorkingTimerRequestDB{},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "user1",
			)),
			prepareMocks:   func(companyMock sqlmock.Sqlmock) {},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "Не удалось извлечь токен для логирования",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepareMocks(companyMock)
			resp, err := timerService.GetWorkingTimerDB(tt.ctx, tt.req)
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				if resp != nil {
					resp.StartTime = tt.expectedResp.StartTime
				}
				assert.Equal(t, tt.expectedResp, resp)
			}
			assert.NoError(t, companyMock.ExpectationsWereMet())
		})
	}
}

// TestEndTimerDB tests the EndTimerDB method of TimerServiceServer.
func TestEndTimerDB(t *testing.T) {
	serverPool := utils.NewMapConnectionsDB()
	companyDB, companyMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer companyDB.Close()
	serverPool.MapDB["test_company_db"] = companyDB
	timerService := dbtimerservice.NewGRPCDBTimerService(serverPool)

	tests := []struct {
		name           string
		req            *dbtimer.StartEndTimerRequestDB
		ctx            context.Context
		prepareMocks   func(companyMock sqlmock.Sqlmock)
		expectedResp   *dbtimer.StartEndTimerResponseDB
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name: "Successful end timer",
			req:  &dbtimer.StartEndTimerRequestDB{},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "user1",
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(companyMock sqlmock.Sqlmock) {
				startTime := time.Now().Add(-1 * time.Hour)
				endTime := time.Now()
				companyMock.ExpectQuery(`UPDATE user_timers SET end_time = NOW\(\), is_active = FALSE WHERE user_id = \$1 AND is_active = TRUE RETURNING start_time, end_time, id`).
					WithArgs("user1").
					WillReturnRows(sqlmock.NewRows([]string{"start_time", "end_time", "id"}).
						AddRow(startTime, endTime, uint64(1)))
			},
			expectedResp: &dbtimer.StartEndTimerResponseDB{
				TimerId:   1,
				StartTime: time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339),
				EndTime:   time.Now().UTC().Format(time.RFC3339),
				Message:   "Таймер завершён",
			},
			expectedErr: false,
		},
		{
			name: "Database error",
			req:  &dbtimer.StartEndTimerRequestDB{},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "user1",
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(companyMock sqlmock.Sqlmock) {
				companyMock.ExpectQuery(`UPDATE user_timers SET end_time = NOW\(\), is_active = FALSE WHERE user_id = \$1 AND is_active = TRUE RETURNING start_time, end_time, id`).
					WithArgs("user1").
					WillReturnError(fmt.Errorf("database error"))
			},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "Ошибка при закрытии старого таймера: database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepareMocks(companyMock)
			resp, err := timerService.EndTimerDB(tt.ctx, tt.req)
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				if resp != nil {
					resp.StartTime = tt.expectedResp.StartTime
					resp.EndTime = tt.expectedResp.EndTime
				}
				assert.Equal(t, tt.expectedResp, resp)
			}
			assert.NoError(t, companyMock.ExpectationsWereMet())
		})
	}
}

// TestAddTimerDB tests the AddTimerDB method of TimerServiceServer.
func TestAddTimerDB(t *testing.T) {
	serverPool := utils.NewMapConnectionsDB()
	companyDB, companyMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer companyDB.Close()
	serverPool.MapDB["test_company_db"] = companyDB
	timerService := dbtimerservice.NewGRPCDBTimerService(serverPool)

	tests := []struct {
		name           string
		req            *dbtimer.AddTimerRequestDB
		ctx            context.Context
		prepareMocks   func(companyMock sqlmock.Sqlmock)
		expectedResp   *dbtimer.AddTimerResponseDB
		expectedErr    bool
		expectedErrMsg string
	}{
		{
			name: "Successful add timer",
			req: &dbtimer.AddTimerRequestDB{
				Description: "test timer",
				StartTime:   time.Now().UTC().Format(time.RFC3339),
				EndTime:     time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339),
			},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"database", "test_company_db",
				"user-id", "user1",
				"authorization", "Bearer test_token",
			)),
			prepareMocks: func(companyMock sqlmock.Sqlmock) {
				startTime := time.Now()
				endTime := time.Now().Add(1 * time.Hour)
				companyMock.ExpectQuery(`INSERT INTO user_timers \(user_id, start_time, end_time, description\) VALUES \(\$1, \$2, \$3, \$4\) RETURNING start_time, end_time, id, duration, description`).
					WithArgs("user1", startTime, endTime, "test timer").
					WillReturnRows(sqlmock.NewRows([]string{"start_time", "end_time", "id", "duration", "description"}).
						AddRow(startTime, endTime, uint64(1), "", "test timer"))
			},
			expectedResp: &dbtimer.AddTimerResponseDB{
				TimerId:     1,
				StartTime:   time.Now().UTC().Format(time.RFC3339),
				EndTime:     time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339),
				Duration:    "",
				Description: "test timer",
				Message:     "Таймер добавлен",
			},
			expectedErr: false,
		},
		{
			name: "Missing database metadata",
			req:  &dbtimer.AddTimerRequestDB{},
			ctx: metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"user-id", "user1",
				"authorization", "Bearer test_token",
			)),
			prepareMocks:   func(companyMock sqlmock.Sqlmock) {},
			expectedResp:   nil,
			expectedErr:    true,
			expectedErrMsg: "database не найдена в метаданных",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepareMocks(companyMock)
			resp, err := timerService.AddTimerDB(tt.ctx, tt.req)
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				if resp != nil {
					resp.StartTime = tt.expectedResp.StartTime
					resp.EndTime = tt.expectedResp.EndTime
				}
				assert.Equal(t, tt.expectedResp, resp)
			}
			assert.NoError(t, companyMock.ExpectationsWereMet())
		})
	}
}

package dbtimerservice

import (
	"context"
	"crmSystem/proto/dbtimer"
	"crmSystem/utils"
	"database/sql"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log"
	"time"
)

type TimerServiceServer struct {
	dbtimer.UnsafeDbTimerServiceServer
	connectionsMap *utils.MapConnectionsDB // Используем указатель
}

func NewGRPCDBTimerService(mapConnect *utils.MapConnectionsDB) *TimerServiceServer {
	return &TimerServiceServer{
		connectionsMap: mapConnect,
	}
}

func (s *TimerServiceServer) ChangeTimerDB(ctx context.Context, req *dbtimer.ChangeTimerRequestDB) (*dbtimer.ChangeTimerResponseDB, error) {

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

	// Открываем соединение с базой данных Авторизации
	dsn := utils.DsnString(database)
	// Получаем соединение с базой данных
	db, err := s.connectionsMap.GetDb(dsn)
	if err != nil {
		// Если произошла ошибка подключения, логируем её и возвращаем ответ с ошибкой.
		log.Printf("Ошибка подключения к базе данных: %s", err)
		return &dbtimer.ChangeTimerResponseDB{
			Message: fmt.Sprintf("Ошибка подключения к базе данных: %s.", err), // Сообщение об ошибке.
		}, err
	}

	// Переменные для хранения значений start_time и end_time
	var startTime, endTime sql.NullTime
	var duration, description string
	var timerId uint64
	var isActive bool

	// Изменение таймера
	err = db.QueryRowContext(ctx, `
        UPDATE user_timers
			SET
				start_time = CASE WHEN $1 IS NOT NULL AND $1 != '' THEN $1 ELSE start_time END,
				end_time = CASE WHEN $1 IS NOT NULL AND $1 != '' THEN $1 ELSE end_time END,
				is_active   BOOLEAN = CASE WHEN $1 IS NOT NULL AND $1 != '' THEN $1 ELSE is_active   BOOLEAN END,
				description = CASE WHEN $1 IS NOT NULL AND $1 != '' THEN $1 ELSE description END,
				end_time = CASE WHEN $2 IS NOT NULL THEN $2 ELSE end_time END
			WHERE id = $3 RETURNING start_time, end_time,id,duration,description,is_active
    	`, userId[0]).Scan(&startTime, &endTime, &timerId, &duration, &description, isActive)

	if err != nil {
		log.Printf("Ошибка при закрытии старого таймера: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка при закрытии старого таймера: %s.", err))
	}

	// Преобразование времени в строку в формате ISO 8601 (UTC)
	var startTimeStr, endTimeStr string

	if startTime.Valid {
		startTimeStr = startTime.Time.UTC().Format(time.RFC3339)
	}

	if endTime.Valid {
		endTimeStr = endTime.Time.UTC().Format(time.RFC3339)
	}

	res := dbtimer.ChangeTimerResponseDB{
		StartTime:   startTimeStr,
		EndTime:     endTimeStr,
		TimerId:     timerId,
		Duration:    duration,
		Description: description,
		Active:      isActive,
		Message:     fmt.Sprintf("Таймер изменён"), // Сообщение об ошибке.
	}

	return &res, nil

}

func (s *TimerServiceServer) StartTimerDB(ctx context.Context, req *dbtimer.StartEndTimerRequestDB) (*dbtimer.StartEndTimerResponseDB, error) {

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

	// Открываем соединение с базой данных Авторизации
	dsn := utils.DsnString(database)
	// Получаем соединение с базой данных
	db, err := s.connectionsMap.GetDb(dsn)
	if err != nil {
		// Если произошла ошибка подключения, логируем её и возвращаем ответ с ошибкой.
		log.Printf("Ошибка подключения к базе данных: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка подключения к базе данных: %s.", err))
	}

	// SQL-запрос для проверки существования активного таймера.
	query := `
    SELECT EXISTS(
        SELECT 1
        FROM user_timers
        WHERE user_id = $1 AND is_active = TRUE
    );
`

	// Переменная для сохранения результата проверки.
	var exists bool

	// Выполняем запрос с параметром user_id и получением открытого таймера exists.
	err = db.QueryRowContext(ctx, query, userId).Scan(&exists)

	if err != nil {
		// Если произошла ошибка подключения, логируем её и возвращаем ответ с ошибкой.
		log.Printf("Ошибка запроса к базе данных: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка запроса к базе данных: %s.", err))
	}

	// Переменные для хранения значений start_time и end_time
	var startTime, endTime sql.NullTime
	var timerId uint64

	if exists {

		_, err = db.Exec(`UPDATE user_timers SET end_time = NOW(), 
				    is_active = FALSE WHERE user_id = $1 AND is_active = TRUE `, userId)
		if err != nil {
			// Если произошла ошибка подключения, логируем её и возвращаем ответ с ошибкой.
			log.Printf("Ошибка обновления для закрытия таймера: %s", err)
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка обновления для закрытия таймера: %s.", err))
		}

		// Начало транзакции
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Не удалось начать транзакцию: %s", err)
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("Не удалось начать транзакцию: %s.", err))
		}

		// Закрытие старого таймера
		_, err = tx.Exec(`
        UPDATE user_timers
        SET end_time = NOW(), is_active = FALSE
        WHERE user_id = $1 AND is_active = TRUE
    	`, userId)

		if err != nil {
			tx.Rollback()
			log.Printf("Ошибка при закрытии старого таймера: %s", err)
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка при закрытии старого таймера: %s.", err))
		}

		// Создание нового таймера
		err = db.QueryRowContext(ctx, `
        INSERT INTO user_timers (user_id, start_time, description, is_active)
        VALUES ($1, NOW(), $2, TRUE) RETURNING start_time, end_time,id
    	`, userId, req.Description).Scan(&startTime, &endTime, &timerId)

		if err != nil {
			tx.Rollback()
			log.Printf("Ошибка при создании нового таймера: %s", err)
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка при создании нового таймера: %s.", err))
		}

		// Завершение транзакции
		if err = tx.Commit(); err != nil {
			log.Printf("Не удалось зафиксировать транзакцию: %s", err)
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("Не удалось зафиксировать транзакцию: %s.", err))
		}

	} else {

		// Создание нового таймера
		err = db.QueryRowContext(ctx, `
        INSERT INTO user_timers (user_id, start_time, description, is_active)
        VALUES ($1, NOW(), $2, TRUE) RETURNING start_time, end_time,id
    	`, userId, req.Description).Scan(&startTime, &endTime, &timerId)

		if err != nil {
			log.Printf("Ошибка при создании нового таймера: %s", err)
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка при создании нового таймера: %s.", err))

		}
	}

	// Преобразование времени в строку в формате ISO 8601 (UTC)
	var startTimeStr, endTimeStr string

	if startTime.Valid {
		startTimeStr = startTime.Time.UTC().Format(time.RFC3339)
	}

	if endTime.Valid {
		endTimeStr = endTime.Time.UTC().Format(time.RFC3339)
	}

	res := dbtimer.StartEndTimerResponseDB{
		StartTime: startTimeStr,
		EndTime:   endTimeStr,
		TimerId:   timerId,
		Message:   fmt.Sprintf("Таймер запушен"), // Сообщение об ошибке.
	}

	return &res, nil
}

func (s *TimerServiceServer) GetWorkingTimerDB(ctx context.Context, req *dbtimer.WorkingTimerRequestDB) (*dbtimer.WorkingTimerResponseDB, error) {

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
	// Открываем соединение с базой данных Авторизации
	dsn := utils.DsnString(database)
	// Получаем соединение с базой данных
	db, err := s.connectionsMap.GetDb(dsn)
	if err != nil {
		// Если произошла ошибка подключения, логируем её и возвращаем ответ с ошибкой.
		log.Printf("Ошибка подключения к базе данных: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка подключения к базе данных: %s.", err))
	}

	// SQL-запрос для проверки существования активного таймера.
	query := `
        SELECT 1
        FROM user_timers
        WHERE user_id = $1 AND is_active = TRUE RETURNING start_time, end_time,id
    `

	// Переменные для хранения значений start_time и end_time
	var startTime, endTime sql.NullTime
	var timerId uint64

	// Выполняем запрос с параметром user_id и получением открытого таймера exists.
	err = db.QueryRowContext(ctx, query, userId).Scan(&startTime, endTime, timerId)

	if err != nil {
		log.Printf("Ошибка при закрытии старого таймера: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка при закрытии старого таймера: %s.", err))
	}

	// Преобразование времени в строку в формате ISO 8601 (UTC)
	var startTimeStr, endTimeStr string

	if startTime.Valid {
		startTimeStr = startTime.Time.UTC().Format(time.RFC3339)
	}

	if endTime.Valid {
		endTimeStr = endTime.Time.UTC().Format(time.RFC3339)
	}

	res := dbtimer.WorkingTimerResponseDB{
		StartTime: startTimeStr,
		EndTime:   endTimeStr,
		TimerId:   timerId,
		Message:   fmt.Sprintf("Найден незавершённый таймер"), // Сообщение об ошибке.
	}

	return &res, nil
}

func (s *TimerServiceServer) EndTimerDB(ctx context.Context, req *dbtimer.StartEndTimerRequestDB) (*dbtimer.StartEndTimerResponseDB, error) {

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

	// Открываем соединение с базой данных Авторизации
	dsn := utils.DsnString(database)
	// Получаем соединение с базой данных
	db, err := s.connectionsMap.GetDb(dsn)
	if err != nil {
		// Если произошла ошибка подключения, логируем её и возвращаем ответ с ошибкой.
		log.Printf("Ошибка подключения к базе данных: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка подключения к базе данных: %s.", err))
	}

	// Переменные для хранения значений start_time и end_time
	var startTime, endTime sql.NullTime
	var timerId uint64

	// Закрытие старого таймера
	err = db.QueryRowContext(ctx, `
        UPDATE user_timers
        SET end_time = NOW(), is_active = FALSE
        WHERE user_id = $1 AND is_active = TRUE RETURNING start_time, end_time,id
    	`, userId).Scan(&startTime, &endTime, &timerId)

	if err != nil {
		log.Printf("Ошибка при закрытии старого таймера: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка при закрытии старого таймера: %s.", err))
	}

	// Преобразование времени в строку в формате ISO 8601 (UTC)
	var startTimeStr, endTimeStr string

	if startTime.Valid {
		startTimeStr = startTime.Time.UTC().Format(time.RFC3339)
	}

	if endTime.Valid {
		endTimeStr = endTime.Time.UTC().Format(time.RFC3339)
	}

	res := dbtimer.StartEndTimerResponseDB{
		StartTime: startTimeStr,
		EndTime:   endTimeStr,
		TimerId:   timerId,
		Message:   fmt.Sprintf("Таймер завершён"), // Сообщение об ошибке.
	}

	return &res, nil
}

func (s *TimerServiceServer) AddTimerDB(ctx context.Context, req *dbtimer.AddTimerRequestDB) (*dbtimer.AddTimerResponseDB, error) {

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

	// Открываем соединение с базой данных Авторизации
	dsn := utils.DsnString(database)
	// Получаем соединение с базой данных
	db, err := s.connectionsMap.GetDb(dsn)
	if err != nil {
		// Если произошла ошибка подключения, логируем её и возвращаем ответ с ошибкой.
		log.Printf("Ошибка подключения к базе данных: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка подключения к базе данных: %s.", err))
	}

	// Переменные для хранения значений start_time и end_time
	var startTime, endTime sql.NullTime
	var duration, description string
	var timerId uint64

	// Создание нового таймера
	err = db.QueryRowContext(ctx, `
        INSERT INTO user_timers (user_id, start_time,end_time_time, description)
        VALUES ($1, NOW(), $2) RETURNING start_time, end_time,id, duration,description
    	`, userId, &req.StartTime, &req.EndTime, &req.Description).Scan(&startTime, &endTime, &timerId, &duration, &description)

	if err != nil {
		log.Printf("Ошибка при создании нового таймера: %s", err)
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка при создании нового таймера: %s.", err))
	}

	// Преобразование времени в строку в формате ISO 8601 (UTC)
	var startTimeStr, endTimeStr string

	if startTime.Valid {
		startTimeStr = startTime.Time.UTC().Format(time.RFC3339)
	}

	if endTime.Valid {
		endTimeStr = endTime.Time.UTC().Format(time.RFC3339)
	}

	res := dbtimer.AddTimerResponseDB{
		StartTime:   startTimeStr,
		EndTime:     endTimeStr,
		TimerId:     timerId,
		Duration:    duration,
		Description: description,
		Message:     fmt.Sprintf("Таймер добавлен"), // Сообщение об ошибке.
	}

	return &res, nil

}

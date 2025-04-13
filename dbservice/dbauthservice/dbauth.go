package dbauthservice

import (
	"context"
	"crmSystem/migrations"
	"crmSystem/proto/dbauth"
	"crmSystem/proto/logs"
	"crmSystem/proto/redis"
	"crmSystem/utils"
	"database/sql"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type AuthServiceServer struct {
	dbauth.UnsafeDbAuthServiceServer
	connectionsMap *utils.MapConnectionsDB // Используем указатель
}

func NewGRPCDBAuthService(mapConnect *utils.MapConnectionsDB) *AuthServiceServer {
	return &AuthServiceServer{
		connectionsMap: mapConnect,
	}
}

// LoginDB обрабатывает запрос на вход пользователя в систему.
func (s *AuthServiceServer) LoginDB(ctx context.Context, req *dbauth.LoginDBRequest) (*dbauth.LoginDBResponse, error) {

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

	// Проверяем пользователя, используя функцию checkUser.
	dbName, userId, companyId, err := checkUser(s, req, token, ctx, clientLogs)
	if err != nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
		if errLogs != nil {
			log.Printf("Внутренняя ошибка проверки пользователя: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	//Проверяем найдена ли база данных для данного пользователя
	if dbName == "" {
		// Если база данных не найдена, формируем ответ с сообщением об ошибке.
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", "Ошибка нахождения базы данных")
		if errLogs != nil {
			log.Printf("Ошибка нахождения базы данных: %v", err)
		}
		return nil, status.Errorf(codes.NotFound, "Ошибка нахождения базы данных: %v", err)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "Не удалось получить метаданные из контекста auth_serivce")
	}

	// Создаем метаданные с Database и CompanyId
	md = metadata.Pairs(
		"database", dbName,
		"user-id", userId,
		"company-id", companyId,
	)

	// Добавляем метаданные в контекст
	err = grpc.SendHeader(ctx, md)
	if err != nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, dbName, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка установки метаданных: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "Ошибка установки метаданных: %v", err)
	}

	// Формируем успешный ответ, если пользователь найден.
	response := &dbauth.LoginDBResponse{
		Message: "Пользователь найден", // Сообщение об успешном входе.
	}

	return response, nil // Возвращаем успешный ответ.
}

// checkUser проверяет пользователя в базе данных авторизации и возвращает имя базы данных компании.
func checkUser(server *AuthServiceServer, req *dbauth.LoginDBRequest, token string,
	ctx context.Context, clientLogs logs.LogsServiceClient) (dbName string, userId string, companyID string, err error) {
	// Приведение данных к нижнему регистру
	emailLower := strings.ToLower(req.Email)
	phoneLower := strings.ToLower(req.Phone)
	password := req.Password // Пароль оставляем без изменений

	// Формируем строку соединения с базой данных авторизации.
	dsn := utils.DsnString(os.Getenv("DB_AUTH_NAME"))
	// Получаем соединение с базой данных.
	db, err := server.connectionsMap.GetDb(dsn)
	if err != nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, dbName, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка при получении соединения из connectionsMap: %v", err)
		}
		log.Printf("Ошибка при получении соединения из connectionsMap")
		return "", "", "", err
	}

	// Устанавливаем соединение с gRPC сервером Redis
	client, err, connRedis := utils.RedisServiceConnector(token)
	if err != nil {
		fmt.Printf("Ошибка подключения к Redis: " + err.Error())
		return "", "", "", err
	}
	defer func(connRedis *grpc.ClientConn) {
		err := connRedis.Close()
		if err != nil {
			errLogs := utils.SaveLogsError(ctx, clientLogs, dbName, userId, err.Error())
			if errLogs != nil {
				log.Printf("Ошибка закрытия соединения c Redis: %v", err)
			}
		}
	}(connRedis)

	// Формируем запрос для Redis с использованием email в нижнем регистре
	req1 := &redis.GetRedisRequest{
		Key: emailLower + "Login" + userId,
	}

	resRedis, err := client.Get(ctx, req1)
	if err != nil {
		log.Printf("Ошибка подключения базы данных: %s", err)
		errLogs := utils.SaveLogsError(ctx, clientLogs, dbName, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка подключения базы данных: %v", err)
		}
		return "", "", "", err
	}

	type DbName struct {
		Database  string
		UserId    string
		CompanyId string
	}

	if resRedis.Status == http.StatusOK {
		convertedRedis, err := utils.ConvertJSONToStruct[DbName](resRedis.Message)
		if err != nil {
			errLogs := utils.SaveLogsError(ctx, clientLogs, dbName, userId, err.Error())
			if errLogs != nil {
				log.Printf("Ошибка ConvertJSONToStruct convertedRedis: %v", err)
			}
			return "", "", "", err
		}
		return convertedRedis.Database, convertedRedis.UserId, convertedRedis.CompanyId, nil
	}

	// SQL-запрос для проверки пользователя с данными в нижнем регистре
	query := `
        SELECT id, company_id
        FROM authusers 
        WHERE (email = $1 OR phone = $2) 
        AND password = $3
    `

	companyID = ""
	authUserId := ""

	// Используем email и phone в нижнем регистре
	err = db.QueryRow(query, emailLower, phoneLower, password).Scan(&authUserId, &companyID)
	if err != nil {
		if err == sql.ErrNoRows {
			errLogs := utils.SaveLogsError(ctx, clientLogs, dbName, userId, err.Error())
			if errLogs != nil {
				log.Printf("Пользователь не найден: %v", err)
			}
			return "", "", "", fmt.Errorf("Пользователь не найден")
		}
		errLogs := utils.SaveLogsError(ctx, clientLogs, dbName, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка при выполнении запроса: %v", err)
		}
		return "", "", "", err
	}

	// SQL-запрос для получения имени базы данных компании
	queryCompanies := `
        SELECT dbName
        FROM companies 
        WHERE id = $1 
    `

	err = db.QueryRow(queryCompanies, companyID).Scan(&dbName)
	if err != nil {
		if err == sql.ErrNoRows {
			errLogs := utils.SaveLogsError(ctx, clientLogs, dbName, userId, err.Error())
			if errLogs != nil {
				log.Printf("Запись о базе данных не найдена: %v", err)
			}
			return "", "", "", fmt.Errorf("запись о базе данных не найдена")
		}
		return "", "", "", err
	}

	// Работа с базой данных компании
	dsnC := utils.DsnString(dbName)
	dbConnCompany, err := server.connectionsMap.GetDb(dsnC)
	if dbConnCompany == nil {
		log.Println("Ошибка: соединение с базой данных компании не инициализировано")
		errLogs := utils.SaveLogsError(ctx, clientLogs, dbName, userId, "Ошибка: соединение с базой данных компании не инициализировано")
		if errLogs != nil {
			log.Printf("Ошибка: соединение с базой данных компании не инициализировано: %v", err)
		}
		return "", "", "", fmt.Errorf("соединение с базой данных компании не инициализировано")
	}

	// Получаем userId из таблицы users
	err = dbConnCompany.QueryRow("SELECT id FROM users WHERE authId = $1", authUserId).Scan(&userId)
	if err != nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, dbName, userId, err.Error())
		if errLogs != nil {
			log.Printf("Не удалось найти пользователя в базе данных компании: %v", err)
		}
		return "", "", "", fmt.Errorf("не удалось найти пользователя в базе данных компании: %v", err)
	}

	// Сохраняем данные в Redis
	toJsonType := &DbName{
		Database:  dbName,
		UserId:    userId,
		CompanyId: companyID,
	}

	toJsonRedis, err := utils.ConvertStructToJSON(toJsonType)
	if err != nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, dbName, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка преобразования в JSON: %v", err)
		}
	}

	saveRequest := &redis.SaveRedisRequest{
		Key:        emailLower + "Login" + userId, // Используем email в нижнем регистре
		Value:      toJsonRedis,
		Expiration: int64((time.Minute * 10).Seconds()),
	}

	_, err = client.Save(ctx, saveRequest)
	if err != nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, dbName, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка выполнения gRPC вызова Save: %v", err)
		}
		fmt.Printf("Ошибка выполнения gRPC вызова Save")
	}

	return dbName, userId, companyID, nil
}

func (s *AuthServiceServer) RegisterCompany(ctx context.Context, req *dbauth.RegisterCompanyRequest) (*dbauth.RegisterCompanyResponse, error) {

	token, err := utils.ExtractTokenFromContext(ctx)
	if err != nil {
		log.Printf("Не удалось извлечь токен для логирования: %v", err)
		return nil, status.Errorf(codes.Unauthenticated, "Не удалось извлечь токен для логирования")
	}

	if len(token) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Токен не найден в метаданных")
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

	// Вызываем функцию registerCompany для создания базы данных и регистрации компании.
	dbName, companyId, userId, statusRegister, err := registerCompany(s, req, token)
	if err != nil {
		// Если произошла ошибка, формируем ответ с сообщением об ошибке.
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
		if errLogs != nil {
			log.Printf("%v", errLogs)
		}
		return nil, status.Errorf(statusRegister, fmt.Sprintf("%v", err))
	}

	// Создаем метаданные с Database и CompanyId
	md := metadata.Pairs(
		"database", dbName,
		"user-id", userId,
		"company-id", companyId,
	)

	// Добавляем метаданные в контекст
	err = grpc.SendHeader(ctx, md)
	if err != nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, dbName, userId, err.Error())
		if errLogs != nil {
			log.Printf("%v", err)
		}
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка установки метаданных: %v", err))
	}

	// Формируем успешный ответ, если регистрация прошла успешно.
	response := &dbauth.RegisterCompanyResponse{
		Message: "Регистрация успешна", // Сообщение об успешной регистрации.
	}

	return response, nil // Возвращаем успешный ответ.
}

// registerCompany регистрирует новую компанию в базе данных и создает
// соответствующего пользователя в системе авторизации.
//
// Параметры:
// - server: Указатель на экземпляр MapConnectionsDB, который используется для управления соединениями с базами данных.
// - req: Указатель на структуру dbauth.RegisterCompanyRequest, содержащую данные о компании и пользователе (имя компании, адрес, email, телефон и пароль).
//
// Возвращает:
// - nameDB: Имя созданной базы данных для компании.
// - Ошибка, если произошла ошибка при создании базы данных, подключении к ней, выполнении миграций или операциях с базой данных.
// В противном случае возвращает nil и код состояния http.StatusOK.
//
// Процесс выполнения:
// 1. Получает имя базы данных авторизации из переменных окружения.
// 2. Создает новую базу данных для компании с помощью функции createClientDatabase.
// 3. Подключается к базе данных авторизации и проверяет состояние соединения.
// 4. Начинает транзакцию для базы данных авторизации.
// 5. Проверяет, существует ли уже компания с указанным именем и адресом:
//   - Если существует, возвращает ошибку с кодом http.StatusConflict.
//   - Если не существует, создает новую запись о компании.
//
// 6. Создает нового пользователя в таблице authusers и сохраняет его ID.
// 7. Фиксирует транзакцию для базы данных авторизации.
// 8. Подключается к только что созданной базе данных компании и начинает новую транзакцию.
// 9. Создает записи о правах и пользователе в базе данных компании.
// 10. Фиксирует транзакцию для базы данных компании.
// 11. Возвращает имя базы данных для компании и nil, если все операции выполнены успешно.
// registerCompany регистрирует новую компанию и создает пользователя в системе авторизации.
func registerCompany(server *AuthServiceServer, req *dbauth.RegisterCompanyRequest, token string) (
	nameDB string, userId string, companyId string, status codes.Code, err error) {

	// Приведение данных из запроса к нижнему регистру
	nameCompanyLower := strings.ToLower(req.NameCompany)
	addressLower := strings.ToLower(req.Address)
	emailLower := strings.ToLower(req.Email)
	phoneLower := strings.ToLower(req.Phone)
	password := req.Password // Пароль оставляем без изменений, если не требуется

	// В случае превышения порога ожидания с сервера в 10 секунд будет ошибка контекста.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Проверка базы данных в Redis
	client, err, connRedis := utils.RedisServiceConnector(token)
	if err != nil {
		fmt.Printf("Ошибка подключения к Redis: " + err.Error())
		return "", "", "", codes.Internal, err
	} else {
		defer func(connRedis *grpc.ClientConn) {
			err := connRedis.Close()
			if err != nil {
				log.Printf(err.Error())
			}
		}(connRedis)
	}

	// Формируем запрос на регистрацию компании
	req1 := &redis.GetRedisRequest{
		Key: nameCompanyLower + "Register", // Используем нижний регистр для ключа
	}

	// Выполняем gRPC вызов Get
	resRedis, err := client.Get(ctx, req1)

	type dbRedisType struct {
		Message   string
		Database  string
		CompanyId string
		UserId    string
	}

	if resRedis.Status == http.StatusOK {
		convertedRedis, err := utils.ConvertJSONToStruct[dbRedisType](resRedis.Message)
		if err != nil {
			return "", "", "", codes.Internal, err
		}
		return convertedRedis.Database, convertedRedis.UserId, convertedRedis.CompanyId, codes.OK, nil
	} else {
		authDBName := os.Getenv("DB_AUTH_NAME")
		newDbName := utils.RandomDBName(25)
		dsn := utils.DsnString(authDBName)

		dbConn, err := server.connectionsMap.GetDb(dsn)
		if err != nil {
			return "", "", "", codes.Internal, err
		}

		if dbConn == nil {
			log.Println("Ошибка: соединение с базой данных авторизации не инициализировано")
			return "", "", "", codes.Internal, fmt.Errorf("соединение с базой данных авторизации не инициализировано")
		}

		tx, err := dbConn.Begin()
		if err != nil {
			return "", "", "", codes.Internal, fmt.Errorf("не удалось начать транзакцию: %v", err)
		}

		defer func() {
			if err != nil {
				_ = tx.Rollback()
				log.Printf("Транзакция откатана (auth DB) из-за ошибки: %v", err)
			}
		}()

		// Проверяем, существует ли компания с именем и адресом в нижнем регистре
		query := "SELECT id FROM companies WHERE name = $1 AND address = $2"
		err = tx.QueryRow(query, nameCompanyLower, addressLower).Scan(&companyId)
		if err != nil {
			if err == sql.ErrNoRows {
				// Вставляем новую компанию с данными в нижнем регистре
				err = tx.QueryRow(
					"INSERT INTO companies (name, address, dbname) VALUES ($1, $2, $3) RETURNING id",
					nameCompanyLower, addressLower, newDbName,
				).Scan(&companyId)
				if err != nil {
					return "", "", "", codes.Internal, fmt.Errorf("не удалось создать компанию: %v", err)
				}
			} else {
				return "", "", "", codes.InvalidArgument, fmt.Errorf("ошибка при проверке существования компании: %v", err)
			}
		} else {
			return "", "", "", codes.AlreadyExists, fmt.Errorf("компания с таким именем и адресом уже существует: %s", nameCompanyLower)
		}

		var authUserId string
		// Вставляем пользователя с данными в нижнем регистре
		err = tx.QueryRow(
			"INSERT INTO authusers (email, phone, password, company_id) VALUES ($1, $2, $3, $4) RETURNING id",
			emailLower, phoneLower, password, companyId,
		).Scan(&authUserId)
		if err != nil {
			if strings.Contains(err.Error(), "authusers_phone_key") {
				return "", "", "", codes.AlreadyExists, fmt.Errorf("дубликат номера телефона: %v", err)
			}
			if strings.Contains(err.Error(), "authusers_email_key") {
				return "", "", "", codes.AlreadyExists, fmt.Errorf("дубликат почты: %v", err)
			}
			return "", "", "", codes.Internal, fmt.Errorf("не удалось создать пользователя: %v", err)
		}

		err = tx.Commit()
		if err != nil {
			return "", "", "", codes.Internal, fmt.Errorf("не удалось зафиксировать транзакцию auth DB: %v", err)
		}

		err = createClientDatabase(newDbName, server, ctx)
		if err != nil {
			return "", "", "", codes.Internal, err
		}

		dsnC := utils.DsnString(newDbName)
		dbConnCompany, err := server.connectionsMap.GetDb(dsnC)
		if dbConnCompany == nil {
			log.Println("Ошибка: соединение с базой данных компании не инициализировано")
			return "", "", "", codes.Internal, fmt.Errorf("соединение с базой данных компании не инициализировано")
		}

		txc, err := dbConnCompany.Begin()
		if err != nil {
			return "", "", "", codes.Internal, fmt.Errorf("не удалось начать транзакцию для компании: %v", err)
		}

		defer func() {
			if err != nil {
				_ = txc.Rollback()
				log.Printf("Транзакция откатана (company DB) из-за ошибки: %v", err)
				err = rollbackAuthDB(dbConn, companyId, authUserId, ctx)
				log.Printf("Не удалось откатить транзакцию: %v", err)
			}
		}()

		role := os.Getenv("FIRST_ROLE")
		var roleID int
		err = txc.QueryRow("INSERT INTO rights (roles) VALUES ($1) RETURNING id", role).Scan(&roleID)
		if err != nil {
			return "", "", "", codes.Unimplemented, fmt.Errorf("не удалось добавить название прав: %v", err)
		}

		var newUserId string
		err = txc.QueryRow(
			"INSERT INTO users (rightsId, authId) VALUES ($1, $2) RETURNING id",
			roleID, authUserId,
		).Scan(&newUserId)
		if err != nil {
			return "", "", "", codes.Unimplemented, fmt.Errorf("не удалось добавить пользователя: %v", err)
		}

		_, err = txc.Exec(
			"INSERT INTO availableactions (roleId, createTasks, createChats, addWorkers) VALUES ($1, $2, $3, $4)",
			roleID, true, true, true,
		)
		if err != nil {
			return "", "", "", codes.Unimplemented, fmt.Errorf("не удалось добавить доступные действия для роли: %v", err)
		}

		err = txc.Commit()
		if err != nil {
			return "", "", "", codes.Unimplemented, fmt.Errorf("не удалось зафиксировать транзакцию компании: %v", err)
		}

		toJsonType := &dbRedisType{
			Message:   nameCompanyLower, // Используем нижний регистр
			Database:  newDbName,
			UserId:    newUserId,
			CompanyId: companyId,
		}

		toJsonRedis, err := utils.ConvertStructToJSON(toJsonType)
		if err != nil {
			fmt.Printf(err.Error())
		}

		saveRequest := &redis.SaveRedisRequest{
			Key:        nameCompanyLower + "Register" + newUserId, // Используем нижний регистр для ключа
			Value:      toJsonRedis,
			Expiration: int64((time.Minute * 10).Seconds()),
		}

		_, err = client.Save(ctx, saveRequest)
		if err != nil {
			fmt.Printf(err.Error())
		}

		return newDbName, newUserId, companyId, codes.OK, nil
	}
}

// createClientDatabase создает новую базу данных с рандомным именем для компании
// и выполняет необходимые миграции для таблицы users.
//
// Параметры:
// - server: Указатель на экземпляр MapConnectionsDB, который используется для управления соединениями с базами данных.
//
// Возвращает:
// - nameDB: Имя созданной базы данных.
// - Ошибка, если произошла ошибка при создании базы данных, подключении к ней или выполнении миграций.
// В противном случае возвращает nil.
//
// Процесс выполнения:
// 1. Генерирует рандомное имя для базы данных с помощью функции utils.RandomDBName.
// 2. Проверяет и создаёт базу данных с помощью функции createInsideDB.
// 3. Подключается к только что созданной базе данных с помощью функции GetDb.
// 4. Выполняет миграцию для таблицы users, используя указанный путь к миграциям (MIGRATION_COMPANYDB_PATH).
// 5. Возвращает имя созданной базы данных и nil, если все операции выполнены успешно.
func createClientDatabase(dbName string, server *AuthServiceServer, _ context.Context) (err error) {

	// Функция проверки и создания базы данных
	err = utils.CreateInsideDB(dbName)
	if err != nil {
		return fmt.Errorf("Ошибка при создании базы данных: %w", err)
	}

	// Теперь подключаемся к только что созданной базе данных
	newDSN := utils.DsnString(dbName) // Создаем новое соединение к этой базе
	newDB, err := server.connectionsMap.GetDb(newDSN)

	//Проверка соединения
	if err != nil {
		return fmt.Errorf("Ошибка подключения к новой базе данных: %w", err)
	}
	/*defer func(newDB *sql.DB) {
		err := newDB.Close()
		if err != nil {
			log.Fatal("Некорректное закрытие базы данных")
		}
	}(newDB)*/

	// Миграция для таблицы users
	migratePath := os.Getenv("MIGRATION_COMPANYDB_PATH")
	err = migrations.Migration(newDB, migratePath, dbName)
	if err != nil {
		return fmt.Errorf("Ошибка при миграции базы данных: %w", err)
	}

	return nil
}

// rollbackAuthDB откатывает изменения в базе данных авторизации, удаляя пользователя и компанию.
func rollbackAuthDB(dbConn *sql.DB, companyId, authUserId string, _ context.Context) error {
	// Начинаем откатную транзакцию.
	tx, err := dbConn.Begin()
	if err != nil {
		log.Printf("Ошибка при начале откатной транзакции: %v", err) // Логируем ошибку, если не удалось начать транзакцию.
		return err                                                   // Завершаем выполнение функции.
	}

	// Удаляем пользователя из таблицы authusers по его ID.
	_, err = tx.Exec("DELETE FROM authusers WHERE id = $1", authUserId)
	if err != nil {
		err := tx.Rollback()
		if err != nil {
			return err
		} // Откатываем транзакцию в случае ошибки.
		log.Printf("Ошибка при удалении пользователя: %v", err) // Логируем ошибку.
		return err                                              // Завершаем выполнение функции.
	}

	// Удаляем компанию из таблицы companies по её ID.
	_, err = tx.Exec("DELETE FROM companies WHERE id = $1", companyId)
	if err != nil {
		err := tx.Rollback()
		if err != nil {
			return err
		} // Откатываем транзакцию в случае ошибки.
		log.Printf("Ошибка при удалении компании: %v", err) // Логируем ошибку.
		return err                                          // Завершаем выполнение функции.
	}

	// Фиксируем откат транзакции.
	err = tx.Commit()
	if err != nil {
		log.Printf("Ошибка при фиксации отката: %v", err) // Логируем ошибку, если фиксация не удалась.
	}
	return nil
}

package dbauthservice

import (
	"context"
	"crmSystem/migrations"
	"crmSystem/proto/dbauth"
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

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "Не удалось получить метаданные из контекста auth_serivce")
	}

	// Извлекаем токен из метаданных
	token := md["auth-token"][0] // токен передается как "auth-token"
	if len(md["auth-token"]) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Токен не найден в метаданных")
	}

	// Проверяем пользователя, используя функцию checkUser.
	dbName, userId, companyId, err := checkUser(s, req, token, ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Внутренняя ошибка: %v", err)
	}

	//Проверяем найдена ли база данных для данного пользователя
	if dbName == "" {
		// Если база данных не найдена, формируем ответ с сообщением об ошибке.
		return nil, status.Errorf(codes.NotFound, "Ошибка нахождения базы данных: %v", err)
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
		return nil, status.Errorf(codes.Internal, "Ошибка установки метаданных: %v", err)
	}

	// Формируем успешный ответ, если пользователь найден.
	response := &dbauth.LoginDBResponse{
		Message: "Пользователь найден", // Сообщение об успешном входе.
	}

	return response, nil // Возвращаем успешный ответ.
}

// checkUser проверяет пользователя в базе данных авторизации и возвращает имя базы данных компании.
func checkUser(server *AuthServiceServer, req *dbauth.LoginDBRequest, token string, ctx context.Context) (dbName string, userId string, companyID string, err error) {
	// Формируем строку соединения с базой данных авторизации.
	dsn := utils.DsnString(os.Getenv("DB_AUTH_NAME"))
	// Получаем соединение с базой данных.
	db, err := server.connectionsMap.GetDb(dsn)

	//Проверка базы данных в редис
	//Устанавливаем соединение с gRPC сервером Redis

	//token, err := utils.GetTokenFromMetadata(ctx)

	//Проверка ошибки при получении
	if err != nil {
		log.Printf(err.Error())
	}

	client, err, connRedis := utils.RedisServiceConnector(token)

	if err != nil {
		fmt.Printf("Ошибка Подключение к redis : " + err.Error())
		return "", "", "", err
	}

	defer connRedis.Close()

	// Формируем запрос на регистрацию компании
	req1 := &redis.GetRedisRequest{
		Key: req.Email + "Login" + userId,
		// Выполняем gRPC вызов RegisterCompany
		//Создаём тип для Получения базы данных
		//Получаем строку из редис и с помощью универсальной функции.
		// Преобразуем её к переданному типу который возвращаем как ответ
	}

	resRedis, err := client.Get(ctx, req1)
	if err != nil {
		log.Printf("Ошибка подключения базы данных: %s", err) // Логируем ошибку подключения.
		return "", "", "", err                                // Возвращаем пустую строку и ошибку.
	}

	type DbName struct {
		Database  string
		UserId    string
		CompanyId string
	}

	if resRedis.Status == http.StatusOK {

		convertedRedis, err := utils.ConvertJSONToStruct[DbName](resRedis.Message)
		if err != nil {
			return "", "", "", err
		}

		return convertedRedis.Database, convertedRedis.UserId, convertedRedis.CompanyId, nil // Возвращаем успешный ответ.

	}

	// SQL-запрос для проверки существования пользователя по email или телефону и паролю.
	query := `
        SELECT id, company_id
        FROM authusers 
        WHERE (email = $1 OR phone = $2) 
        AND password = $3
    `

	// Переменная для хранения ID компании, к которой принадлежит пользователь.
	companyID = ""
	//Id пользователя в авторизационной базе данных
	authUserId := ""

	// Выполняем запрос и сканируем результат в переменную companyID.
	err = db.QueryRow(query, req.Email, req.Phone, req.Password).Scan(&authUserId, &companyID)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", "", fmt.Errorf("Пользователь не найден") // Пользователь не найден, возвращаем ошибку.
		}
		return "", "", "", err // Возвращаем ошибку при выполнении запроса.
	}

	// SQL-запрос для получения имени базы данных компании по ее ID.
	queryCompanies := `
        SELECT dbName
        FROM companies 
        WHERE id = $1 
    `

	// Выполняем запрос и сканируем результат в переменную dbName.
	err = db.QueryRow(queryCompanies, companyID).Scan(&dbName)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", "", fmt.Errorf("запись о базе данных не найден") // Если компании не найдены, возвращаем пустую строку.
		}
		return "", "", "", err // Возвращаем ошибку при выполнении запроса.
	}

	// Работа с базой данных компании.

	dsnC := utils.DsnString(dbName)                         // Формируем строку подключения к базе данных компании.
	dbConnCompany, err := server.connectionsMap.GetDb(dsnC) // Получаем соединение с базой данных компании.

	if dbConnCompany == nil {
		log.Println("Ошибка: соединение с базой данных компании не инициализировано")
		return "", "", "", fmt.Errorf("соединение с базой данных компании не инициализировано") // Возвращаем ошибку, если соединение не удалось.
	}

	// Вставляем нового пользователя в таблицу users.
	/*err = txc.QueryRow(
		"SELECT id FROM users WHERE authId = "
		"INSERT INTO users (roles, companyId, rightsId, authId) VALUES ($1, $2, $3, $4) RETURNING id",
		role, companyId, roleID, authUserId,
	).Scan(&newAdminId)*/

	// Выполняем запрос и сканируем результат в переменную dbName.
	err = dbConnCompany.QueryRow("SELECT id FROM users WHERE authId = $1", authUserId).Scan(&userId)

	if err != nil {
		return "", "", "", fmt.Errorf("не удалось найти пользователя в базе данных компании: %v", err) // Возвращаем ошибку, если вставка не удалась.
	}

	toJsonType := &DbName{
		Database:  dbName,
		UserId:    userId,
		CompanyId: companyID,
	}

	toJsonRedis, err := utils.ConvertStructToJSON(toJsonType)

	//Создаём ключ, значение, и время истечения для сохранения готового запроса
	saveRequest := &redis.SaveRedisRequest{
		Key:        req.Email + "Login" + userId,
		Value:      toJsonRedis,
		Expiration: int64((time.Minute * 10).Seconds()),
	}

	// Выполняем gRPC вызов RegisterCompany
	_, err = client.Save(ctx, saveRequest)

	if err != nil {
		fmt.Printf(err.Error())
	}

	return dbName, userId, companyID, nil // Возвращаем имя базы данных и nil в качестве ошибки, если все прошло успешно.
}

func (s *AuthServiceServer) RegisterCompany(ctx context.Context, req *dbauth.RegisterCompanyRequest) (*dbauth.RegisterCompanyResponse, error) {

	/* Логируем получение запроса на регистрацию компании с именем из запроса.
	log.Printf("Получен запрос на регистрацию организации: %v", req.NameCompany)*/

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "Не удалось получить метаданные из контекста")
	}

	// Извлекаем токен из метаданных
	token := md["auth-token"] // предполагаем, что токен передается как "auth-token"

	if len(token) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "Токен не найден в метаданных")
	}

	// Вызываем функцию registerCompany для создания базы данных и регистрации компании.
	dbName, companyId, userId, statusRegister, err := registerCompany(s, req, token[0])
	if err != nil {
		// Если произошла ошибка, формируем ответ с сообщением об ошибке.
		return nil, status.Errorf(statusRegister, fmt.Sprintf("Ошибка вызова регистрации компании: %v", err))
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

	// В случае превышения порога ожидания с сервера в 10 секунд будет ошибка контекста.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	//Проверка базы данных в редис
	//Соединение с gRPC сервером Redis
	client, err, connRedis := utils.RedisServiceConnector(token)
	if err != nil {
		fmt.Printf("Ошибка Подключение к redis : " + err.Error())
		return "", "", "", codes.Internal, err
	} else {
		defer connRedis.Close()
	}

	// Формируем запрос на регистрацию компании
	req1 := &redis.GetRedisRequest{
		Key: req.NameCompany + "Register",
	}

	// Выполняем gRPC вызов RegisterCompany
	resRedis, err := client.Get(ctx, req1)

	type dbRedisType struct {
		Message   string
		Database  string
		CompanyId string
		UserId    string
		Status    uint32
	}

	if resRedis.Status == http.StatusOK {
		//Получаем строку из редиса и с помощью универсальной функции.
		// Преобразуем её к переданному типу который возвращаем как ответ
		convertedRedis, err := utils.ConvertJSONToStruct[dbRedisType](resRedis.Message)
		if err != nil {
			return "", "", "", codes.Internal, err
		}

		return convertedRedis.Database, convertedRedis.UserId, convertedRedis.CompanyId, codes.OK, nil // Возвращаем успешный ответ.

	} else {

		// Получаем имя базы данных авторизации из переменных окружения.
		authDBName := os.Getenv("DB_AUTH_NAME")

		// Создаём рандомное имя для базы данных компании
		newDbName := utils.RandomDBName(25)

		// Формируем строку подключения к базе данных авторизации.
		dsn := utils.DsnString(authDBName)

		// Получаем соединение с базой данных авторизации.
		dbConn, err := server.connectionsMap.GetDb(dsn)
		if err != nil {
			return "", "", "", codes.Internal, err // Возвращаем ошибку, если соединение не удалось.
		}

		// Логируем состояние соединения с базой данных авторизации.
		if dbConn == nil {
			log.Println("Ошибка: соединение с базой данных авторизации не инициализировано")
			return "", "", "", codes.Internal, fmt.Errorf("соединение с базой данных авторизации не инициализировано")
		}

		// Начинаем транзакцию для базы данных авторизации.
		tx, err := dbConn.Begin()
		if err != nil {
			return "", "", "", codes.Internal, fmt.Errorf("не удалось начать транзакцию: %v", err) // Возвращаем ошибку, если не удалось начать транзакцию.
		}

		defer func() { // Отложенная функция для отката транзакции в случае ошибки.
			if err != nil {
				_ = tx.Rollback()                                                 // Откатываем транзакцию.
				log.Printf("Транзакция откатана (auth DB) из-за ошибки: %v", err) // Логируем откат.
			}
		}()

		// Проверяем, существует ли уже компания с указанным именем и адресом.
		query := "SELECT id FROM companies WHERE name = $1 AND address = $2"
		err = tx.QueryRow(query, req.NameCompany, req.Address).Scan(&companyId)
		if err != nil {
			if err == sql.ErrNoRows { // Если компания не найдена, продолжаем вставку.
				// Вставляем новую компанию и получаем её ID.
				err = tx.QueryRow(
					"INSERT INTO companies (name, address, dbname) VALUES ($1, $2, $3) RETURNING id",
					req.NameCompany, req.Address, newDbName,
				).Scan(&companyId)
				if err != nil {
					return "", "", "", codes.Internal, fmt.Errorf("Не удалось создать компанию: %v", err) // Возвращаем ошибку, если вставка не удалась.
				}
			} else {
				return "", "", "", codes.InvalidArgument, fmt.Errorf("Ошибка при проверке существования компании: %v", err) // Возвращаем ошибку, если произошла другая ошибка.
			}
		} else {
			return "", "", "", codes.AlreadyExists, fmt.Errorf("Компания с таким именем и адресом уже существует: %s", req.NameCompany) // Возвращаем ошибку, если компания уже существует.
		}

		var authUserId string // Переменная для хранения ID пользователя.

		// Вставляем нового пользователя в таблицу authusers и получаем его ID.
		err = tx.QueryRow(
			"INSERT INTO authusers (email, phone, password, company_id) VALUES ($1, $2, $3, $4) RETURNING id",
			req.Email, req.Phone, req.Password, companyId,
		).Scan(&authUserId)
		if err != nil {
			return "", "", "", codes.Internal, fmt.Errorf("Не удалось создать пользователя: %v", err) // Возвращаем ошибку, если вставка не удалась.
		}

		err = tx.Commit() // Фиксируем транзакцию для базы данных авторизации.
		if err != nil {
			return "", "", "", codes.Internal, fmt.Errorf("Не удалось зафиксировать транзакцию auth DB: %v", err) // Возвращаем ошибку, если фиксация не удалась.
		}

		// Создаем базу данных для компании.
		err = createClientDatabase(newDbName, server)
		if err != nil {
			return "", "", "", codes.Internal, err // Возвращаем ошибку, если создание базы данных не удалось.
		}

		// Работа с базой данных компании.
		dsnC := utils.DsnString(newDbName)                      // Формируем строку подключения к базе данных компании.
		dbConnCompany, err := server.connectionsMap.GetDb(dsnC) // Получаем соединение с базой данных компании.

		if dbConnCompany == nil {
			log.Println("Ошибка: соединение с базой данных компании не инициализировано")
			return "", "", "", codes.Internal, fmt.Errorf("соединение с базой данных компании не инициализировано") // Возвращаем ошибку, если соединение не удалось.
		}

		txc, err := dbConnCompany.Begin() // Начинаем транзакцию для базы данных компании.
		if err != nil {
			return "", "", "", codes.Internal, fmt.Errorf("не удалось начать транзакцию для компании: %v", err) // Возвращаем ошибку, если не удалось начать транзакцию.
		}

		defer func() { // Отложенная функция для отката транзакции в случае ошибки.
			if err != nil {
				_ = txc.Rollback()                                                   // Откатываем транзакцию.
				log.Printf("Транзакция откатана (company DB) из-за ошибки: %v", err) // Логируем откат.
				// Откат действий в первой базе данных.
				rollbackAuthDB(dbConn, companyId, authUserId)
			}
		}()

		role := os.Getenv("FIRST_ROLE") // Получаем роль для нового пользователя.

		var roleID int // Переменная для хранения ID роли.

		// Вставляем новую роль в таблицу rights и получаем её ID.
		err = txc.QueryRow("INSERT INTO rights (roles) VALUES ($1) RETURNING id", role).Scan(&roleID)
		if err != nil {
			return "", "", "", codes.Unimplemented, fmt.Errorf("не удалось добавить название прав: %v", err) // Возвращаем ошибку, если вставка не удалась.
		}

		var newUserId string

		// Вставляем нового пользователя в таблицу users.
		err = txc.QueryRow(
			"INSERT INTO users (rightsId, authId) VALUES ($1, $2) RETURNING id",
			roleID, authUserId,
		).Scan(&newUserId)
		if err != nil {
			return "", "", "", codes.Unimplemented, fmt.Errorf("не удалось добавить пользователя: %v", err) // Возвращаем ошибку, если вставка не удалась.
		}

		// Вставляем доступные действия для новой роли.
		_, err = txc.Exec(
			"INSERT INTO availableactions (roleId, createTasks, createChats, addWorkers) VALUES ($1, $2, $3, $4)",
			roleID, true, true, true,
		)
		if err != nil {
			return "", "", "", codes.Unimplemented, fmt.Errorf("не удалось добавить доступные действия для роли: %v", err) // Возвращаем ошибку, если вставка не удалась.
		}

		err = txc.Commit() // Фиксируем транзакцию для базы данных компании.
		if err != nil {
			return "", "", "", codes.Unimplemented, fmt.Errorf("не удалось зафиксировать транзакцию компании: %v", err) // Возвращаем ошибку, если фиксация не удалась.
		}

		toJsonType := &dbRedisType{
			Message:   req.NameCompany,
			Database:  newDbName,
			UserId:    newUserId,
			CompanyId: companyId,
			Status:    http.StatusInternalServerError,
		}

		toJsonRedis, err := utils.ConvertStructToJSON(toJsonType)

		//Проверка на ошибку в преобразовании к JSON строке
		if err != nil {
			fmt.Printf(err.Error())
		}

		//Создаём ключ, значение, и время истечения для сохранения готового запроса
		saveRequest := &redis.SaveRedisRequest{
			Key:        req.NameCompany + "Register" + newUserId,
			Value:      toJsonRedis,
			Expiration: int64((time.Minute * 10).Seconds()),
		}

		// Выполняем gRPC вызов RegisterCompany
		_, err = client.Save(ctx, saveRequest)

		if err != nil {
			fmt.Printf(err.Error())
		}

		return newDbName, newUserId, companyId, codes.OK, nil // Возвращаем имя базы данных, nil и код состояния 200 OK, если все операции выполнены успешно.
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
func createClientDatabase(dbName string, server *AuthServiceServer) (err error) {

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
func rollbackAuthDB(dbConn *sql.DB, companyId, authUserId string) {
	// Начинаем откатную транзакцию.
	tx, err := dbConn.Begin()
	if err != nil {
		log.Printf("Ошибка при начале откатной транзакции: %v", err) // Логируем ошибку, если не удалось начать транзакцию.
		return                                                       // Завершаем выполнение функции.
	}

	// Удаляем пользователя из таблицы authusers по его ID.
	_, err = tx.Exec("DELETE FROM authusers WHERE id = $1", authUserId)
	if err != nil {
		tx.Rollback()                                           // Откатываем транзакцию в случае ошибки.
		log.Printf("Ошибка при удалении пользователя: %v", err) // Логируем ошибку.
		return                                                  // Завершаем выполнение функции.
	}

	// Удаляем компанию из таблицы companies по её ID.
	_, err = tx.Exec("DELETE FROM companies WHERE id = $1", companyId)
	if err != nil {
		tx.Rollback()                                       // Откатываем транзакцию в случае ошибки.
		log.Printf("Ошибка при удалении компании: %v", err) // Логируем ошибку.
		return                                              // Завершаем выполнение функции.
	}

	// Фиксируем откат транзакции.
	err = tx.Commit()
	if err != nil {
		log.Printf("Ошибка при фиксации отката: %v", err) // Логируем ошибку, если фиксация не удалась.
	}
}

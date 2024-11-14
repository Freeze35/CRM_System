package main

import (
	"context"
	"crmSystem/migrations"
	pb "crmSystem/proto/dbservice" // Импортируйте сгенерированный пакет из протобуферов
	"crmSystem/utils"
	"database/sql"
	"fmt"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type DbServiceServer struct {
	pb.UnimplementedDbServiceServer
	mapDB map[string]*sql.DB // Карта для хранения соединений с базами данных
}

// Конструктор для инициализации DbServiceServer
func NewDbServiceServer() *DbServiceServer {
	return &DbServiceServer{
		mapDB: make(map[string]*sql.DB),
	}
}

// GetDb проверяет существование открытого соединения с базой данных по имени dbName.
// Если соединение уже существует и активно, возвращает его.
// В противном случае создаёт новое соединение к базе данных.
//
// Параметры:
// - dbName: Имя базы данных, для которой необходимо получить соединение.
//
// Возвращает:
// - Указатель на sql.DB, если соединение успешно получено или создано.
// - Ошибка, если произошла ошибка при открытии нового соединения или при проверке существующего.
//
// Если существующее соединение не активно, оно будет закрыто и удалено из карты mapDB.
func (s *DbServiceServer) GetDb(dbName string) (*sql.DB, error) {
	if db, exists := s.mapDB[dbName]; exists {
		// Проверяем, активен ли connection
		if err := db.Ping(); err == nil {
			return db, nil // Соединение активное, возвращаем его
		}

		// Соединение не активно, закрываем и удаляем из карты
		delete(s.mapDB, dbName)
		_ = db.Close() // Игнорируем ошибки закрытия
	}

	dsn := dsnString(dbName)
	// Если соединения нет или оно было закрыто, создаем новое
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// Настройка пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(time.Minute * 30)

	s.mapDB[dbName] = db // Сохраняем новое соединение в карту
	return db, nil
}

// dsnString создает строку подключения для базы данных PostgreSQL.
//
// Параметры:
// - dbName: Имя базы данных, к которой необходимо подключиться.
//
// Возвращает:
// - Строку подключения в формате, необходимом для подключения к PostgreSQL.
//
// Строка подключения содержит следующие параметры:
// - host: Адрес хоста базы данных (DB_HOST).
// - port: Порт для подключения к базе данных (DB_PORT).
// - user: Имя пользователя для подключения к базе данных (DB_USER).
// - password: Пароль пользователя для подключения к базе данных (DB_PASSWORD).
// - dbname: Имя базы данных, переданное в качестве параметра dbName.
// - sslmode: Режим SSL (установлено значение 'disable').
func dsnString(dbName string) string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		dbName)
}

// initDB инициализирует соединение с базой данных авторизации и выполняет необходимые миграции.
//
// Параметры:
// - server: Указатель на экземпляр DbServiceServer, который будет использоваться для управления соединениями с базами данных.
//
// Возвращает:
// - Ошибка, если произошла ошибка на любом этапе инициализации. В противном случае возвращает nil.
//
// Процесс выполнения:
// 1. Загружает переменные окружения из файла .env.
// 2. Получает имя базы данных авторизации из переменной окружения DB_AUTH_NAME.
// 3. Создает базу данных авторизации, если она еще не существует, с помощью функции createInsideDB.
// 4. Открывает соединение с базой данных авторизации, используя функцию GetDb.
// 5. Добавляет полученное соединение в сервер в карту mapDB.
// 6. Выполняет миграцию для базы данных авторизации, используя указанный путь к миграциям (MIGRATION_AUTH_PATH).
// 7. Возвращает nil, если все операции выполнены успешно.
func initDB(server *DbServiceServer) error {
	// Загружаем переменные из файла .env
	err := godotenv.Load("/app/.env")
	if err != nil {
		log.Fatalf("Ошибка загрузки .env файла: %v", err)
		return err
	}

	// Находим наименование Авторизационной базы данных
	authDBName := os.Getenv("DB_AUTH_NAME")

	// Создаем Авторизационную базу данных, если она еще не существует
	err = createInsideDB(authDBName)
	if err != nil {
		log.Fatalf("Ошибка создания внутренней БД: %v", err)
		return err
	}

	// Открываем соединение с базой данных Авторизации
	dsn := dsnString(authDBName)
	// Получаем соединение с базой данных
	authDB, err := server.GetDb(dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных авторизации: %s", err)
		return err
	}

	// Добавляем соединение к базе данных авторизации в пул серверов
	server.mapDB[authDBName] = authDB

	// Путь к миграциям
	migratePath := os.Getenv("MIGRATION_AUTH_PATH")

	// Выполняем миграцию для базы данных авторизации
	err = migrations.Migration(authDB, migratePath, authDBName)
	if err != nil {
		log.Fatalf("Ошибка миграции для %s: %v", authDBName, err)
		return err
	}

	// Возвращаем nil, если все прошло успешно
	return nil
}

// createInsideDB создает новую базу данных с указанным именем, если она еще не существует.
//
// Параметры:
// - dbName: Имя базы данных, которую необходимо создать.
//
// Возвращает:
// - Ошибка, если имя базы данных пустое или если произошла ошибка при подключении к серверу базы данных, проверке существования базы данных или её создании.
// В противном случае возвращает nil.
//
// Процесс выполнения:
// 1. Проверяет, что имя базы данных не является пустым.
// 2. Создает строку подключения к серверу PostgreSQL с использованием dsnString.
// 3. Открывает соединение с сервером базы данных PostgreSQL.
// 4. Проверяет, существует ли уже база данных с указанным именем.
// 5. Если база данных существует, логирует это сообщение и возвращает nil.
// 6. Если база данных не существует, выполняет запрос на создание новой базы данных.
// 7. Логирует успешное создание базы данных и возвращает nil.
func createInsideDB(dbName string) error {
	if dbName == "" {
		return fmt.Errorf("Имя базы данных не может быть пустым")
	}

	dsn := dsnString(os.Getenv("SERVER_NAME"))

	// Открываем соединение с базой данных postgres одиночное открытие базы данных
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("Ошибка подключения к базе данных: %w", err)
	}
	/*defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Ошибка при закрытии текущего соединения: %v", err)
		}
	}()*/

	// Проверяем, существует ли уже база данных
	var exists bool
	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname='%s')`, dbName)
	err = db.QueryRow(query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("Ошибка проверки существования базы данных: %w", err)
	}

	// Если база данных уже существует, возвращаем сообщение об этом
	if exists {
		log.Printf("База данных %s уже существует", dbName)
		return nil
	}

	// Выполняем запрос на создание базы данных
	_, err = db.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, dbName))
	if err != nil {
		return fmt.Errorf("Ошибка создания базы данных %s: %w", dbName, err)
	}

	log.Printf("База данных %s успешно создана", dbName)
	return nil
}

// createClientDatabase создает новую базу данных с рандомным именем для компании
// и выполняет необходимые миграции для таблицы users.
//
// Параметры:
// - server: Указатель на экземпляр DbServiceServer, который используется для управления соединениями с базами данных.
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
func createClientDatabase(server *DbServiceServer) (nameDB string, err error) {

	// Создаём рандомное имя для базы данных компании
	randomName := utils.RandomDBName(25)

	// Функция проверки и создания базы данных
	err = createInsideDB(randomName)
	if err != nil {
		return "", fmt.Errorf("Ошибка при создании базы данных: %w", err)
	}

	// Теперь подключаемся к только что созданной базе данных
	newDSN := dsnString(randomName) // Создаем новое соединение к этой базе
	newDB, err := server.GetDb(newDSN)

	if err != nil {
		return "", fmt.Errorf("Ошибка подключения к новой базе данных: %w", err)
	}
	/*defer func(newDB *sql.DB) {
		err := newDB.Close()
		if err != nil {
			log.Fatal("Некорректное закрытие базы данных")
		}
	}(newDB)*/

	// Миграция для таблицы users
	migratePath := os.Getenv("MIGRATION_COMPANYDB_PATH")
	err = migrations.Migration(newDB, migratePath, randomName)
	if err != nil {
		return "", fmt.Errorf("Ошибка при миграции базы данных: %w", err)
	}

	return randomName, nil
}

// registerCompany регистрирует новую компанию в базе данных и создает
// соответствующего пользователя в системе авторизации.
//
// Параметры:
// - server: Указатель на экземпляр DbServiceServer, который используется для управления соединениями с базами данных.
// - req: Указатель на структуру pb.RegisterCompanyRequest, содержащую данные о компании и пользователе (имя компании, адрес, email, телефон и пароль).
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
func registerCompany(server *DbServiceServer, req *pb.RegisterCompanyRequest) (nameDB string, err error, status uint32) {
	// Получаем имя базы данных авторизации из переменных окружения.
	authDBName := os.Getenv("DB_AUTH_NAME")

	// Создаем базу данных для компании.
	dbName, err := createClientDatabase(server)
	if err != nil {
		return "", err, http.StatusInternalServerError // Возвращаем ошибку, если создание базы данных не удалось.
	}

	// Формируем строку подключения к базе данных авторизации.
	dsn := dsnString(authDBName)

	// Получаем соединение с базой данных авторизации.
	dbConn, err := server.GetDb(dsn)
	if err != nil {
		return "", err, http.StatusInternalServerError // Возвращаем ошибку, если соединение не удалось.
	}

	// Логируем состояние соединения с базой данных авторизации.
	if dbConn == nil {
		log.Println("Ошибка: соединение с базой данных авторизации не инициализировано")
		return "", fmt.Errorf("Соединение с базой данных авторизации не инициализировано"), http.StatusInternalServerError
	}

	// Начинаем транзакцию для базы данных авторизации.
	tx, err := dbConn.Begin()
	if err != nil {
		return "", fmt.Errorf("Не удалось начать транзакцию: %v", err), http.StatusInternalServerError // Возвращаем ошибку, если не удалось начать транзакцию.
	}

	defer func() { // Отложенная функция для отката транзакции в случае ошибки.
		if err != nil {
			_ = tx.Rollback()                                                 // Откатываем транзакцию.
			log.Printf("Транзакция откатана (auth DB) из-за ошибки: %v", err) // Логируем откат.
		}
	}()

	var companyId int // Переменная для хранения ID компании.

	// Проверяем, существует ли уже компания с указанным именем и адресом.
	query := "SELECT id FROM companies WHERE name = $1 AND address = $2"
	err = tx.QueryRow(query, req.NameCompany, req.Address).Scan(&companyId)
	if err != nil {
		if err == sql.ErrNoRows { // Если компания не найдена, продолжаем вставку.
			// Вставляем новую компанию и получаем её ID.
			err = tx.QueryRow(
				"INSERT INTO companies (name, address, dbname) VALUES ($1, $2, $3) RETURNING id",
				req.NameCompany, req.Address, dbName,
			).Scan(&companyId)
			if err != nil {
				return "", fmt.Errorf("Не удалось создать компанию: %v", err), http.StatusInternalServerError // Возвращаем ошибку, если вставка не удалась.
			}
		} else {
			return "", fmt.Errorf("Ошибка при проверке существования компании: %v", err), http.StatusUnprocessableEntity // Возвращаем ошибку, если произошла другая ошибка.
		}
	} else {
		return "", fmt.Errorf("Компания с таким именем и адресом уже существует: %s", req.NameCompany), http.StatusConflict // Возвращаем ошибку, если компания уже существует.
	}

	var authUserId int // Переменная для хранения ID пользователя.

	// Вставляем нового пользователя в таблицу authusers и получаем его ID.
	err = tx.QueryRow(
		"INSERT INTO authusers (email, phone, password, companyId) VALUES ($1, $2, $3, $4) RETURNING id",
		req.Email, req.Phone, req.Password, companyId,
	).Scan(&authUserId)
	if err != nil {
		return "", fmt.Errorf("Не удалось создать пользователя: %v", err), http.StatusInternalServerError // Возвращаем ошибку, если вставка не удалась.
	}

	err = tx.Commit() // Фиксируем транзакцию для базы данных авторизации.
	if err != nil {
		return "", fmt.Errorf("Не удалось зафиксировать транзакцию auth DB: %v", err), http.StatusInternalServerError // Возвращаем ошибку, если фиксация не удалась.
	}

	// Работа с базой данных компании.
	dsnC := dsnString(dbName)                // Формируем строку подключения к базе данных компании.
	dbConnCompany, err := server.GetDb(dsnC) // Получаем соединение с базой данных компании.

	if dbConnCompany == nil {
		log.Println("Ошибка: соединение с базой данных компании не инициализировано")
		return "", fmt.Errorf("Соединение с базой данных компании не инициализировано"), http.StatusInternalServerError // Возвращаем ошибку, если соединение не удалось.
	}

	txc, err := dbConnCompany.Begin() // Начинаем транзакцию для базы данных компании.
	if err != nil {
		return "", fmt.Errorf("Не удалось начать транзакцию для компании: %v", err), http.StatusInternalServerError // Возвращаем ошибку, если не удалось начать транзакцию.
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
		return "", fmt.Errorf("Не удалось добавить название прав: %v", err), http.StatusNotImplemented // Возвращаем ошибку, если вставка не удалась.
	}

	// Вставляем нового пользователя в таблицу users.
	_, err = txc.Exec(
		"INSERT INTO users (roles, companyId, rightsId, authId) VALUES ($1, $2, $3, $4)",
		role, companyId, roleID, authUserId,
	)
	if err != nil {
		return "", fmt.Errorf("Не удалось добавить пользователя: %v", err), http.StatusNotImplemented // Возвращаем ошибку, если вставка не удалась.
	}

	// Вставляем доступные действия для новой роли.
	_, err = txc.Exec(
		"INSERT INTO availableactions (roleId, createTasks, createChats, addWorkers) VALUES ($1, $2, $3, $4)",
		roleID, true, true, true,
	)
	if err != nil {
		return "", fmt.Errorf("Не удалось добавить доступные действия для роли: %v", err), http.StatusNotImplemented // Возвращаем ошибку, если вставка не удалась.
	}

	err = txc.Commit() // Фиксируем транзакцию для базы данных компании.
	if err != nil {
		return "", fmt.Errorf("Не удалось зафиксировать транзакцию компании: %v", err), http.StatusNotImplemented // Возвращаем ошибку, если фиксация не удалась.
	}

	return dbName, nil, http.StatusOK // Возвращаем имя базы данных, nil и код состояния 200 OK, если все операции выполнены успешно.
}

// rollbackAuthDB откатывает изменения в базе данных авторизации, удаляя пользователя и компанию.
func rollbackAuthDB(dbConn *sql.DB, companyId, authUserId int) {
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

// checkUser проверяет пользователя в базе данных авторизации и возвращает имя базы данных компании.
func checkUser(server *DbServiceServer, req *pb.LoginDBRequest) (dbName string, err error) {
	// Формируем строку соединения с базой данных авторизации.
	dsn := dsnString(os.Getenv("DB_AUTH_NAME"))
	// Получаем соединение с базой данных.
	db, err := server.GetDb(dsn)

	if err != nil {
		log.Printf("Ошибка подключения базы данных: %s", err) // Логируем ошибку подключения.
		return "", err                                        // Возвращаем пустую строку и ошибку.
	}

	// SQL-запрос для проверки существования пользователя по email или телефону и паролю.
	query := `
        SELECT companyId
        FROM authusers 
        WHERE (email = $1 OR phone = $2) 
        AND password = $3
    `

	// Переменная для хранения ID компании, к которой принадлежит пользователь.
	companyID := ""
	// Выполняем запрос и сканируем результат в переменную companyID.
	err = db.QueryRow(query, req.Email, req.Phone, req.Password).Scan(&companyID)
	fmt.Println(companyID) // Выводим companyID для отладки.

	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("Пользователь не найден") // Пользователь не найден, возвращаем ошибку.
		}
		return "", err // Возвращаем ошибку при выполнении запроса.
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
			return "", nil // Если компании не найдены, возвращаем пустую строку.
		}
		return "", err // Возвращаем ошибку при выполнении запроса.
	}

	return dbName, nil // Возвращаем имя базы данных и nil в качестве ошибки, если все прошло успешно.
}

// RegisterCompany обрабатывает запрос на регистрацию новой компании.
func (s *DbServiceServer) RegisterCompany(_ context.Context, req *pb.RegisterCompanyRequest) (*pb.RegisterCompanyResponse, error) {
	// Логируем получение запроса на регистрацию компании с именем из запроса.
	log.Printf("Получен запрос на регистрацию организации: %v", req.NameCompany)

	// Вызываем функцию registerCompany для создания базы данных и регистрации компании.
	dbName, err, status := registerCompany(s, req)
	if err != nil {
		// Если произошла ошибка, формируем ответ с сообщением об ошибке.
		response := &pb.RegisterCompanyResponse{
			Message:  err.Error(), // Сообщение об ошибке.
			Database: dbName,      // Имя базы данных (может быть пустым в случае ошибки).
			Status:   status,      // Статус ошибки.
		}
		return response, nil // Возвращаем ответ с ошибкой.
	}

	// Формируем успешный ответ, если регистрация прошла успешно.
	response := &pb.RegisterCompanyResponse{
		Message:  "Регистрация успешна", // Сообщение об успешной регистрации.
		Database: dbName,                // Имя базы данных для зарегистрированной компании.
		Status:   http.StatusOK,         // Статус успешного выполнения.
	}

	return response, nil // Возвращаем успешный ответ.
}

// LoginDB обрабатывает запрос на вход пользователя в систему.
func (s *DbServiceServer) LoginDB(_ context.Context, req *pb.LoginDBRequest) (*pb.LoginDBResponse, error) {
	// Проверяем пользователя, используя функцию checkUser.
	dbName, err := checkUser(s, req)

	if err != nil {
		// Если произошла ошибка при проверке пользователя, формируем ответ с сообщением об ошибке.
		response := &pb.LoginDBResponse{
			Message:  "Внутренняя ошибка: " + err.Error(), // Сообщение об ошибке.
			Database: "",                                  // Имя базы данных (пустое в случае ошибки).
			Status:   http.StatusInternalServerError,      // Статус внутренней ошибки.
		}
		return response, nil // Возвращаем ответ с ошибкой.
	}

	if dbName == "" {
		// Если база данных не найдена, формируем ответ с сообщением об ошибке.
		response := &pb.LoginDBResponse{
			Message:  "Ошибка нахождения базы данных: " + err.Error(), // Сообщение об ошибке.
			Database: "",                                              // Имя базы данных (пустое в случае ошибки).
			Status:   http.StatusInternalServerError,                  // Статус внутренней ошибки.
		}
		return response, nil // Возвращаем ответ с ошибкой.
	}

	// Формируем успешный ответ, если пользователь найден.
	response := &pb.LoginDBResponse{
		Message:  "Пользователь найден", // Сообщение об успешном входе.
		Database: dbName,                // Имя базы данных, к которой подключен пользователь.
		Status:   http.StatusOK,         // Статус успешного выполнения.
	}

	return response, nil // Возвращаем успешный ответ.
}

// SaveMessage сохраняет сообщение в базу данных.
func (s *DbServiceServer) SaveMessage(ctx context.Context, req *pb.SaveMessageRequest) (*pb.SaveMessageResponse, error) {
	// Получаем строку подключения к базе данных из переменной окружения с именем базы данных.
	dsn := dsnString(os.Getenv(req.DbName))

	// Получаем соединение с базой данных.
	db, err := s.GetDb(dsn)
	if err != nil {
		// Если произошла ошибка подключения, логируем её и возвращаем ответ с ошибкой.
		log.Printf("Ошибка подключения к базе данных: %s", err)
		return &pb.SaveMessageResponse{
			Response: fmt.Sprintf("Ошибка подключения к базе данных: %s.", err), // Сообщение об ошибке.
			Status:   http.StatusInternalServerError,                            // Статус внутренней ошибки.
		}, err
	}

	// SQL-запрос для сохранения сообщения.
	query := `
        INSERT INTO messages (chat_id, user_id, message, created_at)
        VALUES ($1, $2, $3, to_timestamp($4)) RETURNING id;
    `

	// Переменная для ID сохраненного сообщения.
	var messageID int

	// Выполняем запрос с параметрами из запроса.
	err = db.QueryRowContext(ctx, query, req.ChatId, req.UserId, req.Message, req.CreatedAt).Scan(&messageID)
	if err != nil {
		// Если произошла ошибка при выполнении запроса, логируем её и возвращаем ответ с ошибкой.
		log.Printf("Ошибка сохранения сообщения в базу данных: %s", err)
		return &pb.SaveMessageResponse{
			Response: fmt.Sprintf("Ошибка сохранения сообщения в базу данных: %s.", err), // Сообщение об ошибке.
			Status:   http.StatusInternalServerError,                                     // Статус внутренней ошибки.
		}, err
	}

	// Возвращаем успешный ответ с ID сохраненного сообщения.
	return &pb.SaveMessageResponse{
		Response: fmt.Sprintf("Сообщение успешно сохранено с ID: %d", messageID), // Сообщение о успешном сохранении.
		Status:   http.StatusOK,                                                  // Статус успешного выполнения.
	}, nil
}

// CloseAllDatabases закрывает все открытые базы данных, хранящиеся в mapDB.
func (s *DbServiceServer) CloseAllDatabases() error {
	// Проходим по каждой базе данных в карте mapDB.
	for name, db := range s.mapDB {
		// Закрываем соединение с текущей базой данных.
		if err := db.Close(); err != nil {
			// Если произошла ошибка при закрытии, возвращаем ошибку с именем базы данных и текстом ошибки.
			return fmt.Errorf("Ошибка закрытия базы данных %s: %v", name, err)
		}
	}
	// Если все базы данных успешно закрыты, возвращаем nil.
	return nil
}

func main() {
	// Инициализация пула сервера
	serverPoll := NewDbServiceServer()

	var err error
	// Инициализируем базы данных, загружая настройки из .env файла
	err = initDB(serverPoll)
	if err != nil {
		log.Fatal("Ошибка при инициализации первичной БД")
	}

	// Откладываем закрытие всех баз данных до завершения работы программы
	defer func() {
		if err := serverPoll.CloseAllDatabases(); err != nil {
			log.Fatalf("Не удалось закрыть базы данных: %v", err)
		}
	}()

	// Инициализируем TCP соединение для gRPC сервера на порту 8081
	lis, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatalf("Не удалось запустить сервер: %v", err)
	}

	//Подключаем ssl сертификацию для https
	var opts []grpc.ServerOption
	tlsCredentials, err := utils.LoadTLSCredentials()
	if err != nil {
		log.Fatalf("cannot load TLS credentials: %s", err)
	}
	opts = []grpc.ServerOption{
		grpc.Creds(tlsCredentials), // Добавление TLS опций
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     5 * time.Minute,
			MaxConnectionAge:      15 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Minute,
			Time:                  5 * time.Second, // Таймаут на соединение
		}),
	}

	// Создаем новый gRPC сервер
	grpcServer := grpc.NewServer(opts...) // Здесь можно указать опции для сервера

	// Включаем отражение для gRPC сервера
	reflection.Register(grpcServer)

	// Регистрируем наш DbServiceServer с привязкой к общему пулу соединения
	pb.RegisterDbServiceServer(grpcServer, serverPoll)

	log.Printf("gRPC сервер запущен на %s с TLS", ":8081")

	// Запуск сервера
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Ошибка запуска gRPC сервера: %v", err)
	}
}

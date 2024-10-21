package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"net/http"
	"os"
	"testAuth/migrations"
	pb "testAuth/proto/dbservice" // Импортируйте сгенерированный пакет из протобуферов
	"testAuth/utils"
	"time"
)

type DbServiceServer struct {
	pb.UnimplementedDbServiceServer
	dbs map[string]*sql.DB // Карта для хранения соединений с базами данных
}

// Конструктор для инициализации DbServiceServer
func NewDbServiceServer() *DbServiceServer {
	return &DbServiceServer{
		dbs: make(map[string]*sql.DB),
	}
}

// Функция проверки существования открытого соединения с базой данных данных
// Обеспечение использования уже существующего соединения
func (s *DbServiceServer) GetDb(dsn string) (*sql.DB, error) {
	if db, exists := s.dbs[dsn]; exists {
		// Проверяем, активен ли connection
		if err := db.Ping(); err == nil {
			return db, nil // Соединение активное, возвращаем его
		}

		// Соединение не активно, закрываем и удаляем из карты
		delete(s.dbs, dsn)
		_ = db.Close() // Игнорируем ошибки закрытия
	}

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

	s.dbs[dsn] = db // Сохраняем новое соединение в карту
	return db, nil
}

func dsnString(dbName string) string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		dbName)
}

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

	// Добавляем соединение к базе данных авторизации в сервер
	server.dbs[authDBName] = authDB

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

func createInsideDB(dbName string) error {
	if dbName == "" {
		return fmt.Errorf("имя базы данных не может быть пустым")
	}

	dsn := dsnString(os.Getenv("SERVER_NAME"))

	// Открываем соединение с базой данных postgres одиночное открытие базы данных
	db, err := sql.Open("postgres", dsn)

	if err != nil {
		return fmt.Errorf("ошибка подключения к базе данных: %w", err)
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
		return fmt.Errorf("ошибка проверки существования базы данных: %w", err)
	}

	// Если база данных уже существует, возвращаем сообщение об этом
	if exists {
		log.Printf("База данных %s уже существует", dbName)
		return nil
	}

	// Выполняем запрос на создание базы данных
	_, err = db.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, dbName))
	if err != nil {
		return fmt.Errorf("ошибка создания базы данных %s: %w", dbName, err)
	}

	log.Printf("База данных %s успешно создана", dbName)
	return nil
}

func createClientDatabase(server *DbServiceServer) (nameDB string, err error) {

	//Создаём рандомное имя для базы даннхы компании
	randomName := utils.RandomDBName(25)

	// Функция проверки и создания базы данных
	err = createInsideDB(randomName)
	if err != nil {
		return "", fmt.Errorf("ошибка при создании базы данных: %w", err)
	}

	// Теперь подключаемся к только что созданной базе данных
	newDSN := dsnString(randomName) // Создаем новое соединение к этой базе
	newDB, err := server.GetDb(newDSN)

	if err != nil {
		return "", fmt.Errorf("ошибка подключения к новой базе данных: %w", err)
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
		return "", fmt.Errorf("ошибка при миграции базы данных: %w", err)
	}

	return randomName, nil

}

/*func getAllUsers(dbName string) ([]map[string]interface{}, error) {
	dsn := fmt.Sprintf("postgres://user:password@localhost:5432/%s?sslmode=disable", dbName)
	dbConn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	defer func(dbConn *sql.DB) {
		err := dbConn.Close()
		if err != nil {
			log.Printf("некорректное закрытие базы данных")
		}
	}(dbConn)

	rows, err := dbConn.Query("SELECT id,email,phone,password,companyId,createdAt FROM users")
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("Ошибка при закрытии строк: %v", err)
		}
	}(rows)

	var users []map[string]interface{}
	for rows.Next() {
		var id int
		var email string
		if err := rows.Scan(&id, &email); err != nil {
			return nil, err
		}
		users = append(users, map[string]interface{}{
			"id":    id,
			"email": email,
		})
	}
	return users, nil
}*/

/*// dropClientDatabase удаляет базу данных по имени.
func dropClientDatabase(dbName string) error {
	dsn := dsnString(os.Getenv("DB_AUTH_NAME")) // Получите данные для подключения к основной базе данных
	dbConn, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("не удалось подключиться к базе данных: %v", err)
	}
	defer dbConn.Close()

	_, err = dbConn.Exec("DROP DATABASE IF EXISTS " + dbName)
	if err != nil {
		return fmt.Errorf("не удалось удалить базу данных: %v", err)
	}

	return nil
}*/

func registerCompany(server *DbServiceServer, req *pb.RegisterCompanyRequest) (nameDB string, err error, status int32) {
	authDBName := os.Getenv("DB_AUTH_NAME")
	dbName, err := createClientDatabase(server)
	if err != nil {
		return "", err, http.StatusInternalServerError
	}

	dsn := dsnString(authDBName)
	dbConn, err := server.GetDb(dsn)
	if err != nil {
		return "", err, http.StatusInternalServerError
	}

	// Логируем состояние соединения
	if dbConn == nil {
		log.Println("Ошибка: соединение с базой данных авторизации не инициализировано")
		return "", fmt.Errorf("соединение с базой данных авторизации не инициализировано"), http.StatusInternalServerError
	}

	tx, err := dbConn.Begin()
	if err != nil {
		return "", fmt.Errorf("не удалось начать транзакцию: %v", err), http.StatusInternalServerError
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
			log.Printf("Транзакция откатана (auth DB) из-за ошибки: %v", err)
		}
	}()

	var companyId int
	query := "SELECT id FROM companies WHERE name = $1 AND address = $2"
	err = tx.QueryRow(query, req.NameCompany, req.Address).Scan(&companyId)
	if err != nil {
		if err == sql.ErrNoRows {
			// Компания не найдена, продолжаем вставку
			err = tx.QueryRow(
				"INSERT INTO companies (name, address, dbname) VALUES ($1, $2, $3) RETURNING id",
				req.NameCompany, req.Address, dbName,
			).Scan(&companyId)
			if err != nil {
				return "", fmt.Errorf("не удалось создать компанию: %v", err), http.StatusInternalServerError
			}
		} else {
			return "", fmt.Errorf("ошибка при проверке существования компании: %v", err), http.StatusUnprocessableEntity
		}
	} else {
		return "", fmt.Errorf("компания с таким именем и адресом уже существует: %s", req.NameCompany), http.StatusConflict
	}

	var authUserId int
	err = tx.QueryRow(
		"INSERT INTO authusers (email, phone, password, companyId) VALUES ($1, $2, $3, $4) RETURNING id",
		req.Email, req.Phone, req.Password, companyId,
	).Scan(&authUserId)
	if err != nil {
		return "", fmt.Errorf("не удалось создать пользователя: %v", err), http.StatusInternalServerError
	}

	err = tx.Commit()
	if err != nil {
		return "", fmt.Errorf("не удалось зафиксировать транзакцию auth DB: %v", err), http.StatusInternalServerError
	}

	// Работа с базой данных компании
	dsnC := dsnString(dbName)
	dbConnCompany, err := server.GetDb(dsnC)

	if dbConnCompany == nil {
		log.Println("Ошибка: соединение с базой данных компании не инициализировано")
		return "", fmt.Errorf("соединение с базой данных компании не инициализировано"), http.StatusInternalServerError
	}

	txc, err := dbConnCompany.Begin()
	if err != nil {
		return "", fmt.Errorf("не удалось начать транзакцию для компании: %v", err), http.StatusInternalServerError
	}

	defer func() {
		if err != nil {
			_ = txc.Rollback()
			log.Printf("Транзакция откатана (company DB) из-за ошибки: %v", err)
			// Откат действий в первой базе данных
			rollbackAuthDB(dbConn, companyId, authUserId)
		}
	}()

	role := os.Getenv("FIRST_ROLE")

	var roleID int
	err = txc.QueryRow("INSERT INTO rights (roles) VALUES ($1) RETURNING id", role).Scan(&roleID)
	if err != nil {
		return "", fmt.Errorf("не удалось добавить название прав: %v", err), http.StatusNotImplemented
	}

	_, err = txc.Exec(
		"INSERT INTO users (roles, companyId, rightsId, authId) VALUES ($1, $2, $3, $4)",
		role, companyId, roleID, authUserId,
	)
	if err != nil {
		return "", fmt.Errorf("не удалось добавить пользователя: %v", err), http.StatusNotImplemented
	}

	_, err = txc.Exec(
		"INSERT INTO availableactions (roleId, createTasks, createChats, addWorkers) VALUES ($1, $2, $3, $4)",
		roleID, true, true, true,
	)
	if err != nil {
		return "", fmt.Errorf("не удалось добавить доступные действия для роли: %v", err), http.StatusNotImplemented
	}

	err = txc.Commit()
	if err != nil {
		return "", fmt.Errorf("не удалось зафиксировать транзакцию компании: %v", err), http.StatusNotImplemented
	}

	return dbName, nil, http.StatusOK
}

// Функция для отката изменений в auth DB
func rollbackAuthDB(dbConn *sql.DB, companyId, authUserId int) {
	tx, err := dbConn.Begin()
	if err != nil {
		log.Printf("Ошибка при начале откатной транзакции: %v", err)
		return
	}

	_, err = tx.Exec("DELETE FROM authusers WHERE id = $1", authUserId)
	if err != nil {
		tx.Rollback()
		log.Printf("Ошибка при удалении пользователя: %v", err)
		return
	}

	_, err = tx.Exec("DELETE FROM companies WHERE id = $1", companyId)
	if err != nil {
		tx.Rollback()
		log.Printf("Ошибка при удалении компании: %v", err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("Ошибка при фиксации отката: %v", err)
	}
}

func checkUser(server *DbServiceServer, req *pb.LoginDBRequest) (dbName string, err error) {

	dsn := dsnString(os.Getenv("DB_AUTH_NAME"))
	db, err := server.GetDb(dsn)

	if err != nil {
		log.Printf("Ошибка подключения базы данных: %s", err)
		return "", err
	}
	/*defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Ошибка при закрытии текущего соединения: %v", err)
		}
	}()*/

	query := `
        SELECT companyId
        FROM authusers 
        WHERE (email = $1 OR phone = $2) 
        AND password = $3
    `

	companyID := ""
	err = db.QueryRow(query, req.Email, req.Phone, req.Password).Scan(&companyID)
	fmt.Println(companyID)

	if err != nil {
		if err == sql.ErrNoRows {

			return "", fmt.Errorf("пользователь не найден") // Пользователь не найден
		}

		return "", err // Ошибка при выполнении запроса
	}

	queryCompanies := `
        SELECT dbName
        FROM companies 
        WHERE id = $1 
    `

	err = db.QueryRow(queryCompanies, companyID).Scan(&dbName)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // Пользователь не найден
		}
		return "", err // Ошибка при выполнении запроса
	}

	return dbName, nil

}

/*func registerHandler(w http.ResponseWriter, r *http.Request) {

	// Структура для JSON-данных
	type DbStruct struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		DbName  string `json:"dbName"`
	}

	//Parse Json
	var req DbStruct
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	err := registerCompany(req.Name, req.Address, req.DbName)
	if err != nil {
		http.Error(w, "Ошибка регистрации: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write([]byte("Регистрация успешно завершена"))
	if err != nil {
		http.Error(w, "Ошибка записи ответа с сервера: "+err.Error(), http.StatusInternalServerError)
	}
}*/

/*func createDatabaseHandler(w http.ResponseWriter, r *http.Request) {

	dbName, err := createClientDatabase()
	if err != nil {
		http.Error(w, "Ошибка создания базы данных: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write([]byte(dbName))
	if err != nil {
		http.Error(w, "Ошибка записиответа: "+dbName, http.StatusInternalServerError)
	}
}

func getAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	dbName := r.URL.Query().Get("db_name")
	users, err := getAllUsers(dbName)
	if err != nil {
		http.Error(w, "Ошибка получения пользователей: "+err.Error(), http.StatusInternalServerError)
		return
	}

	for _, user := range users {
		_, err := fmt.Fprintf(w, "ID: %d, Username: %s\n", user["id"], user["username"])
		if err != nil {
			return
		}
	}
}*/

/*func main() {

	var err error
	err = initDB()

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbServiceName := os.Getenv("DB_SERVICE_NAME")

	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	r.HandleFunc(fmt.Sprintf("%s/register", dbServiceName), registerHandler).Methods("POST")
	r.HandleFunc(fmt.Sprintf("%s/create-db", dbServiceName), createDatabaseHandler).Methods("POST")
	r.HandleFunc(fmt.Sprintf("%s/users", dbServiceName), getAllUsersHandler).Methods("GET")

	log.Println("db-server запущен на порту 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}*/

func (s *DbServiceServer) RegisterCompany(_ context.Context, req *pb.RegisterCompanyRequest) (*pb.RegisterCompanyResponse, error) {

	log.Printf("Получен запрос на регистрацию организации: %v", req.NameCompany)

	dbName, err, status := registerCompany(s, req)
	if err != nil {
		response := &pb.RegisterCompanyResponse{
			Message:  err.Error(),
			Database: dbName,
			Status:   status,
		}
		return response, nil

	}

	response := &pb.RegisterCompanyResponse{
		Message:  "Регистрация успешна",
		Database: dbName,
		Status:   http.StatusOK,
	}

	return response, nil
}

func (s *DbServiceServer) LoginDB(_ context.Context, req *pb.LoginDBRequest) (*pb.LoginDBResponse, error) {

	dbName, err := checkUser(s, req)

	if err != nil {
		response := &pb.LoginDBResponse{
			Message:  "Внутренняя ошибка: " + err.Error(),
			Database: "",
			Status:   http.StatusInternalServerError,
		}
		return response, nil
	}

	if dbName == "" {
		response := &pb.LoginDBResponse{
			Message:  "Ошибка нахождения базы данных: " + err.Error(),
			Database: "",
			Status:   http.StatusInternalServerError,
		}
		return response, nil
	}
	// Например, через другие микросервисы или прямой запрос в базу данных.

	// Успешний ответ сервера
	response := &pb.LoginDBResponse{
		Message:  "Пользователь найден",
		Database: dbName,
		Status:   http.StatusOK,
	}

	return response, nil
}

func (s *DbServiceServer) SaveMessage(ctx context.Context, req *pb.SaveMessageRequest) (*pb.SaveMessageResponse, error) {
	dsn := dsnString(os.Getenv(req.DbName))

	// Получаем соединение с базой данных
	db, err := s.GetDb(dsn)
	if err != nil {
		log.Printf("Ошибка подключения к базе данных: %s", err)
		return &pb.SaveMessageResponse{
			Response: fmt.Sprintf("Ошибка подключения к базе данных: %s.", err),
			Status:   http.StatusInternalServerError,
		}, err
	}

	// SQL-запрос для сохранения сообщения
	query := `
        INSERT INTO messages (chat_id, user_id, message, created_at)
        VALUES ($1, $2, $3, to_timestamp($4)) RETURNING id;
    `

	// Переменная для ID сохраненного сообщения
	var messageID int

	// Выполняем запрос
	err = db.QueryRowContext(ctx, query, req.ChatId, req.UserId, req.Message, req.CreatedAt).Scan(&messageID)
	if err != nil {
		log.Printf("Ошибка сохранения сообщения в базу данных: %s", err)
		return &pb.SaveMessageResponse{
			Response: fmt.Sprintf("Ошибка сохранения сообщения в базу данных: %s.", err),
			Status:   http.StatusInternalServerError,
		}, err
	}

	// Возвращаем успешный ответ
	return &pb.SaveMessageResponse{
		Response: fmt.Sprintf("Сообщение успешно сохранено с ID: %d", messageID),
		Status:   http.StatusOK,
	}, nil
}

func (s *DbServiceServer) CloseAllDatabases() error {
	for name, db := range s.dbs {
		if err := db.Close(); err != nil {
			return fmt.Errorf("error closing database %s: %v", name, err)
		}
	}
	return nil
}

func main() {

	// Инициализация пула сервера
	serverPoll := NewDbServiceServer()

	var err error
	err = initDB(serverPoll)

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Откладываем закрытие всех баз данных
	defer func() {
		if err := serverPoll.CloseAllDatabases(); err != nil {
			log.Fatalf("Failed to close databases: %v", err)
		}
	}()

	/*r := mux.NewRouter()

	authServiceName := os.Getenv("AUTH_SERVICE_NAME")

	// Регистрация маршрутов
	r.HandleFunc(fmt.Sprintf("/%s/register", authServiceName), registerHandler).Methods("POST")
	r.HandleFunc(fmt.Sprintf("/%s/test", authServiceName), getTest).Methods("GET")
	r.HandleFunc(fmt.Sprintf("/%s/login", authServiceName), loginHandler).Methods("GET")

	log.Println("auth-service запущен на порту 8081")
	err := http.ListenAndServe(":8081", r)
	if err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}*/
	// Чтение порта из переменной окружения (например, ":50051")
	/*port := os.Getenv("AUTH_SERVICE_PORT")
	if port == "" {
		port = "50051" // если не указано в переменной окружения, используем порт по умолчанию
	}*/

	// Инициализируем TCP соединение для gRPC сервера
	lis, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatalf("Не удалось запустить сервер: %v", err)
	}

	/*// Загрузка TLS-учетных данных
	certFile := "dbservice/ssl/cert.pem" // Укажите путь к вашему сертификату
	keyFile := "dbservice/ssl/key.pem"   // Укажите путь к вашему ключу
	creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	if err != nil {
		log.Fatalf("Не удалось загрузить сертификаты: %v", err)
	}

	// Создаем gRPC сервер с TLS
	opts := []grpc.ServerOption{
		grpc.Creds(creds),                      // Используйте креденшлы
		grpc.MaxRecvMsgSize(1024 * 1024 * 150), // Увеличить размер принимаемых сообщений до 150MB
		grpc.MaxSendMsgSize(1024 * 1024 * 150), // Увеличить размер отправляемых сообщений до 150MB
	}*/

	grpcServer := grpc.NewServer( /*opts...*/ )

	// Включаем отражение
	reflection.Register(grpcServer)

	// Регистрируем наш AuthServiceServer с привязкой к общему пулу соединения
	pb.RegisterDbServiceServer(grpcServer, serverPoll)

	log.Printf("gRPC сервер запущен на %s с TLS", ":8081")

	// Запуск сервера
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Ошибка запуска gRPC сервера: %v", err)
	}
}

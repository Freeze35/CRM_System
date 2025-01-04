package main

import (
	"crmSystem/dbadminservice"
	"crmSystem/dbauthservice"
	"crmSystem/dbchatservice"
	"crmSystem/dbtimerservice"
	"crmSystem/migrations"
	pbAdmin "crmSystem/proto/dbadmin" // Импортируйте сгенерированный пакет из протобуферов dbtimer
	pbAuth "crmSystem/proto/dbauth"   // Импортируйте сгенерированный пакет из протобуферов dbauth
	pbChat "crmSystem/proto/dbchat"   // Импортируйте сгенерированный пакет из протобуферов dbchat
	pbTimer "crmSystem/proto/dbtimer" // Импортируйте сгенерированный пакет из протобуферов dbtimer
	"crmSystem/utils"
	"database/sql"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

// initDB инициализирует соединение с базой данных авторизации и выполняет необходимые миграции.
//
// Параметры:
// - server: Указатель на экземпляр MapConnectionsDB, который будет использоваться для управления соединениями с базами данных.
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
func initDB(server *utils.MapConnectionsDB) error {
	// Загружаем переменные из файла .env
	err := godotenv.Load("/app/.env")
	if err != nil {
		log.Fatalf("Ошибка загрузки .env файла: %v", err)
		return err
	}

	// Находим наименование Авторизационной базы данных
	authDBName := os.Getenv("DB_AUTH_NAME")

	// Создаем Авторизационную базу данных, если она еще не существует
	err = utils.CreateInsideDB(authDBName)
	if err != nil {
		log.Fatalf("Ошибка создания внутренней БД: %v", err)
		return err
	}

	// Открываем соединение с базой данных Авторизации
	dsn := utils.DsnString(authDBName)
	// Получаем соединение с базой данных
	authDB, err := server.GetDb(dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных авторизации: %s", err)
		return err
	}

	// Добавляем соединение к базе данных авторизации в пул серверов
	server.MapDB[authDBName] = authDB

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

/*func (s *MapConnectionsDB) AddUser(ctx context.Context, req *pb.LoginDBRequest) (*pb.LoginDBResponse, error) {
	// Проверяем пользователя, используя функцию checkUser.
	dbName, userId, err := checkUser(s, req, ctx)

	if err != nil {
		// Если произошла ошибка при проверке пользователя, формируем ответ с сообщением об ошибке.
		response := &pb.LoginDBResponse{
			Message:       "Внутренняя ошибка: " + err.Error(), // Сообщение об ошибке.
			Database:      "",                                  // Имя базы данных (пустое в случае ошибки).
			UserCompanyId: "",                                  // ID пользователя в БД компании.
			Status:        http.StatusInternalServerError,      // Статус внутренней ошибки.
		}
		return response, nil // Возвращаем ответ с ошибкой.
	}

	//Проверяем найдена ли база данных для данного пользователя
	if dbName == "" {
		// Если база данных не найдена, формируем ответ с сообщением об ошибке.
		response := &pb.LoginDBResponse{
			Message:       "Ошибка нахождения базы данных: " + err.Error(), // Сообщение об ошибке.
			Database:      "",                                              // Имя базы данных (пустое в случае ошибки).
			UserCompanyId: "",                                              // ID пользователя в БД компании.
			Status:        http.StatusInternalServerError,                  // Статус внутренней ошибки.
		}
		return response, nil // Возвращаем ответ с ошибкой.
	}

	// Формируем успешный ответ, если пользователь найден.
	response := &pb.LoginDBResponse{
		Message:       "Пользователь найден", // Сообщение об успешном входе.
		Database:      dbName,                // Имя базы данных, к которой подключен пользователь.
		UserCompanyId: userId,                // ID пользователя в БД компании.
		Status:        http.StatusOK,         // Статус успешного выполнения.
	}

	return response, nil // Возвращаем успешный ответ.
}*/

/*func formUsersKeyRedis(req *pb.RegisterUsersRequest) string {
	// Формируем ключ для Redis
	var userIdentifiers []string
	for _, user := range req.Users {
		// Например, используем email как идентификатор
		userIdentifiers = append(userIdentifiers, user.Email)
	}

	// Объединяем идентификаторы пользователей через разделитель
	userString := strings.Join(userIdentifiers, ",")

	// Формируем ключ для Redis
	redisKey := "RegisterUsers" + userString
	return redisKey
}

func registerUsersInCompany(server *MapConnectionsDB, req *pb.RegisterUsersRequest, token string) (nameDB string, err error, userCompanyId string, status uint32) {

	// В случае превышения порога ожидания с сервера в 10 секунд будет ошибка контекста.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Проверка базы данных в Redis
	client, err, connRedis := utils.RedisServiceConnector(token)
	if err != nil {
		fmt.Printf("Ошибка Подключение к redis: %v", err)
		return "", err, "", http.StatusInternalServerError
	}
	defer connRedis.Close()

	// Создаем запрос для Redis
	req1 := &redis.GetRedisRequest{
		Key: formUsersKeyRedis(req),
	}

	// Выполняем gRPC вызов для проверки компании в Redis
	resRedis, err := client.Get(ctx, req1)
	if err != nil || resRedis.Status != http.StatusOK {
		return "", fmt.Errorf("Ошибка при получении данных из Redis: %v", err), "", http.StatusInternalServerError
	}

	// Преобразуем данные из Redis
	convertedRedis, err := utils.ConvertJSONToStruct[pb.RegisterCompanyResponse](resRedis.Message)
	if err != nil {
		return "", fmt.Errorf("Ошибка при преобразовании данных из Redis: %v", err), "", http.StatusInternalServerError
	}

	// Получаем имя базы данных компании из Redis
	dbName := convertedRedis.Database
	userCompanyId := convertedRedis.UserCompanyId

	// Создаем соединение с базой данных авторизации
	authDBName := os.Getenv("DB_AUTH_NAME")
	dsn := utils.DsnString(authDBName)
	dbConn, err := server.GetDb(dsn)
	if err != nil {
		return "", err, "", http.StatusInternalServerError
	}

	if dbConn == nil {
		return "", fmt.Errorf("Соединение с базой данных авторизации не инициализировано"), "", http.StatusInternalServerError
	}

	// Начинаем транзакцию для базы данных авторизации
	tx, err := dbConn.Begin()
	if err != nil {
		return "", fmt.Errorf("Не удалось начать транзакцию: %v", err), "", http.StatusInternalServerError
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			log.Printf("Транзакция откатана (auth DB) из-за ошибки: %v", err)
		}
	}()

	// Работа с базой данных компании.
	dsnC := utils.DsnString(dbName)          // Формируем строку подключения к базе данных компании.
	dbConnCompany, err := server.GetDb(dsnC) // Получаем соединение с базой данных компании.

	if dbConnCompany == nil {
		log.Println("Ошибка: соединение с базой данных компании не инициализировано")
		return "", fmt.Errorf("Соединение с базой данных компании не инициализировано"), "", http.StatusInternalServerError // Возвращаем ошибку, если соединение не удалось.
	}

	txc, err := dbConnCompany.Begin() // Начинаем транзакцию для базы данных компании.
	if err != nil {
		return "", fmt.Errorf("Не удалось начать транзакцию для компании: %v", err), "", http.StatusInternalServerError // Возвращаем ошибку, если не удалось начать транзакцию.
	}

	defer func() { // Отложенная функция для отката транзакции в случае ошибки.
		if err != nil {
			_ = txc.Rollback()                                                   // Откатываем транзакцию.
			log.Printf("Транзакция откатана (company DB) из-за ошибки: %v", err) // Логируем откат.
			// Откат действий в первой базе данных.
			rollbackAuthDB(dbConn, companyId, authUserId)
		}
	}()

	// Обрабатываем добавление новых пользователей
	for _, user := range req.Users {
		var existingUserId int

		// Проверяем, существует ли пользователь с указанным email или phone
		err = tx.QueryRow(
			"SELECT id FROM authusers WHERE email = $1 OR phone = $2",
			user.Email, user.Phone,
		).Scan(&existingUserId)

		if err == nil {
			// Пользователь уже существует, пропускаем его создание
			continue
		} else if err != sql.ErrNoRows {
			// Обрабатываем другие ошибки
			return "", fmt.Errorf("Ошибка при проверке существования пользователя: %v", err), "", http.StatusInternalServerError
		}

		// Пользователь не существует, создаем нового
		var userId int
		err = tx.QueryRow(
			"INSERT INTO authusers (email, phone, password, company_id) VALUES ($1, $2, $3, $4) RETURNING id",
			user.Email, user.Phone, user.Password, req.CompanyId,
		).Scan(&userId)
		if err != nil {
			return "", fmt.Errorf("Не удалось создать пользователя: %v", err), "", http.StatusInternalServerError
		}

		// Добавляем пользователя в таблицу users
		role := user.Role
		var roleId int
		err = txc.QueryRow("INSERT INTO rights (roles) VALUES ($1) RETURNING id", role).Scan(&roleId)
		if err != nil {
			return "", fmt.Errorf("Не удалось добавить роль: %v", err), "", http.StatusInternalServerError
		}

		var newUserId string
		err = txc.QueryRow(
			"INSERT INTO users (company_id, rightsId, authId) VALUES ($1, $2, $3) RETURNING id",
			req.CompanyId, roleId, userId,
		).Scan(&newUserId)
		if err != nil {
			return "", fmt.Errorf("Не удалось добавить пользователя в систему: %v", err), "", http.StatusInternalServerError
		}

		// Дополнительная логика для добавления действий, доступных пользователю (если нужно)
		_, err = txc.Exec(
			"INSERT INTO availableactions (roleId, createTasks, createChats, addWorkers) VALUES ($1, $2, $3, $4)",
			roleId, true, true, true,
		)
		if err != nil {
			return "", fmt.Errorf("Не удалось добавить действия для роли: %v", err), "", http.StatusInternalServerError
		}
	}

	// Зафиксировать транзакцию
	err = tx.Commit()
	err = txc.Commit()
	if err != nil {
		return "", fmt.Errorf("Не удалось зафиксировать транзакцию: %v", err), "", http.StatusInternalServerError
	}

	// Преобразуем данные для отправки в Redis
	toJsonType := &pb.RegisterCompanyResponse{
		Message:       req.CompanyName,
		Database:      dbName,
		UserCompanyId: userCompanyId,
		Status:        http.StatusOK,
	}

	toJsonRedis, err := utils.ConvertStructToJSON(toJsonType)
	if err != nil {
		return "", fmt.Errorf("Ошибка при преобразовании данных для Redis: %v", err), "", http.StatusInternalServerError
	}

	// Сохраняем данные в Redis
	saveRequest := &redis.SaveRedisRequest{
		Key:        req.CompanyName + "Register" + userCompanyId,
		Value:      toJsonRedis,
		Expiration: int64((time.Minute * 10).Seconds()),
	}

	_, err = client.Save(ctx, saveRequest)
	if err != nil {
		return "", fmt.Errorf("Ошибка при сохранении данных в Redis: %v", err), "", http.StatusInternalServerError
	}

	return dbName, nil, userCompanyId, http.StatusOK
}*/

// fullCompaniesMigrations проходимся по всем базам данных и применяем последние миграционные обновления
func fullCompaniesMigrations() {
	dsn := utils.DsnString("postgres")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных postgres: %v", err)
	}
	defer db.Close()

	// Получаем список баз данных для миграции
	dbNames, err := migrations.GetDatabasesToMigrate(db)
	if err != nil {
		log.Fatalf("Ошибка получения списка баз данных: %v", err)
	}

	// Путь к миграциям для компаний
	migratePath := os.Getenv("MIGRATION_COMPANYDB_PATH")

	// Запускаем миграции в параллельных горутинах
	var wg sync.WaitGroup

	//Счётчик базщ данных
	counter := 0

	for _, dbName := range dbNames {
		wg.Add(1)

		go migrations.MigrateCompanyDatabase(dbName, migratePath, &wg, &counter)
	}

	// Ожидаем завершения всех горутин
	wg.Wait()

	log.Println("Миграционные обновления завершены для всех баз данных.")
	log.Printf("Проверено и обновлено %s баз данных", strconv.Itoa(counter))
}

func main() {
	// Инициализация пула сервера
	serverPoll := utils.NewMapConnectionsDB()

	var err error
	// Инициализируем базы данных, загружая настройки из .env файла
	err = initDB(serverPoll)

	if err != nil {
		log.Fatal("Ошибка при инициализации первичной БД")
	}

	fullCompaniesMigrations()

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
		log.Fatalf("Невозможно загрузить учетные данные TLS: %s", err)
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

	// Регистрируем AuthService с привязкой к общему переданному пулу соединений
	authService := dbauthservice.NewGRPCDBAuthService(serverPoll)
	pbAuth.RegisterDbAuthServiceServer(grpcServer, authService)

	// Регистрируем TimerService с привязкой к общему переданному пулу соединений
	timerService := dbtimerservice.NewGRPCDBTimerService(serverPoll)
	pbTimer.RegisterDbTimerServiceServer(grpcServer, timerService)

	// Регистрируем ChatService с привязкой к общему переданному пулу соединений
	chatService := dbchatservice.NewGRPCDBChatService(serverPoll)
	pbChat.RegisterDbChatServiceServer(grpcServer, chatService)

	// Регистрируем AdminService с привязкой к общему переданному пулу соединений
	adminService := dbadminservice.NewGRPCDBAdminService(serverPoll)
	pbAdmin.RegisterDbAdminServiceServer(grpcServer, adminService)

	log.Printf("gRPC сервер запущен на %s с TLS", ":8081")

	// Запуск сервера
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Ошибка запуска gRPC сервера: %v", err)
	}
}

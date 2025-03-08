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
	"fmt"
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

	// Путь к миграциям
	migratePath := os.Getenv("MIGRATION_AUTH_PATH")

	// Конфигурация для повторных попыток
	maxAttempts := 10                       // Максимальное количество попыток
	delayBetweenAttempts := 5 * time.Second // Интервал между попытками

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Printf("Попытка подключения к базе данных авторизации (%d/%d)", attempt, maxAttempts)

		// Формируем строку подключения
		dsn := utils.DsnString(authDBName)

		// Получаем соединение с базой данных
		authDB, err := server.GetDb(dsn)
		if err != nil {
			log.Printf("Не удалось подключиться к базе данных: %s", err)

			// Если это последняя попытка, возвращаем ошибку
			if attempt == maxAttempts {
				log.Fatalf("Ошибка подключения к базе данных авторизации после %d попыток: %s", maxAttempts, err)
				return err
			}

			// Ждем перед следующей попыткой
			time.Sleep(delayBetweenAttempts)
			continue
		}

		// Проверяем, готова ли база данных
		for checkAttempt := 1; checkAttempt <= maxAttempts; checkAttempt++ {
			log.Printf("Проверка готовности базы данных (попытка %d/%d)", checkAttempt, maxAttempts)
			if isDatabaseReady(authDB) {
				break
			}

			if checkAttempt == maxAttempts {
				log.Fatalf("База данных %s не готова после %d попыток проверки", authDBName, maxAttempts)
				return fmt.Errorf("база данных %s не готова", authDBName)
			}

			time.Sleep(delayBetweenAttempts)
		}

		// Добавляем соединение к базе данных авторизации в пул серверов
		server.MapDB[authDBName] = authDB

		// Выполняем миграцию для базы данных авторизации
		err = migrations.Migration(authDB, migratePath, authDBName)
		if err != nil {
			log.Printf("Ошибка миграции для %s: %v", authDBName, err)

			// Если это последняя попытка, возвращаем ошибку
			if attempt == maxAttempts {
				log.Fatalf("Ошибка миграции для %s после %d попыток: %v", authDBName, maxAttempts, err)
				return err
			}

			// Ждем перед следующей попыткой
			time.Sleep(delayBetweenAttempts)
			continue
		}

		// Если подключение, проверка и миграция успешны, выходим из цикла
		log.Println("Успешное подключение, проверка готовности и миграция базы данных авторизации")
		return nil
	}

	// Возвращаем ошибку, если все попытки исчерпаны
	return fmt.Errorf("не удалось подключиться и выполнить миграцию для базы данных авторизации после %d попыток", maxAttempts)
}

// Проверка готовности базы данных
func isDatabaseReady(db *sql.DB) bool {
	query := `SELECT 1 FROM pg_database WHERE datname = current_database()`
	var result int
	err := db.QueryRow(query).Scan(&result)
	if err != nil {
		log.Printf("База данных еще не готова: %s", err)
		return false
	}
	return result == 1
}

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
		grpc.UnaryInterceptor(utils.RecoveryInterceptor),
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

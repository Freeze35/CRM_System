package main

import (
	"database/sql"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"testAuth/migrations"
	"testAuth/utils"
)

func dsnString(dbName string) string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		dbName)
}

func initDB() error {
	// Загружаем переменные из файла .env
	err := godotenv.Load("/app/.env")

	if err != nil {
		log.Fatalf("Ошибка загрузки .env файла: %v", err)
	}

	dsn := dsnString(os.Getenv("DB_NAME"))

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения базы данных: %s", err)
		return err
	}

	/*err = db.Ping()
	if err != nil {
		return err
	}*/

	//Находим наименование Авторизационной базы данных
	authDBName := os.Getenv("DB_AUTH_NAME")

	//создаём Авторизационную базу данных если она ещё не существует в докер образе
	err = createInsideDB(authDBName)
	if err != nil {
		log.Fatal(fmt.Sprintf("ошибка создания внутренней БД ", err))
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Ошибка при закрытии текущего соединения: %v", err)
		}
	}()

	//открываем базу данных Авторизации для обновления миграции
	dsn = dsnString(authDBName)
	if err != nil {
		log.Fatalf("Ошибка подключения базы данных: %s", err)
		return err
	}

	db, err = sql.Open("postgres", dsn)

	migratePath := ""

	migratePath = os.Getenv("MIGRATION_AUTH_PATH")
	err = migrations.Migration(db, migratePath, authDBName)
	if err != nil {
		log.Fatal(fmt.Sprintf("ошибка MIGRATION_AUTH_PATH ", err))
		return err
	}

	if err != nil {
		log.Fatal(fmt.Sprintf("ошибка MIGRATION_COMPANIES_PATH ", err))
		return err
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Ошибка при закрытии текущего соединения: %v", err)
		}
	}()

	return nil
}

func createInsideDB(dbName string) error {
	if dbName == "" {
		return fmt.Errorf("имя базы данных не может быть пустым")
	}

	dsn := dsnString(os.Getenv("DB_NAME"))

	// Открываем соединение с базой данных postgres
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("ошибка подключения к базе данных: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Ошибка при закрытии текущего соединения: %v", err)
		}
	}()

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

func createDatabase() (nameDB string, err error) {

	randomName := utils.RandomDBName(25)

	// Открываем соединение с базой данных postgres
	dsn := dsnString(os.Getenv("DB_NAME"))

	newDB, err := sql.Open("postgres", dsn)

	if err != nil {
		return "", fmt.Errorf("ошибка при открытии базы данных: %w", err)
	}

	migratePath := ""

	//Функция проверки создания авторизационной базы или базы компании
	err = createInsideDB(randomName)
	if err != nil {
		return randomName, fmt.Errorf("ошибка вызова создания базы данных: %w", err)
	}

	//миграция для таблицы users
	migratePath = os.Getenv("MIGRATION_USERS_PATH")
	err = migrations.Migration(newDB, migratePath, randomName)
	if err != nil {
		return "", err
	}

	defer func(newDB *sql.DB) {
		err := newDB.Close()
		if err != nil {
			log.Fatal("Некорректное закрытие базы данных")
		}
	}(newDB)

	return randomName, nil

}

func getAllUsers(dbName string) ([]map[string]interface{}, error) {
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
}

func registerCompany(name, address, dbName string) error {
	dsn := dsnString(dbName) /* fmt.Sprintf("postgres://user:password@localhost:5432/%s?sslmode=disable", dbname)*/
	dbConn, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer func(dbConn *sql.DB) {
		err := dbConn.Close()
		if err != nil {
			log.Fatal(fmt.Sprintf("Некорректное закрытие базы данных %s", dbName))
		}
	}(dbConn)

	_, err = dbConn.Exec("INSERT INTO companies (name, address,dbname) VALUES ($1, $2, $3)", name, address, dbName)
	if err != nil {
		return err
	}

	return nil
}

func registerHandler(w http.ResponseWriter, r *http.Request) {

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
}

func createDatabaseHandler(w http.ResponseWriter, r *http.Request) {

	dbName, err := createDatabase()
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
}

func main() {

	var err error
	err = initDB()

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/register", registerHandler).Methods("POST")
	r.HandleFunc("/create-db", createDatabaseHandler).Methods("POST")
	r.HandleFunc("/users", getAllUsersHandler).Methods("GET")

	log.Println("db-server запущен на порту 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

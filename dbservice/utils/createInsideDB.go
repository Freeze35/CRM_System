package utils

import (
	"database/sql"
	"fmt"
	"log"
	"os"
)

// CreateInsideDB создает новую базу данных с указанным именем, если она еще не существует.
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
// 2. Создает строку подключения к серверу PostgreSQL с использованием utils.DsnString.
// 3. Открывает соединение с сервером базы данных PostgreSQL.
// 4. Проверяет, существует ли уже база данных с указанным именем.
// 5. Если база данных существует, логирует это сообщение и возвращает nil.
// 6. Если база данных не существует, выполняет запрос на создание новой базы данных.
// 7. Логирует успешное создание базы данных и возвращает nil.
func CreateInsideDB(dbName string) error {
	if dbName == "" {
		return fmt.Errorf("Имя базы данных не может быть пустым")
	}

	dsn := DsnString(os.Getenv("SERVER_NAME"))

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

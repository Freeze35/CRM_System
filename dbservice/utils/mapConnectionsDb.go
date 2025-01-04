package utils

import (
	"database/sql"
	"fmt"
	"time"
)

type MapConnectionsDB struct {
	MapDB map[string]*sql.DB // Карта для хранения соединений с базами данных
}

// Конструктор для инициализации MapConnectionsDB
func NewMapConnectionsDB() *MapConnectionsDB {
	return &MapConnectionsDB{
		MapDB: make(map[string]*sql.DB),
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
func (s *MapConnectionsDB) GetDb(dbName string) (*sql.DB, error) {
	if db, exists := s.MapDB[dbName]; exists {
		// Проверяем, активен ли connection
		if err := db.Ping(); err == nil {
			return db, nil // Соединение активное, возвращаем его
		}

		// Соединение не активно, закрываем и удаляем из карты
		delete(s.MapDB, dbName)
		_ = db.Close() // Игнорируем ошибки закрытия
	}

	dsn := DsnString(dbName)
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

	s.MapDB[dbName] = db // Сохраняем новое соединение в карту
	return db, nil
}

// CloseAllDatabases закрывает все открытые базы данных, хранящиеся в mapDB.
func (s *MapConnectionsDB) CloseAllDatabases() error {
	// Проходим по каждой базе данных в карте mapDB.
	for name, db := range s.MapDB {
		// Закрываем соединение с текущей базой данных.
		if err := db.Close(); err != nil {
			// Если произошла ошибка при закрытии, возвращаем ошибку с именем базы данных и текстом ошибки.
			return fmt.Errorf("Ошибка закрытия базы данных %s: %v", name, err)
		}
	}
	// Если все базы данных успешно закрыты, возвращаем nil.
	return nil
}

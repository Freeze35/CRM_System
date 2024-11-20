package migrations

import (
	"database/sql"
	"fmt"
)

// GetDatabasesToMigrate Получает список всех баз данных, исключая стандартные
func GetDatabasesToMigrate(db *sql.DB) ([]string, error) {
	var dbNames []string
	query := `SELECT datname FROM pg_database WHERE datistemplate = false AND datname NOT IN ('postgres', 'AuthorizationDB')`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("Ошибка запроса списка баз данных: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			return nil, fmt.Errorf("Ошибка при сканировании имени базы данных: %w", err)
		}
		dbNames = append(dbNames, dbName)
	}

	return dbNames, nil
}

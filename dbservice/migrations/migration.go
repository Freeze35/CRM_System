package migrations

import (
	"database/sql"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	postgresMigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // Пустой импорт для драйвера файла
	"log"
)

func Migration(db *sql.DB, migratePath string, dbName string) error {
	driver, err := postgresMigrate.WithInstance(db, &postgresMigrate.Config{})

	if err != nil {
		log.Fatal(fmt.Sprintf("ошибка postgresMigrate.Config. ", err))
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		migratePath,
		dbName,
		driver,
	)

	if err != nil {
		log.Fatal(fmt.Sprintf("ошибка создания миграции базы данных. ", err))
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {

		log.Fatal(fmt.Sprintf("ошибка повышения миграции. ", err))
		return err
	} else {
		log.Printf("Миграция основной базы данных, обновлена до последней версии")
		return nil
	}

}

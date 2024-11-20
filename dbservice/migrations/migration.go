package migrations

import (
	"crmSystem/utils"
	"database/sql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // Пустой импорт для драйвера файла
	"log"
	"sync"
)

func Migration(db *sql.DB, migratePath string, dbName string) error {
	// Создаём инстанс драйвера для PostgreSQL
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Printf("Ошибка создания инстанса миграции для PostgreSQL: %v", err)
		return err
	}

	// Создаём мигратор с указанием пути к миграциям
	m, err := migrate.NewWithDatabaseInstance(
		migratePath,
		dbName,
		driver,
	)
	if err != nil {
		log.Printf("Ошибка создания миграции базы данных: %v", err)
		return err
	}

	// Получаем текущую версию базы данных
	version, dirty, err := m.Version()

	//Если мы получили ошибку значит базы данных не существовало
	if err != nil {
		log.Printf("Ошибка при получении текущей версии миграции: %v", err)
		if err := m.Up(); err != nil {
			if err != migrate.ErrNoChange {
				log.Printf("Ошибка при выполнении миграции: %v", err)
				return err
			}
			//log.Printf("Миграция не требует изменений.")
		} else {
			log.Printf("Миграция выполнена успешно.")
		}
		return nil
	}

	// Проверка, если база данных находится в грязном состоянии
	if dirty {
		log.Printf("База данных находится в грязном состоянии на версии %d. Попытка исправить.", version)

		// Исправляем грязное состояние, используя Force(0) или Force(1)
		if err := m.Force(int(version) - 1); err != nil {
			log.Printf("Ошибка при установке принудительной версии для миграции: %v", err)
			return err
		}
		log.Printf("База данных была исправлена, откат на версию %d", version-1)

		// Теперь можем откатить базу на одну версию назад
		if err := m.Steps(-1); err != nil {
			log.Printf("Ошибка при откате миграции на одну версию назад: %v", err)
			return err
		}
		log.Printf("База данных откатилась на одну версию назад.")

		// Повторно выполняем миграцию, чтобы обновить базу данных до последней версии
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Printf("Ошибка при повторной миграции: %v", err)
			return err
		}
		log.Printf("Миграция выполнена успешно.")
	} else {
		// Если база данных не в грязном состоянии, просто применяем миграцию
		if err := m.Up(); err != nil {
			if err != migrate.ErrNoChange {
				log.Printf("Ошибка при выполнении миграции: %v", err)
				return err
			}
			//log.Printf("Миграция не требует изменений.")
		} else {
			log.Printf("Миграция выполнена успешно.")
		}
	}

	return nil
}

// Обертка для функции миграции по всем базам данных
func MigrateCompanyDatabase(dbName string, migratePath string, wg *sync.WaitGroup, counter *int) {
	defer wg.Done()

	// Создаем DSN для подключения к базе данных
	dsn := utils.DsnString(dbName)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Printf("Ошибка подключения к базе данных %s: %v", dbName, err)
		return
	}
	defer db.Close()

	// Выполняем миграцию с помощью функции Migration
	err = Migration(db, migratePath, dbName)
	if err != nil {
		log.Printf("Ошибка миграции базы данных %s: %v", dbName, err)
	} else {
		*counter++
		//log.Printf("Миграция для базы данных %s успешно выполнена", dbName)
	}
}

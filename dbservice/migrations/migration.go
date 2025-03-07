package migrations

import (
	"context"
	"crmSystem/proto/logs"
	"crmSystem/utils"
	"database/sql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // Пустой импорт для драйвера файла
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"sync"
)

func Migration(db *sql.DB, migratePath string, dbName string) error {

	ctx := context.Background()

	token, err := utils.JwtGenerate()
	if err != nil {
		err := status.Errorf(codes.Internal, "Не удалось создать токен ", err)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	// Устанавливаем соединение с gRPC сервером Logs
	clientLogs, err, conn := utils.GRPCServiceConnector(token, logs.NewLogsServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		return status.Errorf(codes.Unauthenticated, "Не удалось создать соединение с сервером Logs")
	} else {
		defer func(conn *grpc.ClientConn) {
			err := conn.Close()
			if err != nil {
				log.Printf("Ошибка закрытия соединения: %v", err)
				errLogs := utils.SaveLogsError(ctx, clientLogs, "basemigration", "", err.Error())
				if errLogs != nil {
					log.Printf("Ошибка закрытия соединения: %v", err)
				}
			}
		}(conn)
	}

	// Создаём инстанс драйвера для PostgreSQL
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Printf("Ошибка создания инстанса миграции для PostgreSQL: %v", err)
		errLogs := utils.SaveLogsError(ctx, clientLogs, "basemigration", "", err.Error())
		if errLogs != nil {
			log.Printf("Ошибка сохранения логов: %v", err)
		}
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
		errLogs := utils.SaveLogsError(ctx, clientLogs, "basemigration", "", err.Error())
		if errLogs != nil {
			log.Printf("Ошибка сохранения логов: %v", err)
		}
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
				errLogs := utils.SaveLogsError(ctx, clientLogs, "basemigration", "", err.Error())
				if errLogs != nil {
					log.Printf("Ошибка сохранения логов: %v", err)
				}
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
			errLogs := utils.SaveLogsError(ctx, clientLogs, "basemigration", "", err.Error())
			if errLogs != nil {
				log.Printf("Ошибка сохранения логов: %v", err)
			}
			return err
		}
		log.Printf("База данных была исправлена, откат на версию %d", version-1)

		// Теперь можем откатить базу на одну версию назад
		if err := m.Steps(-1); err != nil {
			log.Printf("Ошибка при откате миграции на одну версию назад: %v", err)
			errLogs := utils.SaveLogsError(ctx, clientLogs, "basemigration", "", err.Error())
			if errLogs != nil {
				log.Printf("Ошибка сохранения логов: %v", err)
			}
			return err
		}
		log.Printf("База данных откатилась на одну версию назад.")

		// Повторно выполняем миграцию, чтобы обновить базу данных до последней версии
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Printf("Ошибка при повторной миграции: %v", err)
			errLogs := utils.SaveLogsError(ctx, clientLogs, "basemigration", "", err.Error())
			if errLogs != nil {
				log.Printf("Ошибка сохранения логов: %v", err)
			}
			return err
		}
		log.Printf("Миграция выполнена успешно.")
	} else {
		// Если база данных не в грязном состоянии, просто применяем миграцию
		if err := m.Up(); err != nil {
			if err != migrate.ErrNoChange {
				log.Printf("Ошибка при выполнении миграции: %v", err)
				errLogs := utils.SaveLogsError(ctx, clientLogs, "basemigration", "", err.Error())
				if errLogs != nil {
					log.Printf("Ошибка сохранения логов: %v", err)
				}
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

	ctx := context.Background()

	token, err := utils.JwtGenerate()
	if err != nil {
		err := status.Errorf(codes.Internal, "Не удалось создать токен ", err)
		if err != nil {
			log.Println(err)
			return
		}
	}

	// Устанавливаем соединение с gRPC сервером Logs
	clientLogs, err, conn := utils.GRPCServiceConnector(token, logs.NewLogsServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		return
	} else {
		defer func(conn *grpc.ClientConn) {
			err := conn.Close()
			if err != nil {
				log.Printf("Ошибка закрытия соединения: %v", err)
				errLogs := utils.SaveLogsError(ctx, clientLogs, "basemigration", "", err.Error())
				if errLogs != nil {
					log.Printf("Ошибка закрытия соединения: %v", err)
				}
			}
		}(conn)
	}

	// Создаем DSN для подключения к базе данных
	dsn := utils.DsnString(dbName)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Printf("Ошибка подключения к базе данных %s: %v", dbName, err)
		errLogs := utils.SaveLogsError(ctx, clientLogs, "basemigration", "", err.Error())
		if errLogs != nil {
			log.Printf("Ошибка закрытия соединения: %v", err)
		}
		return
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			errLogs := utils.SaveLogsError(ctx, clientLogs, "basemigration", "", err.Error())
			if errLogs != nil {
				log.Printf("Ошибка закрытия соединения: %v", err)
			}
		}
	}(db)

	// Выполняем миграцию с помощью функции Migration
	err = Migration(db, migratePath, dbName)
	if err != nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, dbName, "", err.Error())
		if errLogs != nil {
			errLogs := utils.SaveLogsError(ctx, clientLogs, "basemigration", "", err.Error())
			if errLogs != nil {
				log.Printf("Ошибка закрытия соединения: %v", err)
			}
			log.Printf("Ошибка сохранения логов: %v", err)
		}
		log.Printf("Ошибка миграции базы данных %s: %v", dbName, err)
	} else {
		*counter++
		//log.Printf("Миграция для базы данных %s успешно выполнена", dbName)
	}
}

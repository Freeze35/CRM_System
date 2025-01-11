package dbadminservice

import (
	"context"
	pbAdmin "crmSystem/proto/dbadmin"
	"crmSystem/utils"
	"fmt"
	"net/http"
	"os"
	"time"
)

type AdminServiceServer struct {
	pbAdmin.UnsafeDbAdminServiceServer
	connectionsMap *utils.MapConnectionsDB // Используем указатель
}

func NewGRPCDBAdminService(mapConnect *utils.MapConnectionsDB) *AdminServiceServer {
	return &AdminServiceServer{
		connectionsMap: mapConnect,
	}
}

/*func formUsersKeyRedis(req *pbAdmin.RegisterUsersRequest) string {
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

// rollbackAdminDB откатывает изменения в базе данных авторизации, удаляя пользователя из центральной и внутренней базы данных.
func rollbackAdminDB(dbConn *sql.DB, companyId, authUserId int) {
	// Начинаем откатную транзакцию.
	tx, err := dbConn.Begin()
	if err != nil {
		log.Printf("Ошибка при начале откатной транзакции: %v", err) // Логируем ошибку, если не удалось начать транзакцию.
		return                                                       // Завершаем выполнение функции.
	}

	// Удаляем пользователя из таблицы authusers по его ID.
	_, err = tx.Exec("DELETE FROM authusers WHERE id = $1", authUserId)
	if err != nil {
		tx.Rollback()                                           // Откатываем транзакцию в случае ошибки.
		log.Printf("Ошибка при удалении пользователя: %v", err) // Логируем ошибку.
		return                                                  // Завершаем выполнение функции.
	}

	// Фиксируем откат транзакции.
	err = tx.Commit()
	if err != nil {
		log.Printf("Ошибка при фиксации отката: %v", err) // Логируем ошибку, если фиксация не удалась.
	}
}*/

func (s AdminServiceServer) RegisterUsersInCompany(ctx context.Context, req *pbAdmin.RegisterUsersRequest) (*pbAdmin.RegisterUsersResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	dbName := req.DbName

	if dbName == "" {
		return &pbAdmin.RegisterUsersResponse{
			Message: "Имя базы данных компании не указано",
			Status:  http.StatusBadRequest,
		}, fmt.Errorf("имя базы данных компании не указано")
	}

	CompanyId := req.CompanyId

	authDBName := os.Getenv("DB_AUTH_NAME")
	dsn := utils.DsnString(authDBName)
	dbConn, err := s.connectionsMap.GetDb(dsn)
	if err != nil || dbConn == nil {
		return &pbAdmin.RegisterUsersResponse{
			Message: "Ошибка подключения к базе авторизации: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}, err
	}

	tx, err := dbConn.Begin()
	if err != nil {
		return &pbAdmin.RegisterUsersResponse{
			Message: "Ошибка при начале транзакции: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	dsnC := utils.DsnString(dbName)
	dbConnCompany, err := s.connectionsMap.GetDb(dsnC)
	if err != nil || dbConnCompany == nil {
		return &pbAdmin.RegisterUsersResponse{
			Message: "Ошибка подключения к базе компании: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}, err
	}

	txc, err := dbConnCompany.Begin()
	if err != nil {
		return &pbAdmin.RegisterUsersResponse{
			Message: "Ошибка при начале транзакции для компании: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}, err
	}
	defer func() {
		if err != nil {
			_ = txc.Rollback()
		}
	}()

	var registeredUsers []*pbAdmin.UserResponse

	// Пакетное добавление authusers
	authQuery := `
		INSERT INTO authusers (email, phone, password, company_id) 
		VALUES ($1, $2, $3, $4)
		RETURNING id`
	stmtAuth, err := tx.Prepare(authQuery)
	if err != nil {
		return &pbAdmin.RegisterUsersResponse{
			Message: "Ошибка подготовки запроса authusers: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}, err
	}
	defer stmtAuth.Close()

	for _, user := range req.Users {
		// Проверка на существование пользователя в таблице authusers
		var existingAuthUserId int
		err = tx.QueryRow("SELECT id FROM authusers WHERE email = $1 OR phone = $2", user.Email, user.Phone).Scan(&existingAuthUserId)
		if err == nil { // Пользователь уже существует
			var existingUserId string
			err = txc.QueryRow("SELECT id FROM users WHERE authId = $1 AND company_id = $2", existingAuthUserId, CompanyId).Scan(&existingUserId)
			if err == nil { // Пользователь существует и связан с компанией
				registeredUsers = append(registeredUsers, &pbAdmin.UserResponse{
					Email:    user.Email,
					Phone:    user.Phone,
					RoleId:   user.RoleId,
					Password: "default_password",
				})
				continue
			}

			// Если пользователь существует, но не связан с компанией, добавляем связь
			err = txc.QueryRow(
				"INSERT INTO users (rightsId, authId) VALUES ($1, $2) RETURNING id",
				user.RoleId, existingAuthUserId,
			).Scan(&existingUserId)
			if err != nil {
				return &pbAdmin.RegisterUsersResponse{
					Message: "Ошибка добавления связи пользователя с компанией: " + err.Error(),
					Status:  http.StatusInternalServerError,
				}, err
			}

			registeredUsers = append(registeredUsers, &pbAdmin.UserResponse{
				Email:    user.Email,
				Phone:    user.Phone,
				RoleId:   user.RoleId,
				Password: "default_password",
			})
			continue
		}

		// Если пользователь не существует, создаём его в таблице authusers
		var userId int
		err = stmtAuth.QueryRow(user.Email, user.Phone, "default_password", CompanyId).Scan(&userId)
		if err != nil {
			return &pbAdmin.RegisterUsersResponse{
				Message: "Ошибка создания пользователя: " + err.Error(),
				Status:  http.StatusInternalServerError,
			}, err
		}

		// Добавляем пользователя в таблицу users
		var newUserId string
		err = txc.QueryRow(
			"INSERT INTO users (rightsId, authId) VALUES ($1, $2) RETURNING id",
			user.RoleId, userId,
		).Scan(&newUserId)
		if err != nil {
			return &pbAdmin.RegisterUsersResponse{
				Message: "Ошибка добавления пользователя в компанию: " + err.Error(),
				Status:  http.StatusInternalServerError,
			}, err
		}

		registeredUsers = append(registeredUsers, &pbAdmin.UserResponse{
			Email:    user.Email,
			Phone:    user.Phone,
			RoleId:   user.RoleId,
			Password: "default_password",
		})
	}

	txErr := tx.Commit()
	txcErr := txc.Commit()
	if txErr != nil || txcErr != nil {
		return &pbAdmin.RegisterUsersResponse{
			Message: "Ошибка при фиксации транзакции: " + fmt.Sprintf("txErr: %v, txcErr: %v", txErr, txcErr),
			Status:  http.StatusInternalServerError,
		}, fmt.Errorf("commit errors: txErr=%v, txcErr=%v", txErr, txcErr)
	}

	return &pbAdmin.RegisterUsersResponse{
		Users:   registeredUsers,
		Message: "Пользователи успешно добавлены",
		Status:  http.StatusOK,
	}, nil
}

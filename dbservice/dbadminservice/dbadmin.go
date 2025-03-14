package dbadminservice

import (
	"context"
	pbAdmin "crmSystem/proto/dbadmin"
	"crmSystem/proto/logs"
	"crmSystem/utils"
	"database/sql"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log"
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

func (s AdminServiceServer) RegisterUsersInCompany(ctx context.Context, req *pbAdmin.RegisterUsersRequest) (*pbAdmin.RegisterUsersResponse, error) {

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "Не удалось получить метаданные из контекста")
	}

	token, err := utils.ExtractTokenFromContext(ctx)
	if err != nil {
		log.Printf("Не удалось извлечь токен для логирования: %v", err)
		return nil, status.Errorf(codes.Unauthenticated, "Не удалось извлечь токен для логирования")
	}

	// Устанавливаем соединение с gRPC сервером Logs
	clientLogs, err, conn := utils.GRPCServiceConnector(token, logs.NewLogsServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		return nil, status.Errorf(codes.Unauthenticated, "Не удалось создать соединение с сервером Logs")
	} else {
		defer func(conn *grpc.ClientConn) {
			err := conn.Close()
			if err != nil {
				log.Printf("Ошибка закрытия соединения: %v", err)
				errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
				if errLogs != nil {
					log.Printf("Ошибка закрытия соединения: %v", err)
				}
			}
		}(conn)
	}

	// Извлекаем DatabaseName из метаданных
	dbCheck := md["database"]
	if len(dbCheck) == 0 {
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", "database не найдена в метаданных")
		if errLogs != nil {
			log.Printf("database не найдена в метаданных: %v", err)
		}
		return nil, status.Errorf(codes.Unauthenticated, "database не найдена в метаданных")
	}
	database := md["database"][0]

	// Извлекаем UserId из метаданных
	uIdCheck := md["user-id"]
	if len(uIdCheck) == 0 {
		errLogs := utils.SaveLogsError(ctx, clientLogs, database, "", "userId не найдена в метаданных")
		if errLogs != nil {
			log.Printf("userId не найдена в метаданных: %v", err)
		}
		return nil, status.Errorf(codes.Unauthenticated, "userId не найдена в метаданных")
	}
	userId := md["user-id"][0]

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	CompanyId := req.CompanyId

	authDBName := os.Getenv("DB_AUTH_NAME")
	dsn := utils.DsnString(authDBName)
	dbConn, err := s.connectionsMap.GetDb(dsn)
	if err != nil || dbConn == nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, "Ошибка подключения к базе авторизации")
		if errLogs != nil {
			log.Printf("Ошибка подключения к базе авторизации: %v", err)
		}
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка подключения к базе авторизации: "))
	}

	tx, err := dbConn.Begin()
	if err != nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка при начале транзакции: %v", err)
		}
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка при начале транзакции: "+err.Error()))
	}
	defer func() {
		if err != nil {
			errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, err.Error())
			if errLogs != nil {
				log.Printf("Откат транзакции authDBName: %v", err)
			}
			_ = tx.Rollback()
		}
	}()

	dsnC := utils.DsnString(database)
	dbConnCompany, err := s.connectionsMap.GetDb(dsnC)
	if err != nil || dbConnCompany == nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка подключения к базе компании: %v", err)
		}
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка подключения к базе компании: "+err.Error()))
	}

	txc, err := dbConnCompany.Begin()
	if err != nil {
		errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка при начале транзакции для компании: %v", err)
		}
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка при начале транзакции для компании: "+err.Error()))
	}
	defer func() {
		if err != nil {
			errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, err.Error())
			if errLogs != nil {
				log.Printf("Откат транзакции компании: %v", err)
			}
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
		errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка подготовки запроса authusers: %v", err)
		}
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка подготовки запроса authusers: "+err.Error()))
	}
	defer func(stmtAuth *sql.Stmt) {
		err := stmtAuth.Close()
		if err != nil {
			errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, err.Error())
			if errLogs != nil {
				log.Printf("Ошибка закрытия соединения: %v", err)
			}
		}
	}(stmtAuth)

	for _, user := range req.Users {
		// Проверка на существование пользователя в таблице authusers
		var existingAuthUserId int
		err = tx.QueryRow("SELECT id FROM authusers WHERE email = $1 OR phone = $2", user.Email, user.Phone).Scan(&existingAuthUserId)
		if err == nil { // Пользователь уже существует
			var existingUserId string
			err = txc.QueryRow("SELECT id FROM users WHERE authId = $1", existingAuthUserId).Scan(&existingUserId)
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
				errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, err.Error())
				if errLogs != nil {
					log.Printf("Ошибка добавления связи пользователя с компанией: %v", err)
				}
				return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка добавления связи пользователя с компанией: "+err.Error()))
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
			errLogs := utils.SaveLogsError(ctx, clientLogs, database, "", err.Error())
			if errLogs != nil {
				log.Printf("Ошибка добавления связи пользователя с компанией: %v", err)
			}
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка создания пользователя: "+err.Error()))
		}

		// Добавляем пользователя в таблицу users
		var newUserId string
		err = txc.QueryRow(
			"INSERT INTO users (rightsId, authId) VALUES ($1, $2) RETURNING id",
			user.RoleId, userId,
		).Scan(&newUserId)
		if err != nil {
			errLogs := utils.SaveLogsError(ctx, clientLogs, database, "", err.Error())
			if errLogs != nil {
				log.Printf("Ошибка добавления связи пользователя с компанией: %v", err)
			}
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка добавления пользователя в компанию: "+err.Error()))
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
		errLogs := utils.SaveLogsError(ctx, clientLogs, database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка при фиксации транзакции: " + fmt.Sprintf("txErr: %v, txcErr: %v", txErr, txcErr))
		}
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Ошибка при фиксации транзакции: "+fmt.Sprintf("txErr: %v, txcErr: %v", txErr, txcErr)))
	}

	return &pbAdmin.RegisterUsersResponse{
		Users:   registeredUsers,
		Message: "Пользователи успешно добавлены",
	}, nil
}

package transport_rest

import (
	"context"
	"crmSystem/proto/dbauth"
	"crmSystem/proto/logs"
	"crmSystem/transport_rest/types"
	"crmSystem/utils"
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log"
	"net/http"
	"time"
)

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) InitRouter() http.Handler {
	r := mux.NewRouter()

	authRouts := r.PathPrefix("/auth").Subrouter()
	{
		authRouts.HandleFunc("/login", utils.RecoverMiddleware(h.Login)).Methods(http.MethodPost)
		authRouts.HandleFunc("/register", utils.RecoverMiddleware(h.Register)).Methods(http.MethodPost)
		authRouts.HandleFunc("/refresh", utils.RecoverMiddleware(h.RefreshToken)).Methods(http.MethodPost)
		authRouts.HandleFunc("/check", utils.RecoverMiddleware(h.CheckAuth)).Methods(http.MethodPost)
		/*books.HandleFunc("/{id:[0-9]+}", h.getBookByID).Methods(http.MethodGet)
		books.HandleFunc("/{id:[0-9]+}", h.deleteBook).Methods(http.MethodDelete)
		books.HandleFunc("/{id:[0-9]+}", h.updateBook).Methods(http.MethodPut)*/
	}

	// Обертка CORS
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{
			"http://localhost:3001",  // для локальной разработки
			"https://myfrontend.com", // если фронтенд на продакшн домене
			"https://localhost:3300", // если фронтенд на продакшн домене
		}), // Или укажите разрешенные домены
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type"}),
		handlers.AllowCredentials(),
	)(r)

	return corsHandler
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {

	//Генерация JWT токена
	token, err := utils.InternalJwtGenerator()
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не удалось создать токен ", err)
		return
	}

	ctx := context.Background()

	// Устанавливаем соединение с gRPC сервером Logs
	clientLogs, err, conn := utils.GRPCServiceConnector(token, logs.NewLogsServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusBadRequest, "Ошибка подключения", err)
		return
	} else {
		defer func(conn *grpc.ClientConn) {
			err := conn.Close()
			if err != nil {
				log.Printf("Ошибка закрытия соединения: %v", err)
				errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
				if errLogs != nil {
					log.Printf("Ошибка закрытия соединения: %v", err)
					return
				}
			}
		}(conn)
	}

	//Декодируем поступающий от клиента json
	var req types.LoginAuthRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Ошибка при декодировании данных", http.StatusBadRequest)
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
		if errLogs != nil {
			log.Printf("Ошибка закрытия соединения: %v", err)
		}
		return
	}

	// Создаем валидатор
	validate := validator.New()

	// Регистрируем кастомные валидаторы
	err = validate.RegisterValidation("phone", validatePhone)
	if err != nil {
		utils.CreateError(w, http.StatusBadRequest, "Ошибка проверки номера телефона", err)
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
		if errLogs != nil {
			log.Printf("Ошибка проверки номера телефона: %v", err)
		}
		return
	}

	// Регистрируем кастомный валидатор для пароля
	err = validate.RegisterValidation("password", validatePassword)
	if err != nil {
		utils.CreateError(w, http.StatusBadRequest, "Ошибка при проверке пароля", err)
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
		if errLogs != nil {
			log.Printf("Ошибка при проверке пароля: %v", err)
		}
		return
	}

	// Валидация структуры
	err = validate.Struct(req)
	if err != nil {
		// Если есть ошибки валидации, разбираем их и сразу отправляем ошибку
		validationErrors := err.(validator.ValidationErrors)
		for _, e := range validationErrors {
			// Немедленно возвращаем ошибку для каждого поля с ошибкой валидации
			errorMessage := fmt.Sprintf("Поле '%s' не прошло валидацию", e.Field())
			utils.CreateError(w, http.StatusBadRequest, "Ошибка валидации", fmt.Errorf(errorMessage))
			return
		}
	}

	// Устанавливаем соединение с gRPC сервером dbService
	client, err, conn := utils.GRPCServiceConnector(token, dbauth.NewDbAuthServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusBadRequest, "Ошибка подключения", err)
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
		if errLogs != nil {
			log.Printf("Ошибка подключения: %v", err)
		}
		return
	} else {
		defer func(conn *grpc.ClientConn) {
			err := conn.Close()
			if err != nil {
				errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
				if errLogs != nil {
					log.Printf("Ошибка закрытия соединени: %v", err)
				}
				log.Printf("Ошибка закрытия соединени")
				return
			}
		}(conn)
	}

	// Проводим авторизацию пользователя с запросом к dbservice
	response, responseStatus, err := loginUser(w, client, &req, token)
	if err != nil {
		utils.CreateError(w, responseStatus, "Ошибка на сервере", err)
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
		if errLogs != nil {
			log.Printf("Не корректная ошибка на сервере: %v", err)
		}
		return
	}

	// Если авторизация прошла успешно, выводим данные
	if err := utils.WriteJSON(w, http.StatusOK, response); err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не корректная ошибка на сервере", err)
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
		if errLogs != nil {
			log.Printf("Не корректная ошибка на сервере: %v", err)
		}
	}
}

func loginUser(w http.ResponseWriter, client dbauth.DbAuthServiceClient, req *types.LoginAuthRequest, token string) (response *types.LoginAuthResponse, responseStatus uint32, err error) {

	// Формируем запрос на вход в систему
	reqLogin := &dbauth.LoginDBRequest{
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password,
	}

	// Устанавливаем тайм-аут для контекста
	ctxWithMetadata, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Устанавливаем соединение с gRPC сервером Logs
	clientLogs, err, conn := utils.GRPCServiceConnector(token, logs.NewLogsServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusBadRequest, "Ошибка подключения", err)
		return
	} else {
		defer func(conn *grpc.ClientConn) {
			err := conn.Close()
			if err != nil {
				log.Printf("Ошибка закрытия соединения: %v", err)
				errLogs := utils.SaveLogsError(ctxWithMetadata, clientLogs, "", "", err.Error())
				if errLogs != nil {
					log.Printf("Ошибка закрытия соединения: %v", err)
				}
				return
			}
		}(conn)
	}

	// Заголовки из ответа
	header := metadata.MD{}

	// Выполняем gRPC вызов LoginDB, передавая указатель для получения заголовков
	resDB, err := client.LoginDB(ctxWithMetadata, reqLogin, grpc.Header(&header))
	if err != nil {
		// Получаем сообщение об ошибке
		errorMessage := status.Convert(err).Message()

		// Код ошибки
		code := status.Code(err)

		// Логика в зависимости от кода ошибки
		switch code {
		case codes.Unauthenticated:
			errLogs := utils.SaveLogsError(ctxWithMetadata, clientLogs, "", "", err.Error())
			if errLogs != nil {
				log.Printf("Ошибка подключения: %v", err)
			}
			return nil, http.StatusUnauthorized, fmt.Errorf("неавторизированный запрос : %s", errorMessage)

		default:
			errLogs := utils.SaveLogsError(ctxWithMetadata, clientLogs, "", "", err.Error())
			if errLogs != nil {
				log.Printf("Ошибка подключения: %v", err)
			}
			return nil, http.StatusInternalServerError, fmt.Errorf("%s", errorMessage)
		}
	}

	// Проверяем наличие метаданных в ответе
	database := header.Get("database")
	userID := header.Get("user-id")
	companyID := header.Get("company-id")

	if len(database) == 0 || len(userID) == 0 || len(companyID) == 0 {
		errLogs := utils.SaveLogsError(ctxWithMetadata, clientLogs, "", "", "отсутствуют необходимые метаданные")
		if errLogs != nil {
			log.Printf("отсутствуют необходимые метаданные: %v", err)
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("отсутствуют необходимые метаданные")
	}

	accessToken, err := utils.JwtGenerator(userID[0], "access")
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("не удалось сформировать access token %s", err)
	}

	refreshToken, err := utils.JwtGenerator(userID[0], "refresh")
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("не удалось сформировать refresh token %s", err)
	}

	// Устанавливаем Cookie
	utils.AddCookie(w, "access_token", accessToken)
	utils.AddCookie(w, "refresh_token", refreshToken)
	utils.AddCookie(w, "database", database[0])
	utils.AddCookie(w, "user-id", userID[0])
	utils.AddCookie(w, "company-id", companyID[0])

	response = &types.LoginAuthResponse{
		Message: resDB.Message,
	}
	return response, http.StatusOK, nil
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	// Максимальное ожидание ответа при ожидании регистрации 10 секунд
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	iternalToken, err := utils.InternalJwtGenerator()
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не удалось создать токен ", err)
		return
	}

	// Устанавливаем соединение с gRPC сервером Logs
	clientLogs, err, conn := utils.GRPCServiceConnector(iternalToken, logs.NewLogsServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusBadRequest, "Ошибка подключения", err)
		return
	} else {
		defer func(conn *grpc.ClientConn) {
			err := conn.Close()
			if err != nil {
				log.Printf("Ошибка закрытия соединения: %v", err)
				errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
				if errLogs != nil {
					log.Printf("Ошибка закрытия соединения: %v", err)
					return
				}
			}
		}(conn)
	}

	var req types.RegisterAuthRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		utils.CreateError(w, http.StatusBadRequest, "Ошибка при декодировании данных", err)
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
		if errLogs != nil {
			log.Printf("Ошибка при декодировании данных: %v", err)
		}
		return
	}

	// Создаем валидатор
	validate := validator.New()

	// Регистрируем кастомный валидатор для пароля
	err = validate.RegisterValidation("password", validatePassword)
	if err != nil {
		utils.CreateError(w, http.StatusBadRequest, "Ошибка при проверке пароля", err)
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
		if errLogs != nil {
			log.Printf("Ошибка при проверке пароля: %v", err)
		}
		return
	}

	err = validate.RegisterValidation("phone", validatePhone)
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Ошибка регистрации кастомного валидатора", err)
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
		if errLogs != nil {
			log.Printf("Ошибка регистрации кастомного валидатора: %v", err)
		}
		return
	}

	// Валидация структуры
	err = validate.Struct(req) // Исправление: используем req, а не пустую структуру
	if err != nil {
		// Если есть ошибки валидации, разбираем их и сразу отправляем ошибку
		validationErrors := err.(validator.ValidationErrors)
		for _, e := range validationErrors {
			// Немедленно возвращаем ошибку для каждого поля с ошибкой валидации
			errorMessage := fmt.Sprintf("Поле '%s' не прошло валидацию", e.Field())
			utils.CreateError(w, http.StatusBadRequest, "Ошибка валидации", fmt.Errorf(errorMessage))
			errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
			if errLogs != nil {
				log.Printf("Ошибка валидации: %v", err)
			}
			return
		}
	}

	// Устанавливаем соединение с gRPC сервером dbService
	client, err, conn := utils.GRPCServiceConnector(iternalToken, dbauth.NewDbAuthServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusBadRequest, "Ошибка подключения", err)
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
		if errLogs != nil {
			log.Printf("Ошибка подключения: %v", err)
		}
		return
	} else {
		defer func(conn *grpc.ClientConn) {
			err := conn.Close()
			if err != nil {
				errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
				if errLogs != nil {
					log.Printf("Ошибка закрытия подключения: %v", err)
				}
			}
		}(conn)
	}

	// Вызываем метод регистрации компании через gRPC
	response, responseStatus, err := callRegisterCompany(w, client, &req, ctx, clientLogs)
	if err != nil {
		utils.CreateError(w, responseStatus, "Ошибка регистрации компании", err)
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
		if errLogs != nil {
			log.Printf("Ошибка регистрации компании: %v", err)
		}
		return
	}

	// Если запрос успешно выполнен, возвращаем JSON-ответ
	if err := utils.WriteJSON(w, http.StatusOK, response); err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Ошибка записи ответа", err)
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
		if errLogs != nil {
			log.Printf("Ошибка записи ответа: %v", err)
		}
	}
}

func callRegisterCompany(w http.ResponseWriter, client dbauth.DbAuthServiceClient,
	req *types.RegisterAuthRequest, ctx context.Context, clientLogs logs.LogsServiceClient) (response *types.RegisterAuthResponse, responseStatus uint32, err error) {

	// Создаем контекст с тайм-аутом для запроса
	// В случае превышения порога ожидания с сервера в 10 секунд будет ошибка контекста.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Формируем запрос на регистрацию компании
	req1 := &dbauth.RegisterCompanyRequest{
		NameCompany: req.NameCompany,
		Address:     req.Address,
		Email:       req.Email,
		Phone:       req.Phone,
		Password:    req.Password,
	}

	// Заголовки из ответа
	header := metadata.MD{}

	// Выполняем gRPC вызов RegisterCompany
	resDB, err := client.RegisterCompany(ctx, req1, grpc.Header(&header))
	if err != nil {

		// Получаем сообщение об ошибке
		errorMessage := status.Convert(err).Message()

		// Код ошибки
		code := status.Code(err)

		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
		if errLogs != nil {
			log.Printf("Ошибка авторизации: %v", err)
		}

		// Логика в зависимости от кода ошибки
		switch code {
		case codes.Unauthenticated:
			return nil, http.StatusUnauthorized, fmt.Errorf(errorMessage)
		case codes.Unimplemented:
			return nil, http.StatusNotImplemented, fmt.Errorf(errorMessage)
		case codes.AlreadyExists:
			return nil, http.StatusConflict, fmt.Errorf(errorMessage)
		default:
			return nil, http.StatusInternalServerError, fmt.Errorf(errorMessage)
		}
	}

	/*// Обрабатываем ответ
	log.Printf("Ответ сервера: Message: %s, Database: %s, Status: %d", res.GetMessage(), res.GetDatabase(), res.GetStatus())*/

	//Получен ответ о логинизации от dbservice

	// Проверяем наличие метаданных в ответе
	database := header.Get("database")
	userID := header.Get("user-id")
	companyID := header.Get("company-id")

	if len(database) == 0 || len(userID) == 0 || len(companyID) == 0 {
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", "отсутствуют необходимые метаданные")
		if errLogs != nil {
			log.Printf("Отсутствуют необходимые метаданные: %v", err)
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("отсутствуют необходимые метаданные")
	}

	accessToken, err := utils.JwtGenerator(userID[0], "access")
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("не удалось сформировать access token: %s", err)
	}

	refreshToken, err := utils.JwtGenerator(userID[0], "access")
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("не удалось сформировать refresh token %s", err)
	}

	// Устанавливаем HttpOnly Cookie
	utils.AddCookie(w, "access_token", accessToken)
	utils.AddCookie(w, "refresh_token", refreshToken)
	utils.AddCookie(w, "database", database[0])
	utils.AddCookie(w, "user-id", userID[0])
	utils.AddCookie(w, "company-id", companyID[0])

	response = &types.RegisterAuthResponse{
		Message: resDB.Message,
	}
	return response, http.StatusOK, nil

}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	token, err := utils.InternalJwtGenerator()
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не удалось создать токен", err)
		return
	}

	// Устанавливаем соединение с gRPC сервером Logs
	clientLogs, err, conn := utils.GRPCServiceConnector(token, logs.NewLogsServiceClient)
	if err != nil {
		utils.CreateError(w, http.StatusBadRequest, "Ошибка подключения", err)
		log.Printf("Не удалось подключиться к серверу: %v", err)
		return
	}
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

	newAccessToken, err := utils.ValidateAndRefreshToken(w, r)
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не удалось создать токен", err)
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
		if errLogs != nil {
			return
		} // Логируем ошибку
		return
	}

	// Добавляем куку в заголовок ответа
	utils.AddCookie(w, "access_token", newAccessToken)

	// Возвращаем JSON-ответ об успешном обновлении токена
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]string{"message": "Токен успешно обновлён"}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		utils.CreateError(w, http.StatusBadRequest, "Ошибка подключения", err)
		errLogs := utils.SaveLogsError(ctx, clientLogs, "", "", err.Error())
		if errLogs != nil {

		} // Логируем ошибку
		return
	}
}

func (h *Handler) CheckAuth(w http.ResponseWriter, r *http.Request) {

	response := map[string]string{"message": "Проверка пройдена"}
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Printf("Ошибка записи ответа при проверке")
		return
	}
}

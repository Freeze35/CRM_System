package transport_rest

import (
	"context"
	"crmSystem/proto/dbauth"
	"crmSystem/transport_rest/types"
	"crmSystem/utils"
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
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

func (h *Handler) InitRouter() *mux.Router {
	r := mux.NewRouter()

	authRouts := r.PathPrefix("/auth").Subrouter()
	{
		authRouts.HandleFunc("/login", utils.RecoverMiddleware(h.Login)).Methods(http.MethodPost)
		authRouts.HandleFunc("/register", utils.RecoverMiddleware(h.Register)).Methods(http.MethodPost)
		/*books.HandleFunc("/{id:[0-9]+}", h.getBookByID).Methods(http.MethodGet)
		books.HandleFunc("/{id:[0-9]+}", h.deleteBook).Methods(http.MethodDelete)
		books.HandleFunc("/{id:[0-9]+}", h.updateBook).Methods(http.MethodPut)*/
	}

	return r
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {

	//Декодирум поступающий от клиента json
	var req types.LoginAuthRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Ошибка при декодировании данных", http.StatusBadRequest)
		return
	}

	// Создаем валидатор
	validate := validator.New()

	// Регистрируем кастомные валидаторы
	err := validate.RegisterValidation("phone", validatePhone)
	if err != nil {
		utils.CreateError(w, http.StatusBadRequest, "Ошибка проверки номера телефона", err)
		return
	}

	// Регистрируем кастомный валидатор для пароля
	err = validate.RegisterValidation("password", validatePassword)
	if err != nil {
		utils.CreateError(w, http.StatusBadRequest, "Ошибка при проверке пароля", err)
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

	//Генерация JWT токена
	token, err := utils.JwtGenerate()
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не удалось создать токен ", err)
	}

	// Устанавливаем соединение с gRPC сервером dbService
	client, err, conn := utils.GRPCServiceConnector(token, dbauth.NewDbAuthServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusBadRequest, "Ошибка подключения", err)
		return
	} else {
		defer conn.Close()
	}

	// Проводим авторизацию пользователя с запросом к dbservice
	response, responseStatus, err := loginUser(w, client, &req, token)
	if err != nil {
		utils.CreateError(w, responseStatus, "Не корректная ошибка на сервере", err)
		return
	}

	// Если авторизация прошла успешно, выводим данные
	if err := utils.WriteJSON(w, http.StatusOK, response); err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не корректная ошибка на сервере", err)
	}
}

func loginUser(w http.ResponseWriter, client dbauth.DbAuthServiceClient, req *types.LoginAuthRequest, token string) (response *types.LoginAuthResponse, responseStatus uint32, err error) {

	// Формируем запрос на вход в систему
	reqLogin := &dbauth.LoginDBRequest{
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password,
	}

	// Создаем метаданные с токеном
	md := metadata.Pairs("auth-token", token)

	// Создаем контекст с метаданными
	ctxWithMetadata := metadata.NewOutgoingContext(context.Background(), md)

	// Устанавливаем тайм-аут для контекста
	ctxWithMetadata, cancel := context.WithTimeout(ctxWithMetadata, time.Second*10)
	defer cancel()

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
			return nil, http.StatusUnauthorized, fmt.Errorf("неавторизированный запрос : %s", errorMessage)
		default:
			return nil, http.StatusInternalServerError, fmt.Errorf("неизвестная ошибка : %s", errorMessage)
		}
	}

	// Проверяем наличие метаданных в ответе
	database := header.Get("database")
	userID := header.Get("user-id")
	companyID := header.Get("company-id")

	if len(database) == 0 || len(userID) == 0 || len(companyID) == 0 {
		return nil, http.StatusInternalServerError, fmt.Errorf("отсутствуют необходимые метаданные")
	}

	// Устанавливаем HttpOnly Cookie
	utils.AddCookie(w, "auth-token", token)
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

	var req types.RegisterAuthRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		utils.CreateError(w, http.StatusBadRequest, "Ошибка при декодировании данных", err)
		return
	}

	// Создаем валидатор
	validate := validator.New()

	// Регистрируем кастомный валидатор для пароля
	err := validate.RegisterValidation("password", validatePassword)
	if err != nil {
		utils.CreateError(w, http.StatusBadRequest, "Ошибка при проверке пароля", err)
		return
	}

	err = validate.RegisterValidation("phone", validatePhone)
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Ошибка регистрации кастомного валидатора", err)
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
			return
		}
	}

	token, err := utils.JwtGenerate()
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не удалось создать токен ", err)
	}

	// Устанавливаем соединение с gRPC сервером dbService
	client, err, conn := utils.GRPCServiceConnector(token, dbauth.NewDbAuthServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusBadRequest, "Ошибка подключения", err)
		return
	} else {
		defer conn.Close()
	}

	// Вызываем метод регистрации компании через gRPC
	response, responseStatus, err := callRegisterCompany(w, client, &req, ctx, token)
	if err != nil {
		utils.CreateError(w, responseStatus, "Ошибка регистрации компании", err)
		return
	}

	// Если запрос успешно выполнен, возвращаем JSON-ответ
	if err := utils.WriteJSON(w, http.StatusOK, response); err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Ошибка записи ответа", err)
	}
}

func callRegisterCompany(w http.ResponseWriter, client dbauth.DbAuthServiceClient,
	req *types.RegisterAuthRequest, ctx context.Context, token string) (response *types.RegisterAuthResponse, responseStatus uint32, err error) {

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

		// Логика в зависимости от кода ошибки
		switch code {
		case codes.Unauthenticated:
			return nil, http.StatusUnauthorized, fmt.Errorf("неавторизированный запрос : %s", errorMessage)
		case codes.Unimplemented:
			return nil, http.StatusNotImplemented, fmt.Errorf("неавторизированный запрос : %s", errorMessage)
		case codes.AlreadyExists:
			return nil, http.StatusConflict, fmt.Errorf("неавторизированный запрос : %s", errorMessage)
		default:
			return nil, http.StatusInternalServerError, fmt.Errorf("внутреняя ошибка регистрации : %s", errorMessage)
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
		return nil, http.StatusInternalServerError, fmt.Errorf("отсутствуют необходимые метаданные")
	}

	// Устанавливаем HttpOnly Cookie
	utils.AddCookie(w, "auth-token", token)
	utils.AddCookie(w, "database", database[0])
	utils.AddCookie(w, "user-id", userID[0])
	utils.AddCookie(w, "company-id", companyID[0])

	response = &types.RegisterAuthResponse{
		Message: resDB.Message,
	}
	return response, http.StatusOK, nil

}

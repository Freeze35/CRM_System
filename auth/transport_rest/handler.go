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
	"log"
	"net/http"
	"strings"
	"time"
)

type AuthService interface {
	Login(ctx context.Context) error
	Auth(ctx context.Context) error
}

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) InitRouter() *mux.Router {
	r := mux.NewRouter()

	authRouts := r.PathPrefix("/auth").Subrouter()
	{
		authRouts.HandleFunc("/login", h.Login).Methods(http.MethodPost)
		authRouts.HandleFunc("/register", h.Register).Methods(http.MethodPost)
		/*books.HandleFunc("/{id:[0-9]+}", h.getBookByID).Methods(http.MethodGet)
		books.HandleFunc("/{id:[0-9]+}", h.deleteBook).Methods(http.MethodDelete)
		books.HandleFunc("/{id:[0-9]+}", h.updateBook).Methods(http.MethodPut)*/
	}

	return r
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
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

	// Устанавливаем соединение с gRPC сервером dbService
	client, err, conn := utils.GRPCServiceConnector(true, dbauth.NewDbAuthServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusBadRequest, "Ошибка подключения", err)
		return
	} else {
		defer conn.Close()
	}

	// Проводим авторизацию пользователя с запросом к dbservice
	response, err := loginUser(client, &req)
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не корректная ошибка на сервере", err)
		return
	}

	// Если авторизация прошла успешно, выводим данные
	if err := utils.WriteJSON(w, response.Status, response); err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не корректная ошибка на сервере", err)
	}
}

func loginUser(client dbauth.DbAuthServiceClient, req *types.LoginAuthRequest) (response *types.LoginAuthResponse, err error) {

	// Формируем запрос на регистрацию компании
	reqLogin := &dbauth.LoginDBRequest{
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password,
	}

	// Создаем контекст с тайм-аутом для запроса
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Выполняем gRPC вызов RegisterCompany
	resDB, err := client.LoginDB(ctx, reqLogin)

	if err != nil {

		//Проверка на ошибку не авторизованного JWT запроса
		authCheck := strings.Contains(err.Error(), "401")
		var message string
		var status uint32
		if authCheck {
			message = "Пользователь не предоставил авторизационный JWT токен. Ошибка 401"
			status = http.StatusUnauthorized
		} else {
			message = err.Error()
			status = http.StatusInternalServerError
		}

		response := &types.LoginAuthResponse{
			Message:  "Внутреняя ошибка логинизации: " + message,
			Database: "",
			Status:   status,
		}

		log.Printf("Ошибка при логинизации: %v", err)
		return response, nil
	}

	//Получен ответ о логинизации от dbservice
	token, err := utils.JwtGenerate()
	if err != nil || resDB.Status != http.StatusOK {

		response = &types.LoginAuthResponse{
			Message:   resDB.Message,
			Database:  resDB.Database,
			CompanyId: resDB.CompanyId,
			Token:     "",
			Status:    resDB.Status,
		}
		return response, nil
	} else {
		response = &types.LoginAuthResponse{
			Message:   resDB.Message,
			Database:  resDB.Database,
			CompanyId: resDB.CompanyId,
			Token:     token,
			Status:    resDB.Status,
		}
		return response, nil
	}

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

	// Устанавливаем соединение с gRPC сервером dbService
	client, err, conn := utils.GRPCServiceConnector(true, dbauth.NewDbAuthServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusBadRequest, "Ошибка подключения", err)
		return
	} else {
		defer conn.Close()
	}

	// Вызываем метод регистрации компании через gRPC
	response, err := callRegisterCompany(client, &req, ctx)
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Ошибка регистрации компании", err)
		return
	}

	// Если запрос успешно выполнен, возвращаем JSON-ответ
	if err := utils.WriteJSON(w, response.Status, response); err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Ошибка записи ответа", err)
	}
}

func callRegisterCompany(client dbauth.DbAuthServiceClient, req *types.RegisterAuthRequest, ctx context.Context) (response *types.RegisterAuthResponse, err error) {

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

	// Выполняем gRPC вызов RegisterCompany
	resDB, err := client.RegisterCompany(ctx, req1)
	if err != nil {

		//Проверка на ошибку не авторизованного JWT запроса
		authCheck := strings.Contains(err.Error(), "401")
		var message string
		var status uint32
		if authCheck {
			message = "Пользователь не предоставил авторизационный JWT токен. Ошибка 401"
			status = http.StatusUnauthorized
		} else {
			message = err.Error()
			status = http.StatusInternalServerError
		}

		response := &types.RegisterAuthResponse{
			Message:   "Внутреняя ошибка регистрации: " + message,
			Database:  "",
			CompanyId: "",
			Token:     "",
			Status:    status,
		}

		log.Printf("Ошибка при вызове RegisterCompany: %v", err)
		return response, nil
	}

	/*// Обрабатываем ответ
	log.Printf("Ответ сервера: Message: %s, Database: %s, Status: %d", res.GetMessage(), res.GetDatabase(), res.GetStatus())*/

	if resDB.Status == http.StatusOK {
		// Пример успешного ответа с генерированным токеном
		token, err := utils.JwtGenerate()
		if err != nil || resDB.Status != http.StatusOK {

			response := &types.RegisterAuthResponse{
				Message:   resDB.Message,
				Database:  resDB.Database,
				CompanyId: resDB.CompanyId,
				Token:     "",
				Status:    http.StatusOK,
			}
			return response, nil
		} else {
			response := &types.RegisterAuthResponse{
				Message:   resDB.Message,
				Database:  resDB.Database,
				CompanyId: resDB.CompanyId,
				Token:     token,
				Status:    http.StatusOK,
			}
			return response, nil
		}

	} else {
		response := &types.RegisterAuthResponse{
			Message:   resDB.Message,
			Database:  "",
			CompanyId: "",
			Token:     "",
			Status:    resDB.Status,
		}
		return response, nil
	}
}

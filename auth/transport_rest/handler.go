package transport_rest

import (
	"context"
	"crmSystem/proto/dbservice"
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

	books := r.PathPrefix("/auth").Subrouter()
	{
		books.HandleFunc("/login", h.Login).Methods(http.MethodPost)
		books.HandleFunc("/authin", h.AuthIn).Methods(http.MethodGet)
		/*books.HandleFunc("/{id:[0-9]+}", h.getBookByID).Methods(http.MethodGet)
		books.HandleFunc("/{id:[0-9]+}", h.deleteBook).Methods(http.MethodDelete)
		books.HandleFunc("/{id:[0-9]+}", h.updateBook).Methods(http.MethodPut)*/
	}

	return r
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	//id, err := getIdFromRequest(r)
	/*if err != nil {
		log.Println("getBookByID() error:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	book, err := h.booksService.GetByID(context.TODO(), id)
	if err != nil {
		if errors.Is(err, domain.ErrBookNotFound) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Println("getBookByID() error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response, err := json.Marshal(book)
	if err != nil {
		log.Println("getBookByID() error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")*/
	var req types.LoginAuthRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Ошибка при декодировании данных", http.StatusBadRequest)
		return
	}

	// Создаем валидатор
	validate := validator.New()

	// Регистрируем кастомные валидаторы
	err := validate.RegisterValidation("custom_email", validateEmail)
	if err != nil {
		http.Error(w, "Ошибка проверки имени почты", http.StatusBadRequest)
		return
	}
	err = validate.RegisterValidation("custom_phone", validatePhone)
	if err != nil {
		http.Error(w, "Ошибка проверки имени номера телефона", http.StatusBadRequest)
		return
	}

	// Валидация структуры
	err = validate.Struct(types.LoginAuthRequest{})
	if err != nil {
		// Если есть ошибки валидации
		http.Error(w, fmt.Sprintf("Ошибка валидации: %v", err), http.StatusBadRequest)
		return
	}

	// Устанавливаем соединение с gRPC сервером dbService
	client, err, conn := utils.DbServiceConnector(true)
	defer conn.Close()

	if err != nil {
		response := &types.LoginAuthResponse{
			Message:       "Не удалось подключиться к серверу: " + err.Error(),
			Database:      "",
			UserCompanyId: "",
			Token:         "",
			Status:        http.StatusInternalServerError,
		}
		if err := utils.WriteJSON(w, response.Status, response); err != nil {
			utils.WriteError(w, http.StatusInternalServerError, err)
		}
		return
	}

	// Проводим авторизацию пользователя с запросом к dbservice
	response, err := loginUser(client, &req)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	// Если валидация прошла успешно, выводим данные
	if err := utils.WriteJSON(w, response.Status, response); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
	}
}

func loginUser(client dbservice.DbServiceClient, req *types.LoginAuthRequest) (response *types.LoginAuthResponse, err error) {

	// Формируем запрос на регистрацию компании
	reqLogin := &dbservice.LoginDBRequest{
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
	if err != nil {

		fmt.Printf("Ошибка генерации токена: %s", err)
		response = &types.LoginAuthResponse{
			Message:       resDB.Message,
			Database:      resDB.Database,
			UserCompanyId: resDB.UserCompanyId,
			Token:         "",
			Status:        resDB.Status,
		}
		return response, nil
	} else {
		response = &types.LoginAuthResponse{
			Message:       resDB.Message,
			Database:      resDB.Database,
			UserCompanyId: resDB.UserCompanyId,
			Token:         token,
			Status:        resDB.Status,
		}
		return response, nil
	}

}

func (h *Handler) AuthIn(w http.ResponseWriter, r *http.Request) {
	log.Printf("Restro")
	response, _ := json.Marshal("dd")
	w.Write(response)
	/*id, err := getIdFromRequest(r)
	if err != nil {
		log.Println("getBookByID() error:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	book, err := h.booksService.GetByID(context.TODO(), id)
	if err != nil {
		if errors.Is(err, domain.ErrBookNotFound) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Println("getBookByID() error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response, err := json.Marshal(book)
	if err != nil {
		log.Println("getBookByID() error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(response)*/
}

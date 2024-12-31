package transport_rest

import (
	"context"
	"crmSystem/proto/email-service"
	"crmSystem/transport_rest/types"
	"crmSystem/utils"
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	"net/http"
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

	books := r.PathPrefix("/admin").Subrouter()
	{
		books.HandleFunc("/adduser", h.AddUser).Methods(http.MethodPost)
		/*books.HandleFunc("/{id:[0-9]+}", h.getBookByID).Methods(http.MethodGet)
		books.HandleFunc("/{id:[0-9]+}", h.deleteBook).Methods(http.MethodDelete)
		books.HandleFunc("/{id:[0-9]+}", h.updateBook).Methods(http.MethodPut)*/
	}

	return r
}

func (h *Handler) AddUser(w http.ResponseWriter, r *http.Request) {

	// Устанавливаем соединение с gRPC сервером mailService
	client, err, conn := utils.GRPCServiceConnector(true, email.NewEmailServiceClient)
	defer conn.Close()

	var req types.SendEmailRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Ошибка при декодировании данных", http.StatusBadRequest)
		return
	}

	// Создаем валидатор
	validate := validator.New()

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

	response, err := sendToEmailUser(client, &req)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	// Если валидация прошла успешно, выводим данные
	if err := utils.WriteJSON(w, response.Status, response); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
	}

	if err != nil {
		response := &types.SendEmailResponse{
			Message: "Не удалось отправить сообщение: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}
		if err := utils.WriteJSON(w, response.Status, response); err != nil {
			utils.WriteError(w, http.StatusInternalServerError, err)
		}
		return
	}
}

func sendToEmailUser(client email.EmailServiceClient, req *types.SendEmailRequest) (response *email.SendEmailResponse, err error) {
	// Выполняем gRPC вызов RegisterCompany

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Формируем запрос на регистрацию компании
	reqMail := &email.SendEmailRequest{
		Recipient: req.Recipient,
		Subject:   req.Subject,
		Body:      req.Body,
	}

	resDB, err := client.SendEmail(ctx, reqMail)

	response = &email.SendEmailResponse{
		Message: resDB.Message,
		Status:  resDB.Status,
	}

	return response, err
}

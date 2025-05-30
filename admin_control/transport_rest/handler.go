package transport_rest

import (
	"context"
	"crmSystem/proto/dbadmin"
	"crmSystem/proto/email-service"
	"crmSystem/proto/logs"
	"crmSystem/transport_rest/types"
	"crmSystem/utils"
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) InitRouter() *mux.Router {
	r := mux.NewRouter()

	adminRouts := r.PathPrefix("/admin").Subrouter()
	{
		adminRouts.HandleFunc("/addusers", utils.RecoverMiddleware(h.AddUsers)).Methods(http.MethodPost)
	}

	return r
}

func (h *Handler) AddUsers(w http.ResponseWriter, r *http.Request) {

	// Получаем cookie с именами
	token := utils.GetFromCookies(w, r, "access_token")
	if token == "" {
		utils.CreateError(w, http.StatusBadRequest, "Токен не найден", fmt.Errorf(""))
		return
	}

	userId := utils.GetFromCookies(w, r, "user-id")
	if userId == "" {
		utils.CreateError(w, http.StatusBadRequest, "user-id не найден", fmt.Errorf(""))
		return
	}

	database := utils.GetFromCookies(w, r, "database")
	if database == "" {
		utils.CreateError(w, http.StatusBadRequest, "database не найдена", fmt.Errorf(""))
		return
	}

	md := metadata.Pairs(
		"user-id", userId,
		"database", database,
	)

	ctxWithMetadata := metadata.NewOutgoingContext(context.Background(), md)

	// Устанавливаем соединение с gRPC сервером Logs
	clientLogs, err, conn := utils.GRPCServiceConnector(token, logs.NewLogsServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusBadRequest, "Ошибка подключения", err)
		errLogs := utils.SaveLogsError(ctxWithMetadata, clientLogs, database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка закрытия соединения: %v", err)
		}
		return
	} else {
		defer func(conn *grpc.ClientConn) {
			err := conn.Close()
			if err != nil {
				log.Printf("Ошибка закрытия соединения: %v", err)
				errLogs := utils.SaveLogsError(ctxWithMetadata, clientLogs, database, userId, err.Error())
				if errLogs != nil {
					log.Printf("Ошибка закрытия соединения: %v", err)
				}
				return
			}
		}(conn)
	}

	var reqUsers types.RegisterUsersRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&reqUsers); err != nil {
		utils.CreateError(w, http.StatusBadRequest, "Ошибка при декодировании данных", err)
		errLogs := utils.SaveLogsError(ctxWithMetadata, clientLogs, database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка при декодировании данных: %v", err)
		}

		return
	}

	// Создаем валидатор
	validate := validator.New()

	// Валидация структуры
	err = validate.Struct(reqUsers) // Исправление: используем req, а не пустую структуру
	if err != nil {
		// Если есть ошибки валидации, разбираем их и сразу отправляем ошибку
		validationErrors := err.(validator.ValidationErrors)
		for _, e := range validationErrors {
			// Немедленно возвращаем ошибку для каждого поля с ошибкой валидации
			errorMessage := fmt.Sprintf("Поле '%s' не прошло валидацию", e.Field())
			utils.CreateError(w, http.StatusBadRequest, "Ошибка валидации", fmt.Errorf(errorMessage))

			errLogs := utils.SaveLogsError(ctxWithMetadata, clientLogs, database, userId, err.Error())
			if errLogs != nil {
				log.Printf("Ошибка валидации: %v", err)
			}
			return
		}
	}

	// Устанавливаем соединение с gRPC сервером dbService
	client, err, conn := utils.GRPCServiceConnector(token, dbadmin.NewDbAdminServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusBadRequest, "Ошибка подключения", err)
		errLogs := utils.SaveLogsError(ctxWithMetadata, clientLogs, database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Не удалось передать логи ошибки: %v", err)
		}
		return
	} else {
		defer func(conn *grpc.ClientConn) {
			err := conn.Close()
			if err != nil {
				log.Printf("Ошибка закрытия соединения: %v", err)
				errLogs := utils.SaveLogsError(ctxWithMetadata, clientLogs, database, userId, err.Error())
				if errLogs != nil {
					log.Printf("Ошибка закрытия соединения: %v", err)
				}
				return
			}
		}(conn)
	}

	//Вызываем регистрацию пользователя на dbservice
	response, err := CallAddUsers(ctxWithMetadata, client, &reqUsers, clientLogs, database, userId)
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не корректная ошибка на сервере.", err)
		errLogs := utils.SaveLogsError(ctxWithMetadata, clientLogs, database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Не корректная ошибка на сервере: %v", err)
		}
		return
	}

	// Устанавливаем соединение с gRPC сервером dbService
	clientEmail, err, conn := utils.GRPCServiceConnector(token, email.NewEmailServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusBadRequest, "Ошибка подключения", err)

		errLogs := utils.SaveLogsError(ctxWithMetadata, clientLogs, database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка подключения: %v", err)
		}

		return
	} else {
		defer func(conn *grpc.ClientConn) {
			err := conn.Close()
			if err != nil {
				log.Printf("Ошибка закрытия соединения: %v", err)
				errLogs := utils.SaveLogsError(ctxWithMetadata, clientLogs, database, userId, err.Error())
				if errLogs != nil {
					log.Printf("Ошибка закрытия соединения: %v", err)
				}
				return
			}
		}(conn)
	}

	// Подготовим список успешных и неуспешных отправок
	var successCount, failureCount int
	var failureMessages []string

	// Отправка на почту всем пользователям
	for _, user := range response.Users {
		mailRequest := types.SendEmailRequest{
			Email:   user.Email, // используем email текущего пользователя
			Message: "Welcome to our service! FROM PETR",
			Body: fmt.Sprintf(
				`Hello %s,

				Thank you for signing up for our service! We are excited to have you on board.
				
				Here are your login details:
				- **Login**: %s
				- **Password**: %s
				
				If you have any questions, feel free to contact our support team.
				
				Best regards,
				The Team at Our Service`,
				user.Email, user.Email, user.Password),
		}

		// Отправляем письмо
		_, err := SendToEmailUser(clientEmail, &mailRequest)
		if err != nil {
			failureCount++
			failureMessages = append(failureMessages, "Failed to send email to "+user.Email+": "+err.Error())

			errLogs := utils.SaveLogsError(ctxWithMetadata, clientLogs, database, userId, err.Error())
			if errLogs != nil {
				log.Printf("Failed to send email to %s: %v", user.Email, err.Error())
			}
			continue
		}

		// Увеличиваем счетчик успешных отправок
		successCount++
	}

	// Преобразуем список ошибок в одну строку (если есть ошибки)
	var failuresString string
	if len(failureMessages) > 0 {
		failuresString = strings.Join(failureMessages, "\n")
	}

	// Формируем итоговый ответ
	var responseMessage string
	if failureCount > 0 {
		responseMessage = fmt.Sprintf("Successfully sent to %d users, failed for %d users.", successCount, failureCount)
	} else {
		responseMessage = fmt.Sprintf("Successfully sent to all %d users.", successCount)
	}

	// Ответ с результатом отправки
	sendMessageResponse := &types.SendEmailResponse{
		Message:  responseMessage,
		Failures: failuresString, // Если ошибки есть, передаем их как строку
	}

	if err := utils.WriteJSON(w, http.StatusOK, sendMessageResponse); err != nil {

		utils.CreateError(w, http.StatusInternalServerError, "Не корректная ошибка на сервере.", err)
		errLogs := utils.SaveLogsError(ctxWithMetadata, clientLogs, database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Ошибка подключения: %v", err)
		}

		return
	}
}

func TransformUsersConcurrently(users []*types.User) []*dbadmin.User {
	// Канал для передачи преобразованных пользователей
	resultChan := make(chan *dbadmin.User, len(users))

	// Горутина для обработки каждого пользователя
	var wg sync.WaitGroup
	for _, user := range users {
		wg.Add(1)
		go func(u *types.User) {
			defer wg.Done()
			resultChan <- &dbadmin.User{
				Email:  u.Email,
				Phone:  u.Phone,
				RoleId: u.RoleId,
			}
		}(user)
	}

	// Закрываем канал после завершения всех горутин
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Собираем результаты из канала
	var transformedUsers []*dbadmin.User
	for user := range resultChan {
		transformedUsers = append(transformedUsers, user)
	}

	return transformedUsers
}

func CallAddUsers(ctxWithMetadata context.Context, client dbadmin.DbAdminServiceClient, req *types.RegisterUsersRequest,
	clientLogs logs.LogsServiceClient, database string, userId string) (response *dbadmin.RegisterUsersResponse, err error) {

	reqRegisterUsers := &dbadmin.RegisterUsersRequest{
		CompanyId: req.CompanyId,
		Users:     TransformUsersConcurrently(req.Users),
	}

	// Создаем контекст с тайм-аутом для запроса
	ctxWithMetadata, cancel := context.WithTimeout(ctxWithMetadata, time.Second*10)
	defer cancel()

	// Выполняем gRPC вызов RegisterCompanyq
	resDB, err := client.RegisterUsersInCompany(ctxWithMetadata, reqRegisterUsers)

	if err != nil {
		log.Printf("Ошибка при добавлении пользователя: %v", err)
		errLogs := utils.SaveLogsError(ctxWithMetadata, clientLogs, database, userId, err.Error())
		if errLogs != nil {
			log.Printf("Не удалось выполнить gRPC вызов: %v", err)
		}

		return nil, fmt.Errorf("Не удалось выполнить gRPC вызов: %w", err)

		/*log.Printf("Ошибка при добавлении пользователя: %v", err)
		return response, nil*/
	}

	response = &dbadmin.RegisterUsersResponse{
		Message: resDB.Message,
		Users:   resDB.Users,
	}
	return response, nil

}

func SendToEmailUser(client email.EmailServiceClient, req *types.SendEmailRequest) (response *email.SendEmailResponse, err error) {
	// Выполняем gRPC вызов RegisterCompany

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Формируем запрос на регистрацию компании
	reqMail := &email.SendEmailRequest{
		Email:   req.Email,
		Message: req.Message,
		Body:    req.Body,
	}

	resDB, err := client.SendEmail(ctx, reqMail)

	response = &email.SendEmailResponse{
		Message: resDB.Message,
	}

	return response, err
}

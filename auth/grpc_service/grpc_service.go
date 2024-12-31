package grpc_service

import (
	"context"
	"crmSystem/proto/auth"
	"crmSystem/proto/dbservice"
	"crmSystem/utils"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type AuthServiceServer struct {
	auth.UnimplementedAuthServiceServer
}

func NewGRPCService() *AuthServiceServer {
	return &AuthServiceServer{}
}

func callRegisterCompany(client dbservice.DbServiceClient, req *auth.RegisterAuthRequest, ctx context.Context) (response *auth.RegisterAuthResponse, err error) {

	// Создаем контекст с тайм-аутом для запроса
	// В случае превышения порога ожидания с сервера в 10 секунд будет ошибка контекста.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Формируем запрос на регистрацию компании
	req1 := &dbservice.RegisterCompanyRequest{
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

		response := &auth.RegisterAuthResponse{
			Message:       "Внутреняя ошибка регистрации: " + message,
			Database:      "",
			UserCompanyId: "",
			Token:         "",
			Status:        status,
		}

		log.Printf("Ошибка при вызове RegisterCompany: %v", err)
		return response, nil
	}

	/*// Обрабатываем ответ
	log.Printf("Ответ сервера: Message: %s, Database: %s, Status: %d", res.GetMessage(), res.GetDatabase(), res.GetStatus())*/

	if resDB.Status == http.StatusOK {
		// Пример успешного ответа с генерированным токеном
		token, err := utils.JwtGenerate()
		if err != nil {

			fmt.Sprintf("Ошибка генерации токена: %s", err)
			response := &auth.RegisterAuthResponse{
				Message:       resDB.Message,
				Database:      resDB.Database,
				UserCompanyId: resDB.UserCompanyId,
				Token:         "",
				Status:        http.StatusOK,
			}
			return response, nil
		} else {
			response := &auth.RegisterAuthResponse{
				Message:       resDB.Message,
				Database:      resDB.Database,
				UserCompanyId: resDB.UserCompanyId,
				Token:         token,
				Status:        http.StatusOK,
			}
			return response, nil
		}

	} else {
		response := &auth.RegisterAuthResponse{
			Message:       "Внутренняя ошибка создания компании : " + resDB.Message,
			Database:      "",
			UserCompanyId: "",
			Token:         "",
			Status:        resDB.Status,
		}
		return response, nil
	}
}

// Register Реализация метода Register, для регистрации организации и пользователя как администратора
func (s *AuthServiceServer) Register(ctx context.Context, req *auth.RegisterAuthRequest) (*auth.RegisterAuthResponse, error) {

	//dbServicePath := os.Getenv("DB_SERVER_URL") //Требуется поменять соединение в случае,
	//разделения на отдельные докер соединения (то есть без docker-compose)

	// Устанавливаем тайм-аут для соединения
	/*ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()*/

	// Устанавливаем соединение с gRPC сервером dbService
	client, err, conn := utils.GRPCServiceConnector(true, dbservice.NewDbServiceClient)
	defer conn.Close()
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		if err != nil {
			response := &auth.RegisterAuthResponse{
				Message:       "Не удалось подключиться к серверу : " + err.Error(),
				Database:      "",
				UserCompanyId: "",
				Token:         "",
				Status:        http.StatusInternalServerError,
			}
			return response, err
		}
	}
	response, err := callRegisterCompany(client, req, ctx)
	if err != nil {
		response := &auth.RegisterAuthResponse{
			Message:       "Внутренняя ошибка создания компании : " + err.Error(),
			Database:      "",
			UserCompanyId: "",
			Token:         "",
			Status:        http.StatusInternalServerError,
		}
		return response, err
	}

	return response, nil
}

func loginUser(client dbservice.DbServiceClient, req *auth.LoginAuthRequest) (response *auth.LoginAuthResponse, err error) {

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

		response := &auth.LoginAuthResponse{
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

		fmt.Sprintf("Ошибка генерации токена: %s", err)
		response = &auth.LoginAuthResponse{
			Message:       resDB.Message,
			Database:      resDB.Database,
			UserCompanyId: resDB.UserCompanyId,
			Token:         "",
			Status:        resDB.Status,
		}
		return response, nil
	} else {
		response = &auth.LoginAuthResponse{
			Message:       resDB.Message,
			Database:      resDB.Database,
			UserCompanyId: resDB.UserCompanyId,
			Token:         token,
			Status:        resDB.Status,
		}
		return response, nil
	}

}

// Реализация метода Login, для авторизации уже зарегистрированного пользователя в AutorizationDB
func (s *AuthServiceServer) Login(_ context.Context, req *auth.LoginAuthRequest) (*auth.LoginAuthResponse, error) {

	// Устанавливаем соединение с gRPC сервером dbService
	client, err, conn := utils.GRPCServiceConnector(true, dbservice.NewDbServiceClient)
	defer conn.Close()

	if err != nil {
		response := &auth.LoginAuthResponse{
			Message:       "Не удалось подключиться к серверу: " + err.Error(),
			Database:      "",
			UserCompanyId: "",
			Token:         "",
			Status:        http.StatusInternalServerError,
		}
		return response, err
	}

	//Проводим авторизацию пользователя с запросом к dbservice
	response, err := loginUser(client, req)
	if err != nil {
		return nil, err
	}

	return response, nil
}

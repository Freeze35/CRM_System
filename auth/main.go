package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	auth "testAuth/proto/auth"
	dbservice "testAuth/proto/dbservice"
	"testAuth/utils"
	"time"
)

type AuthServiceServer struct {
	auth.UnimplementedAuthServiceServer
}

type DBServiceServer struct {
	dbservice.UnimplementedDbServiceServer
}

/*func registerHandler(w http.ResponseWriter, r *http.Request) {
	// Структура для JSON-данных
	type RegisterRequest struct {
		Username  string `json:"username"`
		Password  string `json:"password"`
		Address   string `json:"address"`
		CompanyDB string `json:"company_db"`
	}

	// Parse JSON
	var req RegisterRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	// Создание новой базы данных компании
	resp, err := http.Post(fmt.Sprintf("%s/%s/create-db", os.Getenv("DB_SERVER_URL"), os.Getenv("DB_SERVICE_NAME")), "", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, "Ошибка создания базы данных компании", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Читаем тело ответа после создания базы данных
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Ошибка при чтении тела ответа", http.StatusInternalServerError)
		return
	}

	// Выводим тело ответа с названием новой базы данных
	dbName := string(body)

	// Структура для JSON-данных
	type dbStruct struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		DbName  string `json:"dbname"`
	}

	dbJson := dbStruct{
		Name:    req.CompanyDB,
		Address: req.Address,
		DbName:  dbName,
	}

	// Регистрация организации в базе компании
	_, err = utils.SendPostRequest(
		fmt.Sprintf("%s/%s/register", os.Getenv("DB_SERVER_URL"), os.Getenv("DB_SERVICE_NAME")),
		dbJson)
	if err != nil {
		http.Error(w, "Ошибка регистрации в базе компании", http.StatusInternalServerError)
		return
	}

	//генерация токина для ответа авторизованного пользователя
	token, err := utils.GenerateToken(req.Username)
	if err != nil {
		http.Error(w, "Ошибка генерации токена", http.StatusInternalServerError)
		return
	}

	// Возврат успешного JSON-ответа
	response := map[string]string{
		"message":  "Регистрация успешна в обеих базах данных",
		"database": "dbName",
		"token":    token,
	}
	WriteJSON(w, http.StatusOK, response)
}*/

/*func getTest(w http.ResponseWriter, r *http.Request) {
	// Здесь вы можете выполнить нужные действия и вернуть ответ
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Тест успешен!"))
}*/

/*func loginHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	resp, err := http.Post(fmt.Sprintf("%s/login?username=%s&password=%s", os.Getenv("DB_SERVER_URL"), username, password), "", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, "Ошибка входа", http.StatusUnauthorized)
		return
	}

	defer resp.Body.Close()

	token, err := utils.GenerateToken(username)
	if err != nil {
		http.Error(w, "Ошибка генерации токена", http.StatusInternalServerError)
		return
	}

	// Возвращаем токен клиенту
	response := map[string]string{
		"message": "Успешный вход",
		"token":   token,
	}
	WriteJSON(w, http.StatusOK, response)
}*/

/*func usersHandler(w http.ResponseWriter, r *http.Request) {
	// Обработка запроса списка пользователей
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	token, err := utils.JwtGenerate()
	if err != nil {
		fmt.Sprintf("Ошибка: %s", err)
	}
	response := map[string]string{
		"token": token,
	}
	WriteJSON(w, http.StatusOK, response)
}*/

func callRegisterCompany(client dbservice.DbServiceClient, req *auth.RegisterAuthRequest) (response *auth.RegisterAuthResponse, err error) {
	// Создаем контекст с тайм-аутом для запроса
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
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
		response := &auth.RegisterAuthResponse{
			Message:  "Внутреняя ошибка регистрации: " + err.Error(),
			Database: "",
			Token:    "",
			Status:   http.StatusInternalServerError,
		}

		log.Printf("Ошибка при вызове RegisterCompany: %v", err)
		return response, nil
	}

	/*// Обрабатываем ответ
	log.Printf("Ответ сервера: Message: %s, Database: %s, Status: %d", res.GetMessage(), res.GetDatabase(), res.GetStatus())*/

	if resDB.Status == http.StatusOK {
		// Пример успешного ответа с сгенерированным токеном
		token, err := utils.JwtGenerate()
		if err != nil {
			fmt.Sprintf("Ошибка: %s", err)
		}
		response := &auth.RegisterAuthResponse{
			Message:  resDB.Message,
			Database: resDB.Database,
			Token:    token,
			Status:   http.StatusOK,
		}
		return response, nil
	} else {
		response := &auth.RegisterAuthResponse{
			Message:  "Внутренняя ошибка создания компании : " + resDB.Message,
			Database: "",
			Token:    "",
			Status:   uint32(resDB.Status),
		}
		return response, nil
	}
}

// Register Реализация метода Register
func (s *AuthServiceServer) Register(ctx context.Context, req *auth.RegisterAuthRequest) (*auth.RegisterAuthResponse, error) {
	log.Printf("Получен запрос на регистрацию пользователя: %v", req.Email)

	//dbServicePath := os.Getenv("DB_SERVER_URL") //Требуется поменять соединение в случае,
	//разделения на отдельные докер соединения (то есть без docker-compose)

	// Устанавливаем тайм-аут для соединения
	/*ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()*/

	// Устанавливаем соединение с gRPC сервером dbService
	client, err, conn := utils.DbServiceConnector()
	defer conn.Close()
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)

		return nil, err
	}
	response, err := callRegisterCompany(client, req)
	if err != nil {
		return nil, err
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	// Выполняем gRPC вызов RegisterCompany
	resDB, err := client.LoginDB(ctx, reqLogin)

	if err != nil {
		response := &auth.LoginAuthResponse{
			Message:  "Внутреняя ошибка логинизации: " + err.Error(),
			Database: "",
			Status:   http.StatusInternalServerError,
		}

		log.Printf("Ошибка при логинизации: %v", err)
		return response, nil
	}

	response = &auth.LoginAuthResponse{
		Message:  resDB.Message,
		Database: resDB.Database,
		Token:    "",
		Status:   resDB.Status,
	}
	return response, nil
}

func (s *AuthServiceServer) Login(_ context.Context, req *auth.LoginAuthRequest) (*auth.LoginAuthResponse, error) {

	// Устанавливаем соединение с gRPC сервером dbService
	client, err, conn := utils.DbServiceConnector()
	defer conn.Close()

	if err != nil {
		return nil, err
	}

	response, err := loginUser(client, req)
	if err != nil {
		return nil, err
	}

	return response, nil
}

const (
	serverCertFile   = "sslkeys/server.pem"
	serverKeyFile    = "sslkeys/server.key"
	clientCACertFile = "sslkeys/ca.crt"
)

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load certificate of the CA who signed client's certificate
	pemClientCA, err := ioutil.ReadFile(clientCACertFile)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemClientCA) {
		return nil, fmt.Errorf("failed to add client CA's certificate")
	}

	// Load server's certificate and private key
	serverCert, err := tls.LoadX509KeyPair(serverCertFile, serverKeyFile)
	if err != nil {
		return nil, err
	}

	// Create the credentials and return it
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}

	return credentials.NewTLS(config), nil
}

func main() {
	// Инициализируем TCP соединение для gRPC сервера

	port := os.Getenv("AUTH_SERVICE_PORT")

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Не удалось запустить сервер: %v", err)
	}

	/*// Загрузка TLS-учетных данных
	certFile := "ssl/cert.pem" // Укажите путь к вашему сертификату
	keyFile := "ssl/key.pem"   // Укажите путь к вашему ключу*/

	/*creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	if err != nil {
		log.Fatalf("Не удалось загрузить сертификаты: %v", err)
	}*/

	// Создаем gRPC сервер с TLS

	var opts []grpc.ServerOption
	tlsCredentials, err := loadTLSCredentials()
	if err != nil {
		log.Fatalf("cannot load TLS credentials: %s", err)
	}

	opts = append(opts, grpc.Creds(tlsCredentials))

	grpcServer := grpc.NewServer(opts...)

	// Регистрируем наш AuthServiceServer
	auth.RegisterAuthServiceServer(grpcServer, &AuthServiceServer{})

	// Включаем отражение
	reflection.Register(grpcServer)

	log.Printf("gRPC сервер запущен на %s с TLS", ":"+port)

	// Запуск сервера
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Ошибка запуска gRPC сервера: %v", err)
	}
}

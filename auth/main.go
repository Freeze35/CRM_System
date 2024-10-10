package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"testAuth/proto/protobuff/auth"
	"testAuth/utils"
)

// WriteJSON - функция для записи JSON-ответа
func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

type AuthServiceServer struct {
	auth.UnimplementedAuthServiceServer
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
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
}

func getTest(w http.ResponseWriter, r *http.Request) {
	// Здесь вы можете выполнить нужные действия и вернуть ответ
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Тест успешен!"))
}

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

func usersHandler(w http.ResponseWriter, r *http.Request) {
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
}

// Реализация метода Register
func (s *AuthServiceServer) Register(ctx context.Context, req *auth.RegisterRequest) (*auth.RegisterResponse, error) {
	log.Printf("Получен запрос на регистрацию пользователя: %v", req.Username)

	// Здесь должна быть логика для создания базы данных и регистрации пользователя.
	// Например, через другие микросервисы или прямой запрос в базу данных.

	// Пример успешного ответа с сгенерированным токеном
	response := &auth.RegisterResponse{
		Message:  "Регистрация успешна",
		Database: "название_базы_данных",
		Token:    "сгенерированный_токен",
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

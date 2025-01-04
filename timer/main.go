package main

import (
	context "context"
	"crmSystem/proto/dbtimer"
	"crmSystem/proto/timer"
	"crmSystem/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type TimerServiceServer struct {
	timer.UnsafeTimerServiceServer
}

// StartTimer запуска таймера для клиента обращаясь через
func (s *TimerServiceServer) StartTimer(ctx context.Context, req *timer.StartEndTimerRequest) (*timer.StartEndTimerResponse, error) {

	token, err := utils.GetTokenFromMetadata(ctx)

	//Проверка ошибки при получении
	if err != nil {
		log.Printf(err.Error())
	}

	// Устанавливаем соединение с gRPC сервером Nginx
	client, err, conn := utils.GrpcConnector(token)
	defer conn.Close()

	if err != nil {
		response := &timer.StartEndTimerResponse{
			Message: "Не удалось подключиться к серверу: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}
		return response, err
	}

	dbReq := &dbtimer.StartEndTimerRequestDB{
		UserId:      req.UserId,
		DbName:      req.DbName,
		Description: req.Description,
	}

	res, err := client.StartTimerDB(ctx, dbReq)

	if err != nil {
		response := &timer.StartEndTimerResponse{
			Message: "Не удалось подключиться к серверу: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}
		return response, err
	}

	response := &timer.StartEndTimerResponse{
		StartTime: res.StartTime,
		EndTime:   res.EndTime,
		TimerId:   res.TimerId,
		Message:   res.Message,
		Status:    res.Status,
	}

	return response, err
}

func (s *TimerServiceServer) EndTimer(ctx context.Context, req *timer.StartEndTimerRequest) (*timer.StartEndTimerResponse, error) {
	token, err := utils.GetTokenFromMetadata(ctx)

	//Проверка ошибки при получении
	if err != nil {
		log.Printf(err.Error())
	}

	// Устанавливаем соединение с gRPC сервером Nginx
	client, err, conn := utils.GrpcConnector(token)
	defer conn.Close()

	if err != nil {
		response := &timer.StartEndTimerResponse{
			Message: "Не удалось подключиться к серверу: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}
		return response, err
	}

	dbReq := &dbtimer.StartEndTimerRequestDB{
		UserId:      req.UserId,
		DbName:      req.DbName,
		Description: req.Description,
	}

	res, err := client.StartTimerDB(ctx, dbReq)

	if err != nil {
		response := &timer.StartEndTimerResponse{
			Message: "Не удалось подключиться к серверу: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}
		return response, err
	}

	response := &timer.StartEndTimerResponse{
		StartTime: res.StartTime,
		EndTime:   res.EndTime,
		TimerId:   res.TimerId,
		Message:   res.Message,
		Status:    res.Status,
	}

	return response, err
}

func (s *TimerServiceServer) GetWorkingTimer(ctx context.Context, req *timer.WorkingTimerRequest) (*timer.WorkingTimerResponse, error) {
	token, err := utils.GetTokenFromMetadata(ctx)

	//Проверка ошибки при получении
	if err != nil {
		log.Printf(err.Error())
	}

	// Устанавливаем соединение с gRPC сервером Nginx
	client, err, conn := utils.GrpcConnector(token)
	defer conn.Close()

	if err != nil {
		response := &timer.WorkingTimerResponse{
			Message: "Не удалось подключиться к серверу: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}
		return response, err
	}

	dbReq := &dbtimer.WorkingTimerRequestDB{
		UserId: req.UserId,
		DbName: req.DbName,
	}

	res, err := client.GetWorkingTimerDB(ctx, dbReq)

	if err != nil {
		response := &timer.WorkingTimerResponse{
			Message: "Не удалось подключиться к серверу: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}
		return response, err
	}

	response := &timer.WorkingTimerResponse{
		StartTime: res.StartTime,
		EndTime:   res.EndTime,
		TimerId:   res.TimerId,
		Message:   res.Message,
		Status:    res.Status,
	}

	return response, err
}

func (s *TimerServiceServer) ChangeTimer(ctx context.Context, req *timer.ChangeTimerRequest) (*timer.ChangeTimerResponse, error) {

	token, err := utils.GetTokenFromMetadata(ctx)

	//Проверка ошибки при получении
	if err != nil {
		log.Printf(err.Error())
	}

	// Устанавливаем соединение с gRPC сервером Nginx
	client, err, conn := utils.GrpcConnector(token)
	defer conn.Close()

	if err != nil {
		response := &timer.ChangeTimerResponse{
			Message: "Не удалось подключиться к серверу: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}
		return response, err
	}

	dbReq := &dbtimer.ChangeTimerRequestDB{
		UserId:  req.UserId,
		DbName:  req.DbName,
		TimerId: req.TimerId,
	}

	res, err := client.ChangeTimerDB(ctx, dbReq)

	if err != nil {
		response := &timer.ChangeTimerResponse{
			Message: "Не удалось подключиться к серверу: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}
		return response, err
	}

	response := &timer.ChangeTimerResponse{
		StartTime:   res.StartTime,
		EndTime:     res.EndTime,
		Duration:    res.Duration,
		Description: res.Description,
		Active:      res.Active,
		TimerId:     res.TimerId,
		Message:     res.Message,
		Status:      res.Status,
	}

	return response, err

}

func (s *TimerServiceServer) AddTimer(ctx context.Context, req *timer.AddTimerRequest) (*timer.AddTimerResponse, error) {

	token, err := utils.GetTokenFromMetadata(ctx)

	//Проверка ошибки при получении
	if err != nil {
		log.Printf(err.Error())
	}

	// Устанавливаем соединение с gRPC сервером Nginx
	client, err, conn := utils.GrpcConnector(token)
	defer conn.Close()

	if err != nil {
		response := &timer.AddTimerResponse{
			Message: "Не удалось подключиться к серверу: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}
		return response, err
	}

	dbReq := &dbtimer.AddTimerRequestDB{
		UserId:      req.UserId,
		DbName:      req.DbName,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		TimerId:     req.TimerId,
		Description: req.Description,
	}

	res, err := client.AddTimerDB(ctx, dbReq)

	if err != nil {
		response := &timer.AddTimerResponse{
			Message: "Не удалось подключиться к серверу: " + err.Error(),
			Status:  http.StatusInternalServerError,
		}
		return response, err
	}

	response := &timer.AddTimerResponse{
		StartTime:   res.StartTime,
		EndTime:     res.EndTime,
		Duration:    res.Duration,
		Description: res.Description,
		TimerId:     res.TimerId,
		Message:     res.Message,
		Status:      res.Status,
	}

	return response, err
}

func main() {
	// Инициализируем TCP соединение для gRPC сервера

	port := os.Getenv("AUTH_SERVICE_PORT")

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Не удалось запустить сервер: %v", err)
	}

	var opts []grpc.ServerOption
	tlsCredentials, err := utils.LoadTLSCredentials()
	if err != nil {
		log.Fatalf("Невозможно загрузить учетные данные TLS: %s", err)
	}
	opts = []grpc.ServerOption{
		grpc.Creds(tlsCredentials), // Добавление TLS опций
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     5 * time.Minute,
			MaxConnectionAge:      15 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Minute,
			Time:                  5 * time.Second, // Таймаут на соединение
		}),
	}

	/*opts = append(opts, grpc.Creds(tlsCredentials))*/

	grpcServer := grpc.NewServer(opts...)

	// Регистрируем наш AuthServiceServer
	timer.RegisterTimerServiceServer(grpcServer, &TimerServiceServer{})

	// Включаем отражение
	reflection.Register(grpcServer)

	log.Printf("gRPC сервер запущен на %s с TLS", ":"+port)

	// Запуск сервера
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Ошибка запуска gRPC сервера: %v", err)
	}
}

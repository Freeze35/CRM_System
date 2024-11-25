package main

import (
	context "context"
	"crmSystem/proto/timer"
	"crmSystem/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"os"
	"time"
)

type TimerServiceServer struct {
	timer.UnsafeTimerServiceServer
}

func (t TimerServiceServer) SaveTimer(ctx context.Context, req *timer.SaveTimerRequest) (*timer.SaveTimerResponse, error) {

	/*token, err := utils.GetTokenFromMetadata(ctx)

	//Проверка ошибки при получении
	if err != nil {
		log.Printf(err.Error())
	}

	// Устанавливаем соединение с gRPC сервером Nginx
	client, err, conn := utils.GrpcConnector(token)
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
	}*/

	//TODO implement me
	panic("implement me")
}

func (t TimerServiceServer) GetOpenTimer(ctx context.Context, req *timer.GetTimerRequest) (*timer.GetTimerResponse, error) {
	//TODO implement me
	panic("implement me")
}

type UserTimer struct {
	ID          int        `json:"id" gorm:"primaryKey"`
	UserID      int        `json:"user_id" gorm:"not null"`
	StartTime   time.Time  `json:"start_time" gorm:"not null"`
	EndTime     *time.Time `json:"end_time"`
	Description string     `json:"description"`
	IsActive    bool       `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

func startTimer() {

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

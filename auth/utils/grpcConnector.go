package utils

import (
	"context"
	"crmSystem/proto/dbservice"
	"google.golang.org/grpc"
	"log"
	"time"
)

func DbServiceConnector() (client dbservice.DbServiceClient, err error, conn *grpc.ClientConn) {
	// Устанавливаем таймаут для подключения
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second) // Таймаут 5 секунд
	defer cancel()                                                          // Освобождаем ресурсы контекста по завершению

	// Устанавливаем соединение с gRPC сервером dbService с таймаутом
	conn, err = grpc.DialContext(ctx, "dbservice:8081", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		return nil, err, conn
	}

	// Возвращаем клиент и соединение
	return dbservice.NewDbServiceClient(conn), nil, conn
}

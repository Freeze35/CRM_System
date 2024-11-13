package utils

import (
	"context"
	"crmSystem/proto/dbservice"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"log"
	"time"
)

func DbServiceConnector() (client dbservice.DbServiceClient, err error, conn *grpc.ClientConn) {
	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	// Загружаем корневой сертификат CA
	caCert, err := ioutil.ReadFile(clientCACertFile)
	if err != nil {
		fmt.Printf("Не удалось прочитать CA сертификат: %v", err)
		return nil, err, nil
	}

	// Создаем пул корневых сертификатов и добавляем CA сертификат
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Настраиваем TLS с клиентскими сертификатами и проверкой CA
	cert, err := tls.LoadX509KeyPair(serverCertFile, serverKeyFile)
	if err != nil {
		log.Fatalf("Не удалось загрузить клиентские сертификаты: %v", err)
		return nil, err, nil
	}

	creds := credentials.NewTLS(&tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: false, // Отключаем проверку, чтобы использовать CA
	})

	// Устанавливаем соединение с gRPC сервером dbService с TLS
	conn, err = grpc.DialContext(ctx, "nginx:443", grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		return nil, err, conn
	}

	fmt.Println("Успешное подключение к gRPC серверу через NGINX с TLS")
	return dbservice.NewDbServiceClient(conn), nil, conn
}

package utils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"log"
	"os"
	"time"
)

// Добавим функцию генерации токена и установим его в gRPC-запрос
func GRPCServiceConnector[T any](token string, clientFactory func(grpc.ClientConnInterface) T) (client T, err error, conn *grpc.ClientConn) {

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	// Загружаем корневой сертификат CA
	caCert, err := ioutil.ReadFile(ClientCACertFile)
	if err != nil {
		log.Printf("Не удалось прочитать CA сертификат: %v", err)
		return
	}

	// Создаем пул корневых сертификатов и добавляем CA сертификат
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Настраиваем TLS с клиентскими сертификатами и проверкой CA
	cert, err := tls.LoadX509KeyPair(ServerCertFile, ServerKeyFile)
	if err != nil {
		log.Printf("Не удалось загрузить клиентские сертификаты: %v", err)
		return
	}

	creds := credentials.NewTLS(&tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: false,
	})

	// Стандартная опция для привязки SSL
	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}
	// Добавляем токен
	opts = append(opts, grpc.WithPerRPCCredentials(jwtTokenAuth{token}), grpc.WithBlock())

	// Проверяем переменную среды GRPC_PROXY_CONNECTOR
	proxyConnection := os.Getenv("GRPC_PROXY_CONNECTOR")
	if proxyConnection == "" {
		err = fmt.Errorf("переменная среды GRPC_PROXY_CONNECTOR не задана")
		log.Printf("Ошибка: %v", err)
		return
	}

	// Настраиваем gRPC соединение
	conn, err = grpc.DialContext(ctx, proxyConnection, opts...)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		return
	}

	client = clientFactory(conn)
	return
}

// jwtTokenAuth структура для установки JWT токена в качестве аутентификационных данных для gRPC
type jwtTokenAuth struct {
	token string
}

// GetRequestMetadata добавляет JWT-токен в метаданные
func (j jwtTokenAuth) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + j.token,
	}, nil
}

// RequireTransportSecurity возвращает true для принудительного использования TLS
func (j jwtTokenAuth) RequireTransportSecurity() bool {
	return true
}

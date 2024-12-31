package utils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"log"
	"os"
	"time"
)

// GRPCServiceConnector создает gRPC-соединение с сервером через указанного прокси-коннектора,
// используя TLS для защиты соединения и, при необходимости, добавляя JWT-аутентификацию.
// Функция универсальна для создания клиентов различных gRPC сервисов.
//
// Параметры:
// - generateToken (bool): Указывает, нужно ли генерировать JWT токен для аутентификации.
// - clientFactory (func(grpc.ClientConnInterface) T): Фабричная функция для создания клиента gRPC-сервиса.
//
// Возвращает:
// - client (T): Экземпляр клиента gRPC сервиса.
// - err (error): Ошибка, если возникла проблема при настройке соединения.
// - conn (*grpc.ClientConn): Установленное gRPC соединение, которое следует закрыть после использования.
func GRPCServiceConnector[T any](generateToken bool, clientFactory func(grpc.ClientConnInterface) T) (client T, err error, conn *grpc.ClientConn) {
	// Генерация JWT-токена
	token, err := JwtGenerate()
	if err != nil {
		log.Printf("Не удалось сгенерировать JWT: %v", err)
		return
	}

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

	// Стандартная опция для привязки SSL и проверка на генерацию токена
	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}
	if generateToken {
		opts = append(opts, grpc.WithPerRPCCredentials(jwtTokenAuth{token}), grpc.WithBlock())
	}

	proxyConnection := os.Getenv("GRPC_PROXY_CONNECTOR")

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
func (j jwtTokenAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + j.token,
	}, nil
}

// RequireTransportSecurity возвращает true для принудительного использования TLS
func (j jwtTokenAuth) RequireTransportSecurity() bool {
	return true
}

package utils

import (
	"context"
	"crmSystem/proto/redis"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"log"
	"time"
)

// Добавим функцию генерации токена и установим его в gRPC-запрос
func RedisServiceConnector(token string) (client redis.RedisServiceClient, err error, conn *grpc.ClientConn) {

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	// Загружаем корневой сертификат CA
	caCert, err := ioutil.ReadFile(ClientCACertFile)
	if err != nil {
		fmt.Printf("Не удалось прочитать CA сертификат: %v", err)
		return nil, err, nil
	}

	// Создаем пул корневых сертификатов и добавляем CA сертификат
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Настраиваем TLS с клиентскими сертификатами и проверкой CA
	cert, err := tls.LoadX509KeyPair(ServerCertFile, ServerKeyFile)
	if err != nil {
		log.Printf("Не удалось загрузить клиентские сертификаты: %v", err)
		return nil, err, nil
	}

	creds := credentials.NewTLS(&tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: false,
	})

	//Стандартная опция для привязки ssl и проврка на генерецию токена
	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}

	opts = append(opts, grpc.WithPerRPCCredentials(jwtTokenAuth{token}), grpc.WithBlock())

	// Настраиваем gRPC соединение с передачей JWT-токена
	// Подключаемся к основному проксировщику nginx
	conn, err = grpc.DialContext(ctx, "nginx:443", opts...)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		return nil, err, conn
	}

	fmt.Println("Успешное подключение RedisServiceConnector к gRPC серверу через NGINX с TLS")
	return redis.NewRedisServiceClient(conn), nil, conn
}

package utils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
)

const (
	serverCertFile   = "sslkeys/server.pem"
	serverKeyFile    = "sslkeys/server.key"
	clientCACertFile = "sslkeys/ca.crt"
)

// LoadTLSCredentials загружает TLS-учетные данные для сервера.
func LoadTLSCredentials() (credentials.TransportCredentials, error) {
	// Загрузка сертификата CA сервера
	pemServerCA, err := ioutil.ReadFile(clientCACertFile)
	if err != nil {
		return nil, fmt.Errorf("не удалось загрузить сертификат CA сервера: %v", err)
	}

	// Создание пула сертификатов и добавление сертификата CA
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("не удалось добавить сертификат CA сервера в пул")
	}

	// Загрузка сертификата и закрытого ключа сервера
	serverCert, err := tls.LoadX509KeyPair(serverCertFile, serverKeyFile)
	if err != nil {
		return nil, fmt.Errorf("не удалось загрузить сертификат и ключ сервера: %v", err)
	}

	// Настройка TLS-конфигурации
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert, // Двусторонняя аутентификация
		ClientCAs:    certPool,                       // Проверка сертификатов клиентов
	}

	// Возвращаем TLS-настройки для gRPC
	return credentials.NewTLS(config), nil
}

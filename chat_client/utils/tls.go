package utils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
)

const (
	ServerCertFile   = "sslkeys/server.pem"
	ServerKeyFile    = "sslkeys/server.key"
	ClientCACertFile = "sslkeys/ca.crt"
)

// LoadTLSCredentials загружает TLS-учетные данные для сервера.
func LoadTLSCredentials() (*tls.Config, error) {

	// Загрузка сертификата CA сервера
	pemServerCA, err := ioutil.ReadFile(ClientCACertFile)
	if err != nil {
		return nil, fmt.Errorf("не удалось загрузить сертификат CA сервера: %v", err)
	}

	// Создание пула сертификатов и добавление сертификата CA
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("не удалось добавить сертификат CA сервера в пул")
	}

	// Загрузка сертификата и закрытого ключа сервера
	serverCert, err := tls.LoadX509KeyPair(ServerCertFile, ServerKeyFile)

	if err != nil {
		return nil, fmt.Errorf("не удалось загрузить сертификат и ключ сервера: %v", err)
	}

	// Настройка TLS-конфигурации
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert, // Двусторонняя аутентификация
		ClientCAs:    certPool,                       // Проверка сертификатов клиентов
	}

	return config, nil
}

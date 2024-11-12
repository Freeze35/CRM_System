package utils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
)

const (
	serverCertFile   = "sslkeys/server.crt" // Сертификат сервера
	serverKeyFile    = "sslkeys/server.key" // Приватный ключ сервера
	serverCACertFile = "./rootca/ca.crt"    // Корневой сертификат CA сервера
)

func LoadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load certificate of the CA who signed client's certificate
	pemClientCA, err := ioutil.ReadFile(serverCACertFile)
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
		ClientCAs:    certPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12, // Совместимость с серверной конфигурацией
	}

	return credentials.NewTLS(config), nil
}

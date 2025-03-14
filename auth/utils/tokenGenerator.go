package utils

import (
	"crypto/rsa"
	"encoding/pem"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/youmark/pkcs8"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

// JwtGenerate генерирует JWT токен (access или refresh)
func JwtGenerator(username, tokenType string) (string, error) {

	// Путь к зашифрованному закрытому ключу
	keyFile := "./opensslkeys/private_key.pem"

	password := os.Getenv("JWT_SECRET_KEY")

	// Читаем закрытый ключ
	keyData, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения файла: %v", err)
	}

	// Декодируем PEM
	block, _ := pem.Decode(keyData)
	if block == nil {
		return "", fmt.Errorf("ошибка: не удалось распознать PEM-формат")
	}

	// Проверяем тип ключа
	if block.Type != "ENCRYPTED PRIVATE KEY" {
		return "", fmt.Errorf("ошибка: блок не является зашифрованным приватным ключом")
	}

	// Расшифровываем ключ
	privKey, err := pkcs8.ParsePKCS8PrivateKey(block.Bytes, []byte(password))
	if err != nil {
		return "", fmt.Errorf("ошибка расшифровки ключа: %v", err)
	}

	// Преобразуем в *rsa.PrivateKey
	rsaKey, ok := privKey.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("ошибка: не удалось преобразовать в RSA ключ")
	}

	// Определяем срок действия токена
	var expiresAt time.Time
	switch tokenType {
	case "access":
		expiresAt = time.Now().Add(15 * time.Minute) // 15 минут
	case "refresh":
		expiresAt = time.Now().Add(7 * 24 * time.Hour) // 7 дней
	default:
		return "", fmt.Errorf("неизвестный тип токена: %s", tokenType)
	}

	// Создаём токен
	token := jwt.New(jwt.SigningMethodRS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["sub"] = username
	claims["exp"] = expiresAt.Unix()

	// Подписываем токен
	tokenString, err := token.SignedString(rsaKey)
	if err != nil {
		return "", fmt.Errorf("ошибка подписания токена: %v", err)
	}

	return tokenString, nil
}

// ValidateAndRefreshToken проверяет токены в cookie и обновляет access token при необходимости
func ValidateAndRefreshToken(_ http.ResponseWriter, r *http.Request) (string, error) {

	refreshToken, err := GetFromCookies(r, "refresh_token")
	if err != nil {
		return "", err
	}

	accessToken, err := GetFromCookies(r, "access_token")
	if err != nil {
		return "", err
	}

	pubKey, err := loadPublicKey()
	if err != nil {
		return "", fmt.Errorf("ошибка загрузки публичного ключа: %v", err)
	}

	claims, err := validateToken(accessToken, pubKey)
	if err == nil {
		return accessToken, nil // access token действителен
	}

	claims, err = validateToken(refreshToken, pubKey)
	if err != nil {
		return "", fmt.Errorf("оба токена недействительны: %v", err)
	}

	username := claims["sub"].(string)
	newAccessToken, err := JwtGenerator(username, "access")
	if err != nil {
		return "", fmt.Errorf("ошибка генерации нового access token: %v", err)
	}

	// Не устанавливаем куку здесь
	return newAccessToken, nil
}

// validateToken проверяет JWT токен
func validateToken(tokenString string, pubKey *rsa.PublicKey) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("неверный метод подписи")
		}
		return pubKey, nil
	})
	if err != nil {
		return nil, err
	}

	// Проверяем, являются ли claims корректными
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("токен недействителен")
}

// loadPublicKey загружает публичный RSA ключ
func loadPublicKey() (*rsa.PublicKey, error) {
	publicKeyFile := "./opensslkeys/public_key.pem"

	keyData, err := ioutil.ReadFile(publicKeyFile)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения публичного ключа: %v", err)
	}

	if len(keyData) == 0 {
		return nil, fmt.Errorf("файл публичного ключа пуст")
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("ошибка декодирования PEM: файл не в формате PEM")
	}

	if block.Type != "PUBLIC KEY" && block.Type != "RSA PUBLIC KEY" {
		return nil, fmt.Errorf("неверный тип PEM блока: ожидается 'PUBLIC KEY' или 'RSA PUBLIC KEY', получено '%s'", block.Type)
	}

	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(keyData)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга публичного ключа: %v", err)
	}

	return pubKey, nil
}

func InternalJwtGenerator() (string, error) {
	// Путь к зашифрованному закрытому ключу
	keyFile := "./opensslkeys/private_key.pem"

	// Получаем пароль из переменной окружения
	password := os.Getenv("JWT_SECRET_KEY")
	/*password := os.Getenv("PEM_PASSWORD")
	if password == "" {
		return "", fmt.Errorf("PEM_PASSWORD не установлена")
	}*/

	// Чтение файла с ключом
	keyData, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return "", fmt.Errorf("Ошибка чтения файла: %v", err)
	}

	// Расшифровка закрытого ключа
	block, _ := pem.Decode(keyData)
	if block == nil {
		return "", fmt.Errorf("Ошибка: не удалось распознать PEM-формат")
	}

	// Проверка, является ли блок зашифрованным
	if block.Type != "ENCRYPTED PRIVATE KEY" {
		return "", fmt.Errorf("Ошибка: блок не является зашифрованным приватным ключом")
	}

	// Расшифровка ключа с использованием pkcs8
	privKey, err := pkcs8.ParsePKCS8PrivateKey(block.Bytes, []byte(password))
	if err != nil {
		return "", fmt.Errorf("Ошибка расшифровки ключа: %v", err)
	}

	// Генерация JWT
	token := jwt.New(jwt.SigningMethodRS256)

	// Установка данных в токен
	claims := token.Claims.(jwt.MapClaims)
	claims["foo"] = "bar"
	claims["exp"] = time.Now().Add(3 * time.Minute).Unix() // Время истечения токена

	// Подпись токена
	tokenString, err := token.SignedString(privKey)
	if err != nil {
		return "", fmt.Errorf("Ошибка подписывания токена: %v", err)
	}

	return tokenString, nil
}

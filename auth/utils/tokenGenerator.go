package utils

import (
	"encoding/pem"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/youmark/pkcs8"
	"io/ioutil"
	"log"
	"os"
	"time"
)

// Секретный ключ для подписания токена
var jwtSecret = []byte("your_secret_key")

// Функция для генерации JWT токена
func GenerateToken(username string) (string, error) {
	// Определяем время действия токена (1 день)
	expirationTime := time.Now().Add(time.Hour * 24).Unix()

	// Создаем токен с необходимыми полями
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": "example_user",
		"exp":      expirationTime,
	})

	// Подписываем токен с секретным ключом
	tokenString, err := token.SignedString(os.Getenv(""))
	if err != nil {
		log.Fatal("Error generating token:", err)
	}

	return tokenString, nil
}

func JwtGenerate() (string, error) {
	// Путь к зашифрованному закрытому ключу
	keyFile := "./opensslkeys/private_key.pem"

	// Получаем пароль из переменной окружения
	password := "standard_password"
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
	claims["exp"] = time.Now().Add(1440 * time.Minute).Unix() // Время истечения токена // 1 день

	// Подпись токена
	tokenString, err := token.SignedString(privKey)
	if err != nil {
		return "", fmt.Errorf("Ошибка подписывания токена: %v", err)
	}

	fmt.Println("Сгенерированный JWT:", tokenString)
	return tokenString, nil
}

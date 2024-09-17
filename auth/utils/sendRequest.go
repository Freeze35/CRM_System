package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func SendPostRequest(url string, sendData any) (*http.Response, error) {
	// Создаем буфер для записи JSON
	var buf bytes.Buffer

	// Создаем JSON-Encoder и сериализуем структуру в JSON
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ") // Опционально: Установка отступов для читаемости

	if err := encoder.Encode(sendData); err != nil {
		return nil, fmt.Errorf("ошибка сериализации JSON: %w", err)
	}

	// Отправляем POST-запрос
	resp, err := http.Post(url, "application/json", &buf)
	if err != nil {
		return nil, fmt.Errorf("ошибка отправки POST-запроса: %w", err)
	}

	return resp, nil
}

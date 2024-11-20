package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ConvertJSONToStruct универсальная функция для преобразования JSON в структуру
// Может принимать как строку, так и io.Reader
func ConvertJSONToStruct[T any](input interface{}) (*T, error) {
	var reader io.Reader

	// Если передан тип string, создаем io.Reader из строки
	switch v := input.(type) {
	case string:
		reader = strings.NewReader(v)
	case io.Reader:
		reader = v
	default:
		return nil, fmt.Errorf("invalid input type, must be string or io.Reader")
	}

	var result T
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return &result, nil
}

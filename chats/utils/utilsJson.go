package utils

import (
	"crmSystem/transport_rest/types"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status uint32, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(int(status))
	return json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status uint32, err error) {
	WriteJSON(w, status, map[string]string{"error": err.Error()})
}

// ParseJSON декодирует JSON данные из io.Reader в указанный объект v.
// Эта функция использует json.NewDecoder для построчного декодирования данных.
// Она может работать с любым io.Reader, например, с *bytes.Reader или *http.Request.
func ParseJSON(r io.Reader, v any) error {

	// Проверяем, что переданный io.Reader не равен nil
	if r == nil {
		// Если r равен nil, возвращаем ошибку
		return fmt.Errorf("missing request body")
	}

	// Используем json.NewDecoder для декодирования данных из io.Reader в объект v
	return json.NewDecoder(r).Decode(v)
}

func CreateError(w http.ResponseWriter, status uint32, v string, err error) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(int(status))
	response := types.ErrorResponse{
		Message: fmt.Errorf("%s: %v", v, err).Error(),
		Status:  http.StatusBadRequest,
	}

	encodeErr := json.NewEncoder(w).Encode(response)
	if err != nil {
		fmt.Printf("Ошибка при отправке JSON-ответа: %v\n", encodeErr)
	}
}

func ToJSON(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("Ошибка при сериализации в JSON: %v", err)
		return nil // Возвращаем nil в случае ошибки
	}
	return data
}

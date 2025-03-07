package utils

import (
	"crmSystem/transport_rest/types"
	"encoding/json"
	"fmt"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status uint32, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(int(status))
	return json.NewEncoder(w).Encode(v)
}

func CreateError(w http.ResponseWriter, status uint32, v string, err error) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(int(status))
	response := types.ErrorResponse{
		Message: fmt.Errorf("%s %v", v, err).Error(),
	}

	encodeErr := json.NewEncoder(w).Encode(response)
	if encodeErr != nil {
		fmt.Printf("Ошибка при отправке JSON-ответа: %v\n", encodeErr)
	}
}

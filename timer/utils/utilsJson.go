package utils

import (
	"crmSystem/utils/types"
	"encoding/json"
	"fmt"
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

func ParseJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return fmt.Errorf("missing request body")
	}

	return json.NewDecoder(r.Body).Decode(v)
}

func CreateError(w http.ResponseWriter, status uint32, v string, err error) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(int(status))
	response := types.ErrorResponse{
		Message: fmt.Errorf("%s %v", v, err).Error(),
	}

	encodeErr := json.NewEncoder(w).Encode(response)
	if err != nil {
		fmt.Printf("Ошибка при отправке JSON-ответа: %v\n", encodeErr)
	}
}

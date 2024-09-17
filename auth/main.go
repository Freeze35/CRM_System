package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"testAuth/utils"

	"github.com/gorilla/mux"
)

// WriteJSON - функция для записи JSON-ответа
func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	// Структура для JSON-данных
	type RegisterRequest struct {
		Username  string `json:"username"`
		Password  string `json:"password"`
		Address   string `json:"address"`
		CompanyDB string `json:"company_db"`
	}

	// Parse JSON
	var req RegisterRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	// Создание новой базы данных компании
	resp, err := http.Post(fmt.Sprintf("%s/create-db", os.Getenv("DB_SERVER_URL")), "", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, "Ошибка создания базы данных компании", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Читаем тело ответа после создания базы данных
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Ошибка при чтении тела ответа", http.StatusInternalServerError)
		return
	}

	// Выводим тело ответа с названием новой базы данных
	dbName := string(body)

	// Структура для JSON-данных
	type dbStruct struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		DbName  string `json:"dbname"`
	}

	dbJson := dbStruct{
		Name:    req.CompanyDB,
		Address: req.Address,
		DbName:  dbName,
	}

	// Регистрация организации в базе компании
	_, err = utils.SendPostRequest(fmt.Sprintf("%s/register", os.Getenv("DB_SERVER_URL")), dbJson)
	if err != nil {
		http.Error(w, "Ошибка регистрации в базе компании", http.StatusInternalServerError)
		return
	}

	// Возврат успешного JSON-ответа
	response := map[string]string{
		"message":  "Регистрация успешна в обеих базах данных",
		"database": dbName,
	}
	WriteJSON(w, http.StatusOK, response)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	resp, err := http.Post(fmt.Sprintf("%s/login?username=%s&password=%s", os.Getenv("DB_SERVER_URL"), username, password), "", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, "Ошибка входа", http.StatusUnauthorized)
		return
	}
	defer resp.Body.Close()

	w.Write([]byte("Успешный вход"))
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	// Обработка запроса списка пользователей
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/register", registerHandler).Methods("POST")
	r.HandleFunc("/login", loginHandler).Methods("POST")
	r.HandleFunc("/users", usersHandler).Methods("GET")

	log.Println("auth-service запущен на порту 8081")
	err := http.ListenAndServe(":8081", r)
	if err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

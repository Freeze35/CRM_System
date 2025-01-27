package transport_rest

import (
	"context"
	"crmSystem/proto/dbtimer"
	"crmSystem/transport_rest/types"
	"crmSystem/utils"
	"encoding/json"
	"github.com/gorilla/mux"
	"google.golang.org/grpc/metadata"
	"log"
	"net/http"
)

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) InitRouter() *mux.Router {
	r := mux.NewRouter()

	timerRouts := r.PathPrefix("/timer").Subrouter()
	{
		timerRouts.HandleFunc("/start-timer", utils.RecoverMiddleware(h.StartTimer)).Methods(http.MethodPost)
		timerRouts.HandleFunc("/end-timer", utils.RecoverMiddleware(h.EndTimer)).Methods(http.MethodPost)
		timerRouts.HandleFunc("/get-working-timer", utils.RecoverMiddleware(h.GetWorkingTimer)).Methods(http.MethodGet)
		/*books.HandleFunc("/{id:[0-9]+}", h.getBookByID).Methods(http.MethodGet)
		books.HandleFunc("/{id:[0-9]+}", h.deleteBook).Methods(http.MethodDelete)
		books.HandleFunc("/{id:[0-9]+}", h.updateBook).Methods(http.MethodPut)*/
	}

	return r
}

// StartTimer запуска таймера для клиента обращаясь через
func (h *Handler) StartTimer(w http.ResponseWriter, r *http.Request) {

	//Получаем Json
	var req types.StartEndTimerRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Ошибка при декодировании данных", http.StatusBadRequest)
		return
	}

	// Получаем cookie с именами
	token := utils.GetFromCookies(w, r, "auth-token")
	if token == "" {
		return
	}

	userId := utils.GetFromCookies(w, r, "user-id")
	if userId == "" {
		return
	}

	database := utils.GetFromCookies(w, r, "database")
	if database == "" {
		return
	}

	// Устанавливаем соединение с gRPC сервером Nginx
	client, err, conn := utils.GRPCServiceConnector(token, dbtimer.NewDbTimerServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusInternalServerError, "Не удалось подключиться к серверу", err)
		return
	} else {
		defer conn.Close()
	}

	md := metadata.Pairs(
		"user-id", userId,
		"database", database,
	)

	ctxWithMetadata := metadata.NewOutgoingContext(context.Background(), md)

	dbReq := &dbtimer.StartEndTimerRequestDB{
		Description: req.Description,
	}

	res, err := client.StartTimerDB(ctxWithMetadata, dbReq)
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не удалось подключиться к серверу", err)
		return
	}

	response := &types.StartEndTimerResponse{
		StartTime: res.StartTime,
		EndTime:   res.EndTime,
		TimerId:   res.TimerId,
		Message:   res.Message,
	}

	if err := utils.WriteJSON(w, http.StatusOK, response); err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не корректная ошибка на сервере", err)
	}
}

func (h *Handler) EndTimer(w http.ResponseWriter, r *http.Request) {

	//Получаем Json
	var req types.StartEndTimerRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Ошибка при декодировании данных", http.StatusBadRequest)
		return
	}

	// Получаем cookie с именами
	token := utils.GetFromCookies(w, r, "auth-token")
	if token == "" {
		return
	}

	userId := utils.GetFromCookies(w, r, "user-id")
	if userId == "" {
		return
	}

	database := utils.GetFromCookies(w, r, "database")
	if database == "" {
		return
	}

	// Устанавливаем соединение с gRPC сервером Nginx
	client, err, conn := utils.GRPCServiceConnector(token, dbtimer.NewDbTimerServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusInternalServerError, "Не удалось подключиться к серверу", err)
		return
	} else {
		defer conn.Close()
	}

	md := metadata.Pairs(
		"user-id", userId,
		"database", database,
	)

	ctxWithMetadata := metadata.NewOutgoingContext(context.Background(), md)

	dbReq := &dbtimer.StartEndTimerRequestDB{
		Description: req.Description,
	}

	res, err := client.EndTimerDB(ctxWithMetadata, dbReq)

	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не удалось подключиться к серверу", err)
		return
	}

	response := &types.StartEndTimerResponse{
		StartTime: res.StartTime,
		EndTime:   res.EndTime,
		TimerId:   res.TimerId,
		Message:   res.Message,
	}

	if err := utils.WriteJSON(w, http.StatusOK, response); err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не корректная ошибка на сервере", err)
	}
}

func (h *Handler) GetWorkingTimer(w http.ResponseWriter, r *http.Request) {

	// Получаем cookie с именами
	token := utils.GetFromCookies(w, r, "auth-token")
	if token == "" {
		return
	}

	userId := utils.GetFromCookies(w, r, "user-id")
	if userId == "" {
		return
	}

	database := utils.GetFromCookies(w, r, "database")
	if database == "" {
		return
	}

	// Устанавливаем соединение с gRPC сервером Nginx
	client, err, conn := utils.GRPCServiceConnector(token, dbtimer.NewDbTimerServiceClient)
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не удалось подключиться к серверу", err)
		return
	} else {
		defer conn.Close()
	}

	dbReq := &dbtimer.WorkingTimerRequestDB{}

	md := metadata.Pairs(
		"user-id", userId,
		"database", database,
	)

	ctxWithMetadata := metadata.NewOutgoingContext(context.Background(), md)

	res, err := client.GetWorkingTimerDB(ctxWithMetadata, dbReq)

	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не удалось подключиться к серверу", err)
	}

	response := &types.WorkingTimerResponse{
		StartTime: res.StartTime,
		EndTime:   res.EndTime,
		TimerId:   res.TimerId,
		Message:   res.Message,
	}

	if err := utils.WriteJSON(w, http.StatusOK, response); err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не корректная ошибка на сервере", err)
	}
}

func (h *Handler) ChangeTimer(w http.ResponseWriter, r *http.Request) {

	//Получаем Json
	var req types.ChangeTimerRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Ошибка при декодировании данных", http.StatusBadRequest)
		return
	}

	// Получаем cookie с именами
	token := utils.GetFromCookies(w, r, "auth-token")
	if token == "" {
		return
	}

	userId := utils.GetFromCookies(w, r, "user-id")
	if userId == "" {
		return
	}

	database := utils.GetFromCookies(w, r, "database")
	if database == "" {
		return
	}

	// Устанавливаем соединение с gRPC сервером Nginx
	client, err, conn := utils.GRPCServiceConnector(token, dbtimer.NewDbTimerServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusInternalServerError, "Не корректная ошибка на сервере", err)
		return
	} else {
		defer conn.Close()
	}

	dbReq := &dbtimer.ChangeTimerRequestDB{
		TimerId: req.TimerId,
	}

	md := metadata.Pairs(
		"user-id", userId,
		"database", database,
	)

	ctxWithMetadata := metadata.NewOutgoingContext(context.Background(), md)

	res, err := client.ChangeTimerDB(ctxWithMetadata, dbReq)

	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не корректная ошибка на сервере", err)
		return
	}

	response := &dbtimer.ChangeTimerResponseDB{
		StartTime:   res.StartTime,
		EndTime:     res.EndTime,
		Duration:    res.Duration,
		Description: res.Description,
		Active:      res.Active,
		TimerId:     res.TimerId,
		Message:     res.Message,
	}

	if err := utils.WriteJSON(w, http.StatusOK, response); err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не корректная ошибка на сервере", err)
	}

}

func (h *Handler) AddTimer(w http.ResponseWriter, r *http.Request) {

	//Получаем Json
	var req types.AddTimerRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Ошибка при декодировании данных", http.StatusBadRequest)
		return
	}

	// Получаем cookie с именами
	token := utils.GetFromCookies(w, r, "auth-token")
	if token == "" {
		return
	}

	userId := utils.GetFromCookies(w, r, "user-id")
	if userId == "" {
		return
	}

	database := utils.GetFromCookies(w, r, "database")
	if database == "" {
		return
	}

	// Устанавливаем соединение с gRPC сервером Nginx
	client, err, conn := utils.GRPCServiceConnector(token, dbtimer.NewDbTimerServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusInternalServerError, "Не корректная ошибка на сервере", err)
		return
	} else {
		defer conn.Close()
	}

	dbReq := &dbtimer.AddTimerRequestDB{
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		TimerId:     req.TimerId,
		Description: req.Description,
	}

	md := metadata.Pairs(
		"user-id", userId,
		"database", database,
	)

	ctxWithMetadata := metadata.NewOutgoingContext(context.Background(), md)

	res, err := client.AddTimerDB(ctxWithMetadata, dbReq)

	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не удалось подключиться к серверу", err)
		return
	}

	response := &dbtimer.AddTimerResponseDB{
		StartTime:   res.StartTime,
		EndTime:     res.EndTime,
		Duration:    res.Duration,
		Description: res.Description,
		TimerId:     res.TimerId,
		Message:     res.Message,
	}

	if err := utils.WriteJSON(w, http.StatusOK, response); err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Не корректная ошибка на сервере", err)
	}
}

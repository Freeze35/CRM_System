package transport_rest

import (
	"context"
	"crmSystem/transport_rest/types"
	"crmSystem/utils"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"crmSystem/proto/dbchat" // Импорт gRPC-протокола
	"github.com/gorilla/mux"
	"github.com/streadway/amqp"
)

type Handler struct {
	rabbitMQConn *amqp.Connection
	clients      map[string]*amqp.Queue
	grpcClient   dbchat.DbChatServiceClient // gRPC клиент
}

func NewHandler(rabbitMQConn *amqp.Connection) *Handler {
	return &Handler{
		rabbitMQConn: rabbitMQConn,
		clients:      make(map[string]*amqp.Queue),
	}
}

func (h *Handler) InitRouter() *mux.Router {
	r := mux.NewRouter()
	chatsRouts := r.PathPrefix("/chats").Subrouter()
	{
		chatsRouts.HandleFunc("/createNewChat", h.CreateNewChat).Methods(http.MethodPost)
	}
	return r
}

func convertToProtoUsers(users []types.UserID) []*dbchat.UserId {
	protoUsers := make([]*dbchat.UserId, len(users))
	for i, user := range users {
		protoUsers[i] = &dbchat.UserId{
			UserId: user.UserId,
			RoleId: user.RoleId,
		}
	}
	return protoUsers
}

func (h *Handler) CreateNewChat(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Подключение к gRPC серверу dbService
	client, err, conn := utils.GRPCServiceConnector(true, dbchat.NewDbChatServiceClient)
	if err != nil {
		log.Printf("Не удалось подключиться к серверу: %v", err)
		utils.CreateError(w, http.StatusBadRequest, "Ошибка подключения к dbchatclient", err)
		return
	} else {
		defer conn.Close()
	}

	var req types.CreateChatRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		utils.CreateError(w, http.StatusBadRequest, "Ошибка при декодировании данных.", err)
		return
	}

	dbReq := &dbchat.CreateChatRequest{
		UsersId:  convertToProtoUsers(req.UsersId),
		DbName:   req.DbName,
		ChatName: req.ChatName,
	}

	res, err := client.CreateChat(ctx, dbReq)
	if err != nil {
		utils.CreateError(w, http.StatusBadRequest, "Ошибка при создании чата.", err)
		return
	}

	// Публикация сообщения в RabbitMQ
	if err := h.publishToRabbitMQ(res.ChatId, req.UsersId); err != nil {
		log.Printf("Ошибка публикации в RabbitMQ: %v", err)
		utils.CreateError(w, http.StatusInternalServerError, "Ошибка при публикации чата.", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Chat created and published to RabbitMQ successfully"))
}

func (h *Handler) publishToRabbitMQ(chatID int64, users []types.UserID) error {
	channel, err := h.rabbitMQConn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	// Генерируем уникальное имя очереди, основанное на chatID
	queueName := fmt.Sprintf("chat_queue_%d", chatID)

	// Объявление очереди (если она еще не существует)
	queue, err := channel.QueueDeclare(
		queueName, // имя очереди
		true,      // durable
		false,     // autoDelete
		false,     // exclusive
		false,     // noWait
		nil,       // arguments
	)
	if err != nil {
		return err
	}

	message := struct {
		ChatID int64          `json:"chat_id"`
		Users  []types.UserID `json:"users"`
	}{
		ChatID: chatID,
		Users:  users,
	}

	messageBody, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Публикация сообщения
	err = channel.Publish(
		"",         // exchange
		queue.Name, // routing key (имя очереди)
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        messageBody,
		},
	)
	return err
}

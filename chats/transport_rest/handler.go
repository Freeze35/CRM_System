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
		chatsRouts.HandleFunc("/{chatID}/sendMessage", h.SendMessage).Methods(http.MethodPost)
		chatsRouts.HandleFunc("/{chatID}/messages", h.GetMessages).Methods(http.MethodGet)
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

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {

	log.Printf("LogedSend")
	vars := mux.Vars(r)
	chatID := vars["chatID"]

	var message types.ChatMessage
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		utils.CreateError(w, http.StatusBadRequest, "Ошибка декодирования сообщения", err)
		return
	}

	log.Printf("LogedSend1")

	channel, err := h.rabbitMQConn.Channel()
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Ошибка подключения к каналу RabbitMQ", err)
		return
	} else {
		defer channel.Close()
	}
	log.Printf("LogedSend2")
	// Объявление обменника
	exchangeName := fmt.Sprintf("chat_exchange_%s", chatID)
	if err := channel.ExchangeDeclare(
		exchangeName, "fanout", true, false, false, false, nil,
	); err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Ошибка создания обменника", err)
		return
	}
	log.Printf("LogedSend31")
	message.Time = time.Now()
	body, err := json.Marshal(message)
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Ошибка сериализации сообщения", err)
		return
	}
	log.Printf("LogedSend4")
	// Публикация сообщения
	if err := channel.Publish(exchangeName, "", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	}); err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Ошибка публикации сообщения", err)
		return
	}
	log.Printf("LogedSend5")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Сообщение отправлено"))

}

func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	log.Printf("GetMessages started")
	vars := mux.Vars(r)
	chatID := vars["chatID"]

	// Подключаемся к каналу RabbitMQ
	channel, err := h.rabbitMQConn.Channel()
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Ошибка подключения к каналу RabbitMQ", err)
		return
	}
	defer channel.Close()

	// Имя обменника
	exchangeName := fmt.Sprintf("chat_exchange_%s", chatID)

	// Создаем временную уникальную очередь для каждого клиента
	queue, err := channel.QueueDeclare(
		"",    // Имя очереди (пустое, чтобы RabbitMQ сгенерировал уникальное имя)
		false, // durable
		true,  // autoDelete (очередь удаляется, если клиент отключается)
		true,  // exclusive (только для этого подключения)
		false, // noWait
		nil,   // arguments
	)
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Ошибка создания очереди", err)
		return
	}

	// Привязываем очередь к обменнику
	if err := channel.QueueBind(queue.Name, "", exchangeName, false, nil); err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Ошибка привязки очереди к обменнику", err)
		return
	}

	// Подписываемся на очередь
	messages, err := channel.Consume(
		queue.Name, // Имя очереди
		"",         // consumer
		true,       // autoAck
		false,      // exclusive
		false,      // noLocal
		false,      // noWait
		nil,        // arguments
	)
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Ошибка получения сообщений", err)
		return
	}

	var response []types.ChatMessage
	timeout := time.After(5 * time.Second) // Тайм-аут ожидания сообщений

loop:
	for {
		select {
		case msg := <-messages:
			var message types.ChatMessage
			if err := json.Unmarshal(msg.Body, &message); err != nil {
				log.Printf("Ошибка декодирования сообщения: %v", err)
				continue
			}
			response = append(response, message)
		case <-timeout:
			log.Println("Тайм-аут ожидания сообщений")
			break loop
		}
	}

	// Устанавливаем заголовки ответа
	w.Header().Set("Content-Type", "application/json")
	if len(response) == 0 {
		// Возврат пустого массива, если сообщений нет
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("[]")); err != nil {
			log.Printf("Ошибка записи пустого ответа: %v", err)
		}
		return
	}

	// Кодируем и отправляем ответ
	if err := json.NewEncoder(w).Encode(response); err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Ошибка отправки ответа", err)
		return
	}

	log.Printf("GetMessages completed, messages sent: %d", len(response))
}

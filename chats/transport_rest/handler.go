package transport_rest

import (
	"context"
	"crmSystem/transport_rest/types"
	"crmSystem/utils"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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
		chatsRouts.HandleFunc("/createNewChat", utils.RecoverMiddleware(h.CreateNewChat)).Methods(http.MethodPost)
		chatsRouts.HandleFunc("/{chatID}/sendMessage", utils.RecoverMiddleware(h.SendMessage)).Methods(http.MethodPost)
		chatsRouts.HandleFunc("/{chatID}/messages", utils.RecoverMiddleware(h.GetMessages)).Methods(http.MethodGet)
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

	md := metadata.Pairs(
		"user-id", userId,
		"database", database,
	)

	ctxWithMetadata := metadata.NewOutgoingContext(context.Background(), md)

	ctxWithMetadata, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Подключение к gRPC серверу dbService
	client, err, conn := utils.GRPCServiceConnector(token, dbchat.NewDbChatServiceClient)
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
		ChatName: req.ChatName,
	}

	res, err := client.CreateChat(ctxWithMetadata, dbReq)
	if err != nil {
		// Получаем сообщение об ошибке
		errorMessage := status.Convert(err).Message()

		// Код ошибки
		code := status.Code(err)

		// Логика в зависимости от кода ошибки
		switch code {
		case codes.Unauthenticated:
			utils.CreateError(w, http.StatusBadRequest, fmt.Sprintf("неизвестная ошибка : %s", errorMessage), err)

		default:
			utils.CreateError(w, http.StatusBadRequest, fmt.Sprintf("ошибка при создании чата,неизвестная ошибка: %s", errorMessage), err)
		}
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
	_, err = w.Write([]byte("Чат создан и опубликован в RabbitMQ успешно"))
	if err != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Ошибка при записи ответа.", err)
		return
	}
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

	//Данные из параметров маршрута /chats/{chatID}
	vars := mux.Vars(r)
	chatID := vars["chatID"]

	var message types.ChatMessage
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		utils.CreateError(w, http.StatusBadRequest, "Ошибка декодирования сообщения", err)
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

	md := metadata.Pairs(
		"user-id", userId,
		"database", database,
	)

	ctxWithMetadata := metadata.NewOutgoingContext(context.Background(), md)

	ctxWithMetadata, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Устанавливаем текущее время для сообщения
	message.Time = time.Now()

	// Канал для сбора ошибок из горутин
	errChan := make(chan error, 2)
	defer close(errChan)

	// Асинхронная отправка в RabbitMQ
	go func() {
		channel, err := h.rabbitMQConn.Channel()
		if err != nil {
			errChan <- fmt.Errorf("Ошибка подключения к каналу RabbitMQ: %v", err)
			return
		}
		defer channel.Close()

		// Объявление обменника
		exchangeName := fmt.Sprintf("chat_exchange_%s", chatID)
		if err := channel.ExchangeDeclare(
			exchangeName, "fanout", true, false, false, false, nil,
		); err != nil {
			errChan <- fmt.Errorf("Ошибка создания обменника: %v", err)
			return
		}

		// Сериализация сообщения
		body, err := json.Marshal(message)
		if err != nil {
			errChan <- fmt.Errorf("Ошибка сериализации сообщения: %v", err)
			return
		}

		// Публикация сообщения
		if err := channel.Publish(exchangeName, "", false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		}); err != nil {
			errChan <- fmt.Errorf("Ошибка публикации сообщения: %v", err)
			return
		}

		errChan <- nil
	}()

	// Асинхронное сохранение в базу данных через gRPC
	go func() {
		client, err, conn := utils.GRPCServiceConnector(token, dbchat.NewDbChatServiceClient)
		if err != nil {
			errChan <- fmt.Errorf("Ошибка подключения к gRPC серверу: %v", err)
			return
		} else {
			defer conn.Close()
		}

		// Формирование запроса
		req := &dbchat.SaveMessageRequest{
			ChatId:  message.ChatID,
			Content: message.Content,
			Time:    timestamppb.New(message.Time),
		}

		_, err = client.SaveMessage(ctxWithMetadata, req)
		if err != nil {
			log.Printf("Ошибка сохранения сообщения в базе данных: %v", err)
			errChan <- err
			return
		}

		errChan <- nil
	}()

	// Обработка результатов выполнения горутин
	var rabbitErr, dbErr error
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			if rabbitErr == nil {
				rabbitErr = err
			} else {
				dbErr = err
			}
		}
	}

	// Ответ клиенту в зависимости от результатов
	if rabbitErr == nil && dbErr == nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Сообщение отправлено и сохранено"))
	} else if rabbitErr != nil && dbErr != nil {
		utils.CreateError(w, http.StatusInternalServerError, "Ошибка отправки и сохранения сообщения", fmt.Errorf("%v; %v", rabbitErr, dbErr))
	} else if rabbitErr != nil {
		// Получаем сообщение об ошибке
		errorMessage := status.Convert(rabbitErr).Message()

		// Код ошибки
		code := status.Code(rabbitErr)

		// Логика в зависимости от кода ошибки
		switch code {
		case codes.Unauthenticated:
			utils.CreateError(w, http.StatusBadRequest, fmt.Sprintf("неизвестная ошибка : %s", errorMessage), rabbitErr)
		default:
			utils.CreateError(w, http.StatusInternalServerError, "Ошибка отправки сообщения в RabbitMQ.", rabbitErr)
		}

	} else {
		// Получаем сообщение об ошибке
		errorMessage := status.Convert(rabbitErr).Message()

		// Код ошибки
		code := status.Code(rabbitErr)

		// Логика в зависимости от кода ошибки
		switch code {
		case codes.Unauthenticated:
			utils.CreateError(w, http.StatusBadRequest, fmt.Sprintf("неизвестная ошибка : %s", errorMessage), rabbitErr)
		default:
			utils.CreateError(w, http.StatusInternalServerError, "Ошибка сохранения сообщения в базе данных", dbErr)
		}

	}
}

func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {

	//TODO Add Get FROM DB
	//Данные из параметров маршрута /chats/{chatID}
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
			/*log.Println("Тайм-аут ожидания сообщений")*/
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

}

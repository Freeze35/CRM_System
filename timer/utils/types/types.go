package types

import "time"

type UserID struct {
	UserId int64 `json:"user_id"`
	RoleId int64 `json:"role_id"`
}

type CreateChatRequest struct {
	UsersId  []UserID `json:"users_id"`
	DbName   string   `json:"dbName"`
	ChatName string   `json:"chatName"`
}

type CreateChatResponse struct {
	ChatId    int64  `json:"chatId"`
	DbName    string `json:"dbName"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

type ChatMessage struct {
	DBName  string    `json:"db_name"`
	ChatID  int64     `json:"chat_id"`
	UserID  int64     `json:"user_id"`
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
}

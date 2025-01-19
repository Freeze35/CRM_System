package types

type SendEmailRequest struct {
	Email   string `json:"email" validate:"required,email"`
	Message string `json:"message"`
	Body    string `json:"body"`
}

type SendEmailResponse struct {
	Message  string `json:"message"`
	Status   uint32 `json:"status"`
	Failures string `json:"failures"`
}

type ErrorResponse struct {
	Message string `json:"message"`
	Status  uint32 `json:"status"`
}

type User struct {
	Email  string `json:"email" validate:"required,email"` // Обязательно, должен быть корректным email
	Phone  string `json:"phone" validate:"required"`       // Обязательно
	RoleId int64  `json:"roleId" validate:"required"`      // Обязательно
}

type RegisterUsersRequest struct {
	DbName    string  `json:"dbName" validate:"required"`
	CompanyId string  `json:"companyId" validate:"required"`  // Исправлено на CompanyId
	Users     []*User `json:"users" validate:"required,dive"` // Исправлено закрытие кавычек
}

type UserResponse struct {
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone" validate:"phone"`
	RoleId   string `json:"roleId"`
	Password string `json:"password"`
}

type RegisterUsersResponse struct {
	Message   string         `json:"message"`
	Users     []UserResponse `json:"userResponse"` // Исправлено users вместо userResponse
	CompanyId string         `json:"companyId"`    // Исправлено на CompanyId
	Status    uint32         `json:"status"`       // Исправлено на status
}

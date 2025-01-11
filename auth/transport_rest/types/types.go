package types

type RegisterAuthRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Phone       string `json:"phone" validate:"omitempty,phone"`
	Password    string `json:"password"`
	NameCompany string `json:"nameCompany"`
	Address     string `json:"address"`
	CompanyDb   string `json:"company_db"`
}

type RegisterAuthResponse struct {
	Message   string `json:"message"`
	Database  string `json:"database"`
	CompanyId string `json:"companyId"`
	Token     string `json:"token"`
	Status    uint32 `json:"status"`
}

type LoginAuthRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone" validate:"omitempty,phone"`
	Password string `json:"password" validate:"required,password"`
}

type LoginAuthResponse struct {
	Message   string `json:"message"`
	Database  string `json:"database"`
	CompanyId string `json:"companyId"`
	Token     string `json:"token"`
	Status    uint32 `json:"status"`
}

type SendEmailRequest struct {
	Email   string `json:"recipient"`
	Message string `json:"message"`
	Body    string `json:"body"`
}

type SendEmailResponse struct {
	Message string `json:"message"`
	Status  uint32 `json:"status"`
}

type ErrorResponse struct {
	Message string `json:"message"`
	Status  uint32 `json:"status"`
}

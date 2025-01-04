package types

type SendEmailRequest struct {
	Email   string `json:"email" validate:"required,email"`
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

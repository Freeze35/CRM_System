package types

type SendEmailRequest struct {
	Recipient string `json:"recipient" validate:"required,email"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
}

type SendEmailResponse struct {
	Message string `json:"message"`
	Status  uint32 `json:"status"`
}

type ErrorResponse struct {
	Message string `json:"message"`
	Status  uint32 `json:"status"`
}

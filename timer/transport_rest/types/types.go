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
	Message string `json:"message"`
}

type StartEndTimerRequest struct {
	Description string `json:"description"`
}

type StartEndTimerResponse struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	TimerId   uint64 `json:"timerId"`
	Message   string `json:"message"`
}

type WorkingTimerRequest struct {
	Description string `json:"description"`
}

type WorkingTimerResponse struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	TimerId   uint64 `json:"timerId"`
	Message   string `json:"message"`
}

type ChangeTimerRequest struct {
	TimerId uint64 `json:"timerId"`
}

type AddTimerRequest struct {
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
	TimerId     uint64 `json:"timerId"`
	Description string `json:"description"`
}

type AddTimerResponse struct {
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Duration    uint64 `json:"duration"`
	Description string `json:"description"`
	TimerId     uint64 `json:"timer_id"`
	Message     string `json:"message"`
}

type SendEmailRequest struct {
	Email   string `json:"recipient"`
	Message string `json:"message"`
	Body    string `json:"body"`
}

type SendEmailResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

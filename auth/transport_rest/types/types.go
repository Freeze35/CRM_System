package types

type RegisterAuthRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Phone       string `json:"phone" validate:"omitempty,phone"`
	Password    string `json:"password" validate:"required,password"`
	NameCompany string `json:"nameCompany" validate:"required"`
	Address     string `json:"address" validate:"required"`
	CompanyDb   string `json:"company_db" validate:"required"`
}

type RegisterAuthResponse struct {
	Message       string `json:"message"`
	Database      string `json:"database"`
	UserCompanyId string `json:"userCompanyId"`
	Token         string `json:"token"`
	Status        uint32 `json:"status"`
}

type LoginAuthRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone" validate:"omitempty,phone"`
	Password string `json:"password" validate:"required,password"`
}

type LoginAuthResponse struct {
	Message       string `json:"message"`
	Database      string `json:"database"`
	UserCompanyId string `json:"userCompanyId"`
	Token         string `json:"token"`
	Status        uint32 `json:"status"`
}

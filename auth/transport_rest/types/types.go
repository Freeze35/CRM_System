package types

type RegisterAuthRequest struct {
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Password    string `json:"password"`
	NameCompany string `json:"nameCompany"`
	Address     string `json:"address"`
	CompanyDb   string `json:"company_db"`
}

type RegisterAuthResponse struct {
	Message       string `json:"message"`
	Database      string `json:"database"`
	UserCompanyId string `json:"userCompanyId"`
	Token         string `json:"token"`
	Status        uint32 `json:"status"`
}

type LoginAuthRequest struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type LoginAuthResponse struct {
	Message       string `json:"message"`
	Database      string `json:"database"`
	UserCompanyId string `json:"userCompanyId"`
	Token         string `json:"token"`
	Status        uint32 `json:"status"`
}

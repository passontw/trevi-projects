package interfaces

type LoginRequest struct {
	Phone    string `json:"phone" binding:"required" example:"0987654321"`
	Password string `json:"password" binding:"required" example:"a12345678"`
}

type LoginResponse struct {
	Token string `json:"token"  example:"0987654321"`
	User  User   `json:"user"`
}

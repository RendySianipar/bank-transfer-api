package model

type User struct {
	ID           string `json:"id"`
	UserName     string `json:"username"`
	PasswordHash string `json:"-"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

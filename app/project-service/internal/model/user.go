package model

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Role     string `json:"role"`
	Password string `json:"-"` // hashed password
}

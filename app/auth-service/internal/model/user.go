package model

type User struct {
	ID       int32  `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Password string `json:"-"` // hashed password
}

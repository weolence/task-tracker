package model

type User struct {
	ID       int
	Email    string
	Name     string
	Surname  string
	Password string // hashed password
}

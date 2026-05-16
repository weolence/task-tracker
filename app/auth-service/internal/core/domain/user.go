package domain

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type User struct {
	ID       int32
	Email    string
	Name     string
	Surname  string
	Role     string
	Password string
}

func IsValidRole(role string) bool {
	return role == RoleUser || role == RoleAdmin
}

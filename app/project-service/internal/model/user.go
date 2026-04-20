package model

// MemberInfo contains user information for project members
type MemberInfo struct {
	ID      int    `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
}

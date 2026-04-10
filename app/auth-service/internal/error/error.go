package error

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrConnectionToDB     = errors.New("unable to get data from database")
	ErrJwtTokenCreation   = errors.New("unable to create session")
)

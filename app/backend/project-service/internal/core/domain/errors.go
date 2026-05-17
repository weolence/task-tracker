package domain

import "errors"

var (
	ErrProjectNotFound   = errors.New("project not found")
	ErrTaskNotFound      = errors.New("task not found")
	ErrMemberNotFound    = errors.New("member not found")
	ErrUserAlreadyMember = errors.New("user is already a member")
)

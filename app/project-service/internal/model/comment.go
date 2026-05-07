package model

import "time"

type Comment struct {
	ID           int32
	AuthorID     int32
	TaskID       int32
	Content      string
	CreationDate time.Time
}

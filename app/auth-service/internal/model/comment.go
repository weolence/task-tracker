package model

import "time"

type Comment struct {
	ID           int
	AuthorID     int
	TaskID       int
	Content      string
	CreationDate time.Time
}

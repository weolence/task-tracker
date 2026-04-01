package model

import "time"

type ProjectStatus int

const (
	ProjectStatusInWork = iota
	ProjectStatusEnded
)

type Project struct {
	ID          int
	ManagerID   int
	Name        string
	Description string
	Status      ProjectStatus
	StartDate   time.Time
	EndDate     *time.Time
}

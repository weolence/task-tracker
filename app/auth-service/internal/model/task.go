package model

import "time"

type TaskDifficulty int
type TaskPriority int
type TaskStatus int

const (
	TaskDifficultyEasy TaskDifficulty = iota
	TaskDifficultyMedium
	TaskDifficultyHard
)

const (
	TaskPriorityLow TaskPriority = iota
	TaskPriorityMedium
	TaskPriorityHigh
)

const (
	TaskStatusNotStarted TaskStatus = iota
	TaskStatusInWork
	TaskStatusCompleted
)

type Task struct {
	ID         int
	ProjectID  int
	AssigneeID int
	Name       string
	Priority   TaskPriority
	Difficulty TaskDifficulty
	Status     TaskStatus
	StartDate  time.Time
	EndDate    *time.Time
}

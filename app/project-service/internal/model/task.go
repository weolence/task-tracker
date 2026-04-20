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
	ID          int            `json:"id"`
	ProjectID   int            `json:"project_id"`
	AssigneeID  *int           `json:"assignee_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Priority    TaskPriority   `json:"priority"`
	Difficulty  TaskDifficulty `json:"difficulty"`
	Status      TaskStatus     `json:"status"`
	StartDate   time.Time      `json:"start_date"`
	EndDate     *time.Time     `json:"end_date"`
}

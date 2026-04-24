package model

import "time"

type TaskDifficulty int
type TaskPriority int
type TaskStatus int

const (
	TaskDifficultyEasy TaskDifficulty = iota + 1
	TaskDifficultyMedium
	TaskDifficultyHard
)

const (
	TaskPriorityLow TaskPriority = iota + 1
	TaskPriorityMedium
	TaskPriorityHigh
)

const (
	TaskStatusNotStarted TaskStatus = iota + 1
	TaskStatusInWork
	TaskStatusCompleted
)

type Task struct {
	ID          int32          `json:"id"`
	ProjectID   int32          `json:"project_id"`
	AssigneeID  *int32         `json:"assignee_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Priority    TaskPriority   `json:"priority"`
	Difficulty  TaskDifficulty `json:"difficulty"`
	Status      TaskStatus     `json:"status"`
	StartDate   *time.Time     `json:"start_date,omitempty"`
	EndDate     *time.Time     `json:"end_date"`
}

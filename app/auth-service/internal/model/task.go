package model

import "time"

type TaskDifficulty int
type TaskPriority int
type TaskState int

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
	TaskStateNotStarted TaskState = iota
	TaskStateInWork
	TaskStateCompleted
)

type Task struct {
	ID         int
	Name       string
	Priority   TaskPriority
	Difficulty TaskDifficulty
	StartDate  time.Time
	EndDate    time.Time
}

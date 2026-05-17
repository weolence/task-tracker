package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"project-service/internal/core/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TaskRepository struct {
	pool *pgxpool.Pool
}

func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

func (r *TaskRepository) categoryID(ctx context.Context, categoryType string, name string) (int, error) {
	var id int
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM task_categories
		WHERE category_type = $1 AND name = $2
		LIMIT 1
	`, categoryType, name).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return id, nil
}

func (r *TaskRepository) statusID(ctx context.Context, status domain.TaskStatus) (int, error) {
	var id int
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM task_statuses
		WHERE name = $1
		LIMIT 1
	`, statusNameForTaskStatus(status)).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return id, nil
}

func statusNameForTaskStatus(s domain.TaskStatus) string {
	switch s {
	case domain.TaskStatusNotStarted:
		return "Not Started"
	case domain.TaskStatusInWork:
		return "In Work"
	case domain.TaskStatusOnReview:
		return "On Review"
	case domain.TaskStatusClosed:
		return "Closed"
	default:
		return "Not Started"
	}
}

func taskStatusFromName(name string) domain.TaskStatus {
	switch strings.ToLower(name) {
	case "not started":
		return domain.TaskStatusNotStarted
	case "in work":
		return domain.TaskStatusInWork
	case "on review":
		return domain.TaskStatusOnReview
	case "closed":
		return domain.TaskStatusClosed
	default:
		return domain.TaskStatusNotStarted
	}
}

func categoryNameForDifficulty(d domain.TaskDifficulty) string {
	switch d {
	case domain.TaskDifficultyEasy:
		return "Easy"
	case domain.TaskDifficultyMedium:
		return "Medium"
	case domain.TaskDifficultyHard:
		return "Hard"
	default:
		return "Medium"
	}
}

func categoryNameForPriority(p domain.TaskPriority) string {
	switch p {
	case domain.TaskPriorityLow:
		return "Low"
	case domain.TaskPriorityMedium:
		return "Medium"
	case domain.TaskPriorityHigh:
		return "High"
	default:
		return "Medium"
	}
}

func difficultyFromCategory(name string) domain.TaskDifficulty {
	switch strings.ToLower(name) {
	case "easy":
		return domain.TaskDifficultyEasy
	case "medium":
		return domain.TaskDifficultyMedium
	case "hard":
		return domain.TaskDifficultyHard
	default:
		return domain.TaskDifficultyMedium
	}
}

func priorityFromCategory(name string) domain.TaskPriority {
	switch strings.ToLower(name) {
	case "low":
		return domain.TaskPriorityLow
	case "medium":
		return domain.TaskPriorityMedium
	case "high":
		return domain.TaskPriorityHigh
	default:
		return domain.TaskPriorityMedium
	}
}

func ptrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (r *TaskRepository) CreateTask(ctx context.Context, task domain.Task) (int32, error) {
	difficultyID, err := r.categoryID(ctx, "difficulty", categoryNameForDifficulty(task.Difficulty))
	if err != nil {
		return 0, err
	}
	priorityID, err := r.categoryID(ctx, "priority", categoryNameForPriority(task.Priority))
	if err != nil {
		return 0, err
	}
	statusID, err := r.statusID(ctx, task.Status)
	if err != nil {
		return 0, err
	}

	query := `
		INSERT INTO tasks (project_id, assignee_id, name, description, difficulty_category_id, priority_category_id, status_id, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	var id int32
	err = r.pool.QueryRow(ctx, query,
		task.ProjectID,
		task.AssigneeID,
		task.Name,
		task.Description,
		difficultyID,
		priorityID,
		statusID,
		task.StartDate,
		task.EndDate,
	).Scan(&id)
	return id, err
}

func (r *TaskRepository) DeleteTask(ctx context.Context, taskID int) error {
	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, taskID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func (r *TaskRepository) GetTaskByID(ctx context.Context, taskID int) (*domain.Task, error) {
	query := `
		SELECT t.id, t.project_id, t.assignee_id, t.name, t.description, ts.name AS status_name, t.start_date, t.end_date,
		       pc.name AS priority_name, dc.name AS difficulty_name
		FROM tasks t
		JOIN task_statuses ts ON t.status_id = ts.id
		LEFT JOIN task_categories pc ON t.priority_category_id = pc.id
		LEFT JOIN task_categories dc ON t.difficulty_category_id = dc.id
		WHERE t.id = $1
	`

	var task domain.Task
	var startDate *time.Time
	var endDate *time.Time
	var assigneeID *int32
	var statusName string
	var priorityName *string
	var difficultyName *string

	err := r.pool.QueryRow(ctx, query, taskID).Scan(
		&task.ID,
		&task.ProjectID,
		&assigneeID,
		&task.Name,
		&task.Description,
		&statusName,
		&startDate,
		&endDate,
		&priorityName,
		&difficultyName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	task.AssigneeID = assigneeID
	task.StartDate = startDate
	if endDate != nil {
		task.EndDate = endDate
	}
	task.Status = taskStatusFromName(statusName)
	task.Priority = priorityFromCategory(ptrValue(priorityName))
	task.Difficulty = difficultyFromCategory(ptrValue(difficultyName))

	return &task, nil
}

func (r *TaskRepository) GetTasksByProjectAndAssignee(ctx context.Context, projectID int, assigneeID int) ([]domain.Task, error) {
	closedID, err := r.statusID(ctx, domain.TaskStatusClosed)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT t.id, t.project_id, t.assignee_id, t.name, t.description, ts.name AS status_name, t.start_date, t.end_date,
		       pc.name AS priority_name, dc.name AS difficulty_name
		FROM tasks t
		JOIN task_statuses ts ON t.status_id = ts.id
		LEFT JOIN task_categories pc ON t.priority_category_id = pc.id
		LEFT JOIN task_categories dc ON t.difficulty_category_id = dc.id
		WHERE t.project_id = $1 AND t.assignee_id = $2 AND t.status_id != $3
		ORDER BY CASE pc.name WHEN 'High' THEN 3 WHEN 'Medium' THEN 2 WHEN 'Low' THEN 1 ELSE 2 END DESC,
		         CASE dc.name WHEN 'Hard' THEN 3 WHEN 'Medium' THEN 2 WHEN 'Easy' THEN 1 ELSE 2 END DESC
	`

	rows, err := r.pool.Query(ctx, query, projectID, assigneeID, closedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTasks(rows)
}

func (r *TaskRepository) GetAllTasksByProject(ctx context.Context, projectID int) ([]domain.Task, error) {
	closedID, err := r.statusID(ctx, domain.TaskStatusClosed)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT t.id, t.project_id, t.assignee_id, t.name, t.description, ts.name AS status_name, t.start_date, t.end_date,
		       pc.name AS priority_name, dc.name AS difficulty_name
		FROM tasks t
		JOIN task_statuses ts ON t.status_id = ts.id
		LEFT JOIN task_categories pc ON t.priority_category_id = pc.id
		LEFT JOIN task_categories dc ON t.difficulty_category_id = dc.id
		WHERE t.project_id = $1 AND t.status_id != $2
		ORDER BY CASE pc.name WHEN 'High' THEN 3 WHEN 'Medium' THEN 2 WHEN 'Low' THEN 1 ELSE 2 END DESC,
		         CASE dc.name WHEN 'Hard' THEN 3 WHEN 'Medium' THEN 2 WHEN 'Easy' THEN 1 ELSE 2 END DESC
	`

	rows, err := r.pool.Query(ctx, query, projectID, closedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTasks(rows)
}

func (r *TaskRepository) GetClosedTasksByProject(ctx context.Context, projectID int) ([]domain.Task, error) {
	closedID, err := r.statusID(ctx, domain.TaskStatusClosed)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT t.id, t.project_id, t.assignee_id, t.name, t.description, ts.name AS status_name, t.start_date, t.end_date,
		       pc.name AS priority_name, dc.name AS difficulty_name
		FROM tasks t
		JOIN task_statuses ts ON t.status_id = ts.id
		LEFT JOIN task_categories pc ON t.priority_category_id = pc.id
		LEFT JOIN task_categories dc ON t.difficulty_category_id = dc.id
		WHERE t.project_id = $1 AND t.status_id = $2
		ORDER BY t.end_date DESC,
		         CASE pc.name WHEN 'High' THEN 3 WHEN 'Medium' THEN 2 WHEN 'Low' THEN 1 ELSE 2 END DESC,
		         CASE dc.name WHEN 'Hard' THEN 3 WHEN 'Medium' THEN 2 WHEN 'Easy' THEN 1 ELSE 2 END DESC
	`

	rows, err := r.pool.Query(ctx, query, projectID, closedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanTasks(rows)
}

func (r *TaskRepository) GetTaskByProjectAndName(ctx context.Context, projectID int32, name string) (*domain.Task, error) {
	query := `
		SELECT t.id, t.project_id, t.assignee_id, t.name, t.description, ts.name AS status_name, t.start_date, t.end_date,
		       pc.name AS priority_name, dc.name AS difficulty_name
		FROM tasks t
		JOIN task_statuses ts ON t.status_id = ts.id
		LEFT JOIN task_categories pc ON t.priority_category_id = pc.id
		LEFT JOIN task_categories dc ON t.difficulty_category_id = dc.id
		WHERE t.project_id = $1 AND t.name = $2
		ORDER BY t.id
		LIMIT 1
	`

	var task domain.Task
	var startDate *time.Time
	var endDate *time.Time
	var assigneeID *int32
	var statusName string
	var priorityName *string
	var difficultyName *string

	err := r.pool.QueryRow(ctx, query, projectID, name).Scan(
		&task.ID,
		&task.ProjectID,
		&assigneeID,
		&task.Name,
		&task.Description,
		&statusName,
		&startDate,
		&endDate,
		&priorityName,
		&difficultyName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	task.AssigneeID = assigneeID
	task.StartDate = startDate
	task.EndDate = endDate
	task.Status = taskStatusFromName(statusName)
	task.Priority = priorityFromCategory(ptrValue(priorityName))
	task.Difficulty = difficultyFromCategory(ptrValue(difficultyName))

	return &task, nil
}

func (r *TaskRepository) AssignTask(ctx context.Context, taskID int, assigneeID int) error {
	notStartedID, err := r.statusID(ctx, domain.TaskStatusNotStarted)
	if err != nil {
		return err
	}
	closedID, err := r.statusID(ctx, domain.TaskStatusClosed)
	if err != nil {
		return err
	}

	query := `
		UPDATE tasks
		SET assignee_id = $1,
			status_id = $2,
			start_date = NULL,
			end_date = NULL
		WHERE id = $3 AND status_id != $4
	`

	cmdTag, err := r.pool.Exec(ctx, query, assigneeID, notStartedID, taskID, closedID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func (r *TaskRepository) UnassignTask(ctx context.Context, taskID int) error {
	notStartedID, err := r.statusID(ctx, domain.TaskStatusNotStarted)
	if err != nil {
		return err
	}
	closedID, err := r.statusID(ctx, domain.TaskStatusClosed)
	if err != nil {
		return err
	}

	query := `
		UPDATE tasks
		SET assignee_id = NULL,
			status_id = $1,
			start_date = NULL,
			end_date = NULL
		WHERE id = $2 AND status_id != $3
	`

	cmdTag, err := r.pool.Exec(ctx, query, notStartedID, taskID, closedID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func (r *TaskRepository) UpdateTaskStatus(ctx context.Context, taskID int, status domain.TaskStatus) error {
	newStatusID, err := r.statusID(ctx, status)
	if err != nil {
		return err
	}
	closedID, err := r.statusID(ctx, domain.TaskStatusClosed)
	if err != nil {
		return err
	}

	var query string
	switch status {
	case domain.TaskStatusInWork:
		query = `
			UPDATE tasks
			SET status_id = $1,
				start_date = CASE WHEN start_date IS NULL THEN now() ELSE start_date END
			WHERE id = $2 AND status_id != $3
		`
	case domain.TaskStatusNotStarted:
		query = `
			UPDATE tasks
			SET status_id = $1,
				end_date = NULL
			WHERE id = $2 AND status_id != $3
		`
	case domain.TaskStatusClosed:
		query = `
			UPDATE tasks
			SET status_id = $1,
				end_date = now()
			WHERE id = $2 AND status_id != $3
		`
	default:
		query = `
			UPDATE tasks
			SET status_id = $1
			WHERE id = $2 AND status_id != $3
		`
	}

	cmdTag, err := r.pool.Exec(ctx, query, newStatusID, taskID, closedID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func (r *TaskRepository) UpdateTask(ctx context.Context, task domain.Task) error {
	difficultyID, err := r.categoryID(ctx, "difficulty", categoryNameForDifficulty(task.Difficulty))
	if err != nil {
		return err
	}
	priorityID, err := r.categoryID(ctx, "priority", categoryNameForPriority(task.Priority))
	if err != nil {
		return err
	}
	statusID, err := r.statusID(ctx, task.Status)
	if err != nil {
		return err
	}

	query := `
		UPDATE tasks
		SET project_id = $1,
			assignee_id = $2,
			name = $3,
			description = $4,
			difficulty_category_id = $5,
			priority_category_id = $6,
			status_id = $7,
			start_date = $8,
			end_date = $9
		WHERE id = $10
	`

	cmdTag, err := r.pool.Exec(ctx, query,
		task.ProjectID,
		task.AssigneeID,
		task.Name,
		task.Description,
		difficultyID,
		priorityID,
		statusID,
		task.StartDate,
		task.EndDate,
		task.ID,
	)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func (r *TaskRepository) CloseTask(ctx context.Context, taskID int) error {
	closedID, err := r.statusID(ctx, domain.TaskStatusClosed)
	if err != nil {
		return err
	}

	query := `
		UPDATE tasks
		SET status_id = $2,
			end_date = now()
		WHERE id = $1 AND status_id != $3
	`

	cmdTag, err := r.pool.Exec(ctx, query, taskID, closedID, closedID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("task not found")
	}

	return nil
}

func (r *TaskRepository) UnassignTasksByMemberAndProject(ctx context.Context, projectID int, userID int32) error {
	notStartedID, err := r.statusID(ctx, domain.TaskStatusNotStarted)
	if err != nil {
		return err
	}
	closedID, err := r.statusID(ctx, domain.TaskStatusClosed)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx, `
		UPDATE tasks
		SET assignee_id = NULL,
		    status_id = $1,
		    start_date = NULL,
		    end_date = NULL
		WHERE project_id = $2 AND assignee_id = $3 AND status_id != $4
	`, notStartedID, projectID, userID, closedID)
	return err
}

func scanTasks(rows interface {
	Next() bool
	Scan(dest ...any) error
	Close()
}) ([]domain.Task, error) {
	tasks := make([]domain.Task, 0)
	for rows.Next() {
		var task domain.Task
		var startDate *time.Time
		var endDate *time.Time
		var assigneeID *int32
		var statusName string
		var priorityName *string
		var difficultyName *string

		if err := rows.Scan(
			&task.ID,
			&task.ProjectID,
			&assigneeID,
			&task.Name,
			&task.Description,
			&statusName,
			&startDate,
			&endDate,
			&priorityName,
			&difficultyName,
		); err != nil {
			return nil, err
		}

		task.AssigneeID = assigneeID
		task.StartDate = startDate
		if endDate != nil {
			task.EndDate = endDate
		}
		task.Status = taskStatusFromName(statusName)
		task.Priority = priorityFromCategory(ptrValue(priorityName))
		task.Difficulty = difficultyFromCategory(ptrValue(difficultyName))

		tasks = append(tasks, task)
	}

	return tasks, nil
}

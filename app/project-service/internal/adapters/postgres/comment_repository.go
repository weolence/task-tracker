package postgres

import (
	"context"
	"errors"

	"project-service/internal/core/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommentRepository struct {
	pool *pgxpool.Pool
}

func NewCommentRepository(pool *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{pool: pool}
}

func (r *CommentRepository) CreateComment(ctx context.Context, comment domain.Comment) (int32, error) {
	query := `
		INSERT INTO comments (author_id, task_id, content, creation_date)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	var commentID int32
	err := r.pool.QueryRow(ctx, query,
		comment.AuthorID,
		comment.TaskID,
		comment.Content,
		comment.CreationDate,
	).Scan(&commentID)
	if err != nil {
		return 0, err
	}

	return commentID, nil
}

func (r *CommentRepository) DeleteComment(ctx context.Context, commentID int) error {
	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM comments WHERE id = $1`, commentID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("comment not found")
	}

	return nil
}

func (r *CommentRepository) GetCommentByID(ctx context.Context, commentID int32) (*domain.Comment, error) {
	query := `
		SELECT id, author_id, task_id, content, creation_date
		FROM comments
		WHERE id = $1
	`

	var comment domain.Comment
	err := r.pool.QueryRow(ctx, query, commentID).Scan(
		&comment.ID,
		&comment.AuthorID,
		&comment.TaskID,
		&comment.Content,
		&comment.CreationDate,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &comment, nil
}

func (r *CommentRepository) GetCommentsByAuthorID(ctx context.Context, authorID int) ([]domain.Comment, error) {
	query := `
		SELECT id, author_id, task_id, content, creation_date
		FROM comments
		WHERE author_id = $1
		ORDER BY creation_date, id
	`

	rows, err := r.pool.Query(ctx, query, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanComments(rows)
}

func (r *CommentRepository) GetCommentsByTaskID(ctx context.Context, taskID int) ([]domain.Comment, error) {
	query := `
		SELECT id, author_id, task_id, content, creation_date
		FROM comments
		WHERE task_id = $1
		ORDER BY creation_date, id
	`

	rows, err := r.pool.Query(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanComments(rows)
}

func (r *CommentRepository) UpdateComment(ctx context.Context, comment domain.Comment) error {
	query := `
		UPDATE comments
		SET author_id = $1,
			task_id = $2,
			content = $3,
			creation_date = $4
		WHERE id = $5
	`

	cmdTag, err := r.pool.Exec(ctx, query,
		comment.AuthorID,
		comment.TaskID,
		comment.Content,
		comment.CreationDate,
		comment.ID,
	)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("comment not found")
	}

	return nil
}

func scanComments(rows interface {
	Next() bool
	Scan(dest ...any) error
	Close()
}) ([]domain.Comment, error) {
	comments := make([]domain.Comment, 0)
	for rows.Next() {
		var comment domain.Comment
		if err := rows.Scan(
			&comment.ID,
			&comment.AuthorID,
			&comment.TaskID,
			&comment.Content,
			&comment.CreationDate,
		); err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	return comments, nil
}

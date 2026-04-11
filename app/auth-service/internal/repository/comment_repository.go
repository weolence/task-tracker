package repository

import (
	"auth-service/internal/model"
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type CommentRepository struct {
	Conn *pgx.Conn
}

func NewCommentRepository(ctx context.Context, dbLink string) (*CommentRepository, error) {
	conn, err := pgx.Connect(ctx, dbLink)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		conn.Close(ctx)
		return nil, err
	}

	query := `
	CREATE TABLE IF NOT EXISTS comments (
		id SERIAL PRIMARY KEY,
		author_id INT NOT NULL,
		task_id INT NOT NULL,
		content TEXT NOT NULL,
		creation_date TIMESTAMP NOT NULL
	);
	`

	_, err = conn.Exec(ctx, query)
	if err != nil {
		return nil, err
	}

	return &CommentRepository{
		Conn: conn,
	}, nil
}

func (commentRepository *CommentRepository) CreateComment(ctx context.Context, comment model.Comment) error {
	query := `
		INSERT INTO comments (author_id, task_id, content, creation_date)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	return commentRepository.Conn.QueryRow(ctx, query,
		comment.AuthorID,
		comment.TaskID,
		comment.Content,
		comment.CreationDate,
	).Scan(&comment.ID)
}

func (commentRepository *CommentRepository) DeleteComment(ctx context.Context, commentID int) error {
	query := `DELETE FROM comments WHERE id = $1`

	cmdTag, err := commentRepository.Conn.Exec(ctx, query, commentID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("comment not found")
	}

	return nil
}

func (commentRepository *CommentRepository) GetCommentsByAuthorID(ctx context.Context, authorID int) ([]model.Comment, error) {
	query := `
		SELECT id, author_id, task_id, content, creation_date
		FROM comments
		WHERE author_id = $1
		ORDER BY creation_date
	`

	rows, err := commentRepository.Conn.Query(ctx, query, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []model.Comment

	for rows.Next() {
		var comment model.Comment

		err := rows.Scan(
			&comment.ID,
			&comment.AuthorID,
			&comment.TaskID,
			&comment.Content,
			&comment.CreationDate,
		)
		if err != nil {
			return nil, err
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

func (commentRepository *CommentRepository) GetCommentsByTaskID(ctx context.Context, taskID int) ([]model.Comment, error) {
	query := `
		SELECT id, author_id, task_id, content, creation_date
		FROM comments
		WHERE task_id = $1
		ORDER BY creation_date
	`

	rows, err := commentRepository.Conn.Query(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []model.Comment

	for rows.Next() {
		var comment model.Comment

		err := rows.Scan(
			&comment.ID,
			&comment.AuthorID,
			&comment.TaskID,
			&comment.Content,
			&comment.CreationDate,
		)
		if err != nil {
			return nil, err
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

func (commentRepository *CommentRepository) ChangeContent(ctx context.Context, commentID int, newContent string) error {
	query := `
		UPDATE comments
		SET content = $1
		WHERE id = $2
	`

	cmdTag, err := commentRepository.Conn.Exec(ctx, query, newContent, commentID)
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return errors.New("comment not found")
	}

	return nil
}

package repository

import (
	"context"
	"errors"
	"project-service/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommentRepository struct {
	Conn *pgxpool.Pool
}

func NewCommentRepository(ctx context.Context, dbLink string) (*CommentRepository, error) {
	conn, err := pgxpool.New(ctx, dbLink)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		conn.Close()
		return nil, err
	}

	query := `
	CREATE TABLE IF NOT EXISTS comments (
		id SERIAL PRIMARY KEY,
		author_id INT NOT NULL,
		task_id INT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		content TEXT NOT NULL,
		creation_date TIMESTAMP NOT NULL
	);

	CREATE INDEX IF NOT EXISTS comments_task_id_idx ON comments(task_id);
	CREATE INDEX IF NOT EXISTS comments_author_id_idx ON comments(author_id);
	`

	_, err = conn.Exec(ctx, query)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &CommentRepository{
		Conn: conn,
	}, nil
}

func (commentRepository *CommentRepository) CreateComment(ctx context.Context, comment model.Comment) (int32, error) {
	query := `
		INSERT INTO comments (author_id, task_id, content, creation_date)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	var commentID int32
	err := commentRepository.Conn.QueryRow(ctx, query,
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

func (commentRepository *CommentRepository) GetCommentByID(ctx context.Context, commentID int32) (*model.Comment, error) {
	query := `
		SELECT id, author_id, task_id, content, creation_date
		FROM comments
		WHERE id = $1
	`

	var comment model.Comment
	err := commentRepository.Conn.QueryRow(ctx, query, commentID).Scan(
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

func (commentRepository *CommentRepository) GetCommentsByAuthorID(ctx context.Context, authorID int) ([]model.Comment, error) {
	query := `
		SELECT id, author_id, task_id, content, creation_date
		FROM comments
		WHERE author_id = $1
		ORDER BY creation_date, id
	`

	rows, err := commentRepository.Conn.Query(ctx, query, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := make([]model.Comment, 0)
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
		ORDER BY creation_date, id
	`

	rows, err := commentRepository.Conn.Query(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := make([]model.Comment, 0)
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

func (commentRepository *CommentRepository) UpdateComment(ctx context.Context, comment model.Comment) error {
	query := `
		UPDATE comments
		SET author_id = $1,
			task_id = $2,
			content = $3,
			creation_date = $4
		WHERE id = $5
	`

	cmdTag, err := commentRepository.Conn.Exec(ctx, query,
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

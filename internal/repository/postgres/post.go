package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"posts-service/internal/model"
)

type PostRepo struct {
	pool PgxPool
}

func NewPostRepo(pool PgxPool) *PostRepo {
	return &PostRepo{pool: pool}
}

const postColumns = "id, title, content, author, comments_disabled, created_at"

func scanPost(row pgx.Row) (*model.Post, error) {
	var p model.Post
	err := row.Scan(&p.ID, &p.Title, &p.Content, &p.Author, &p.CommentsDisabled, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrPostNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan post: %w", err)
	}
	return &p, nil
}

func (r *PostRepo) Create(ctx context.Context, post *model.Post) error {
	_, err := querier(ctx, r.pool).Exec(ctx,
		`INSERT INTO posts (id, title, content, author, comments_disabled, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		post.ID, post.Title, post.Content, post.Author, post.CommentsDisabled, post.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert post: %w", err)
	}
	return nil
}

func (r *PostRepo) GetByID(ctx context.Context, id string) (*model.Post, error) {
	row := querier(ctx, r.pool).QueryRow(ctx,
		`SELECT `+postColumns+` FROM posts WHERE id = $1`, id)
	return scanPost(row)
}

func (r *PostRepo) GetByIDForShare(ctx context.Context, id string) (*model.Post, error) {
	row := querier(ctx, r.pool).QueryRow(ctx,
		`SELECT `+postColumns+` FROM posts WHERE id = $1 FOR SHARE`, id)
	return scanPost(row)
}

func (r *PostRepo) List(ctx context.Context, page model.Page) ([]*model.Post, bool, error) {
	query := `SELECT ` + postColumns + ` FROM posts`
	args := []any{}
	if page.After != nil {
		query += ` WHERE (created_at, id) < ($1, $2)`
		args = append(args, page.After.CreatedAt, page.After.ID)
	}
	query += fmt.Sprintf(` ORDER BY created_at DESC, id DESC LIMIT %d`, page.Limit+1)

	rows, err := querier(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("select posts: %w", err)
	}
	defer rows.Close()

	posts := make([]*model.Post, 0, page.Limit)
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, false, err
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate posts: %w", err)
	}

	hasNext := len(posts) > page.Limit
	if hasNext {
		posts = posts[:page.Limit]
	}
	return posts, hasNext, nil
}

func (r *PostRepo) SetCommentsDisabled(ctx context.Context, postID string, disabled bool) (*model.Post, error) {
	row := querier(ctx, r.pool).QueryRow(ctx,
		`UPDATE posts SET comments_disabled = $2 WHERE id = $1 RETURNING `+postColumns, postID, disabled)
	return scanPost(row)
}

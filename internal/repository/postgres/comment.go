package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"posts-service/internal/model"
)

type CommentRepo struct {
	pool PgxPool
}

func NewCommentRepo(pool PgxPool) *CommentRepo {
	return &CommentRepo{pool: pool}
}

const commentColumns = "id, post_id, parent_id, author, text, created_at"

func scanComment(row pgx.Row) (*model.Comment, error) {
	var c model.Comment
	err := row.Scan(&c.ID, &c.PostID, &c.ParentID, &c.Author, &c.Text, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrCommentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan comment: %w", err)
	}
	return &c, nil
}

func (r *CommentRepo) Create(ctx context.Context, comment *model.Comment) error {
	_, err := querier(ctx, r.pool).Exec(ctx,
		`INSERT INTO comments (id, post_id, parent_id, author, text, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		comment.ID, comment.PostID, comment.ParentID, comment.Author, comment.Text, comment.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert comment: %w", err)
	}
	return nil
}

func (r *CommentRepo) GetByID(ctx context.Context, id string) (*model.Comment, error) {
	row := querier(ctx, r.pool).QueryRow(ctx,
		`SELECT `+commentColumns+` FROM comments WHERE id = $1`, id)
	return scanComment(row)
}

func (r *CommentRepo) ListTopLevel(ctx context.Context, postID string, page model.Page) ([]*model.Comment, bool, error) {
	query := `SELECT ` + commentColumns + ` FROM comments WHERE post_id = $1 AND parent_id IS NULL`
	args := []any{postID}
	if page.After != nil {
		query += ` AND (created_at, id) > ($2, $3)`
		args = append(args, page.After.CreatedAt, page.After.ID)
	}
	query += fmt.Sprintf(` ORDER BY created_at, id LIMIT %d`, page.Limit+1)
	return r.listPage(ctx, query, args, page.Limit)
}

func (r *CommentRepo) ListReplies(ctx context.Context, parentID string, page model.Page) ([]*model.Comment, bool, error) {
	query := `SELECT ` + commentColumns + ` FROM comments WHERE parent_id = $1`
	args := []any{parentID}
	if page.After != nil {
		query += ` AND (created_at, id) > ($2, $3)`
		args = append(args, page.After.CreatedAt, page.After.ID)
	}
	query += fmt.Sprintf(` ORDER BY created_at, id LIMIT %d`, page.Limit+1)
	return r.listPage(ctx, query, args, page.Limit)
}

func (r *CommentRepo) ListRepliesBatch(ctx context.Context, parentIDs []string, limit int) (map[string][]*model.Comment, map[string]bool, error) {

	rows, err := querier(ctx, r.pool).Query(ctx, fmt.Sprintf(
		`SELECT %s FROM (
		     SELECT %s, ROW_NUMBER() OVER (PARTITION BY parent_id ORDER BY created_at, id) AS rn
		     FROM comments WHERE parent_id = ANY($1)
		 ) ranked WHERE rn <= %d ORDER BY created_at, id`, commentColumns, commentColumns, limit+1), parentIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("select replies batch: %w", err)
	}
	defer rows.Close()

	items := make(map[string][]*model.Comment, len(parentIDs))
	hasNext := make(map[string]bool, len(parentIDs))
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, nil, err
		}
		parentID := *c.ParentID
		if len(items[parentID]) == limit {
			hasNext[parentID] = true
			continue
		}
		items[parentID] = append(items[parentID], c)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate replies batch: %w", err)
	}
	return items, hasNext, nil
}

func (r *CommentRepo) listPage(ctx context.Context, query string, args []any, limit int) ([]*model.Comment, bool, error) {
	rows, err := querier(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("select comments: %w", err)
	}
	defer rows.Close()

	comments := make([]*model.Comment, 0, limit)
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, false, err
		}
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate comments: %w", err)
	}

	hasNext := len(comments) > limit
	if hasNext {
		comments = comments[:limit]
	}
	return comments, hasNext, nil
}

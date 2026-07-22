package repository

import (
	"context"

	"posts-service/internal/model"
)

type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type PostRepository interface {
	Create(ctx context.Context, post *model.Post) error
	GetByID(ctx context.Context, id string) (*model.Post, error)

	GetByIDForShare(ctx context.Context, id string) (*model.Post, error)

	List(ctx context.Context, page model.Page) ([]*model.Post, bool, error)
	SetCommentsDisabled(ctx context.Context, postID string, disabled bool) (*model.Post, error)
}

type CommentRepository interface {
	Create(ctx context.Context, comment *model.Comment) error
	GetByID(ctx context.Context, id string) (*model.Comment, error)

	ListTopLevel(ctx context.Context, postID string, page model.Page) ([]*model.Comment, bool, error)

	ListReplies(ctx context.Context, parentID string, page model.Page) ([]*model.Comment, bool, error)

	ListRepliesBatch(ctx context.Context, parentIDs []string, limit int) (map[string][]*model.Comment, map[string]bool, error)
}

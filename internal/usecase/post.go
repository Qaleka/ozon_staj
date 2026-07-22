package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"posts-service/internal/model"
	"posts-service/internal/repository"
)

type PostUsecase struct {
	posts repository.PostRepository
}

func NewPostUsecase(posts repository.PostRepository) *PostUsecase {
	return &PostUsecase{posts: posts}
}

func (u *PostUsecase) CreatePost(ctx context.Context, title, content, author string, commentsDisabled bool) (*model.Post, error) {
	if strings.TrimSpace(title) == "" || strings.TrimSpace(author) == "" {
		return nil, model.ErrEmptyField
	}

	post := &model.Post{
		ID:               newID(),
		Title:            title,
		Content:          content,
		Author:           author,
		CommentsDisabled: commentsDisabled,
		CreatedAt:        time.Now().UTC(),
	}
	if err := u.posts.Create(ctx, post); err != nil {
		return nil, err
	}
	return post, nil
}

func (u *PostUsecase) GetPost(ctx context.Context, id string) (*model.Post, error) {
	return u.posts.GetByID(ctx, id)
}

func (u *PostUsecase) ListPosts(ctx context.Context, page model.Page) ([]*model.Post, bool, error) {
	return u.posts.List(ctx, page)
}

func (u *PostUsecase) SetCommentsDisabled(ctx context.Context, postID, author string, disabled bool) (*model.Post, error) {
	post, err := u.posts.GetByID(ctx, postID)
	if err != nil {
		return nil, err
	}
	if post.Author != author {
		return nil, model.ErrForbidden
	}
	return u.posts.SetCommentsDisabled(ctx, postID, disabled)
}

func newID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}

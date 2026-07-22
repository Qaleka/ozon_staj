package usecase

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"posts-service/internal/model"
	"posts-service/internal/repository"
)

type CommentPublisher interface {
	Publish(comment *model.Comment)
}

type CommentUsecase struct {
	comments  repository.CommentRepository
	posts     repository.PostRepository
	tx        repository.TxManager
	publisher CommentPublisher
}

func NewCommentUsecase(comments repository.CommentRepository, posts repository.PostRepository, tx repository.TxManager, publisher CommentPublisher) *CommentUsecase {
	return &CommentUsecase{comments: comments, posts: posts, tx: tx, publisher: publisher}
}

func (u *CommentUsecase) CreateComment(ctx context.Context, postID string, parentID *string, author, text string) (*model.Comment, error) {
	if strings.TrimSpace(author) == "" || strings.TrimSpace(text) == "" {
		return nil, model.ErrEmptyField
	}
	if utf8.RuneCountInString(text) > model.MaxCommentLength {
		return nil, model.ErrCommentTooLong
	}

	comment := &model.Comment{
		ID:        newID(),
		PostID:    postID,
		ParentID:  parentID,
		Author:    author,
		Text:      text,
		CreatedAt: time.Now().UTC(),
	}

	err := u.tx.WithinTx(ctx, func(ctx context.Context) error {
		post, err := u.posts.GetByIDForShare(ctx, postID)
		if err != nil {
			return err
		}
		if post.CommentsDisabled {
			return model.ErrCommentsDisabled
		}

		if parentID != nil {
			parent, err := u.comments.GetByID(ctx, *parentID)
			if err != nil {
				return err
			}
			if parent.PostID != postID {
				return model.ErrParentFromAnotherPost
			}
		}

		return u.comments.Create(ctx, comment)
	})
	if err != nil {
		return nil, err
	}

	u.publisher.Publish(comment)
	return comment, nil
}

func (u *CommentUsecase) ListTopLevel(ctx context.Context, postID string, page model.Page) ([]*model.Comment, bool, error) {
	if _, err := u.posts.GetByID(ctx, postID); err != nil {
		return nil, false, err
	}
	return u.comments.ListTopLevel(ctx, postID, page)
}

func (u *CommentUsecase) ListReplies(ctx context.Context, parentID string, page model.Page) ([]*model.Comment, bool, error) {
	return u.comments.ListReplies(ctx, parentID, page)
}

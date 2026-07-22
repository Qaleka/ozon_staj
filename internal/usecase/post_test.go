package usecase

import (
	"context"
	"errors"
	"testing"

	"posts-service/internal/model"
)

func TestCreatePost(t *testing.T) {
	u := NewPostUsecase(newMockPostRepo())

	post, err := u.CreatePost(context.Background(), "Title", "Content", "alice", false)
	if err != nil {
		t.Fatalf("CreatePost: unexpected error %v", err)
	}
	if post.ID == "" {
		t.Error("post ID must be generated")
	}
	if post.CreatedAt.IsZero() {
		t.Error("post CreatedAt must be set")
	}
	if post.CommentsDisabled {
		t.Error("commentsDisabled must be false by default")
	}
}

func TestCreatePost_EmptyFields(t *testing.T) {
	u := NewPostUsecase(newMockPostRepo())

	cases := []struct {
		name          string
		title, author string
	}{
		{"empty title", "", "alice"},
		{"blank title", "   ", "alice"},
		{"empty author", "Title", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := u.CreatePost(context.Background(), tc.title, "c", tc.author, false); !errors.Is(err, model.ErrEmptyField) {
				t.Errorf("expected ErrEmptyField, got %v", err)
			}
		})
	}
}

func TestGetPost_NotFound(t *testing.T) {
	u := NewPostUsecase(newMockPostRepo())

	if _, err := u.GetPost(context.Background(), "missing"); !errors.Is(err, model.ErrPostNotFound) {
		t.Errorf("expected ErrPostNotFound, got %v", err)
	}
}

func TestSetCommentsDisabled(t *testing.T) {
	u := NewPostUsecase(newMockPostRepo())
	post, _ := u.CreatePost(context.Background(), "Title", "Content", "alice", false)

	updated, err := u.SetCommentsDisabled(context.Background(), post.ID, "alice", true)
	if err != nil {
		t.Fatalf("SetCommentsDisabled: unexpected error %v", err)
	}
	if !updated.CommentsDisabled {
		t.Error("comments must be disabled")
	}
}

func TestSetCommentsDisabled_OnlyAuthor(t *testing.T) {
	u := NewPostUsecase(newMockPostRepo())
	post, _ := u.CreatePost(context.Background(), "Title", "Content", "alice", false)

	if _, err := u.SetCommentsDisabled(context.Background(), post.ID, "bob", true); !errors.Is(err, model.ErrForbidden) {
		t.Errorf("expected ErrForbidden for non-author, got %v", err)
	}
}

func TestSetCommentsDisabled_PostNotFound(t *testing.T) {
	u := NewPostUsecase(newMockPostRepo())

	if _, err := u.SetCommentsDisabled(context.Background(), "missing", "alice", true); !errors.Is(err, model.ErrPostNotFound) {
		t.Errorf("expected ErrPostNotFound, got %v", err)
	}
}

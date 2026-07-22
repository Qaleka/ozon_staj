package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"posts-service/internal/model"
)

type testEnv struct {
	comments    *CommentUsecase
	posts       *PostUsecase
	commentRepo *mockCommentRepo
	publisher   *mockPublisher
	tx          *mockTxManager
	post        *model.Post
}

func fixture(t *testing.T) testEnv {
	t.Helper()
	postRepo := newMockPostRepo()
	commentRepo := newMockCommentRepo()
	publisher := &mockPublisher{}
	tx := &mockTxManager{}
	postUC := NewPostUsecase(postRepo)
	commentUC := NewCommentUsecase(commentRepo, postRepo, tx, publisher)

	post, err := postUC.CreatePost(context.Background(), "Title", "Content", "alice", false)
	if err != nil {
		t.Fatalf("fixture: create post: %v", err)
	}
	return testEnv{
		comments:    commentUC,
		posts:       postUC,
		commentRepo: commentRepo,
		publisher:   publisher,
		tx:          tx,
		post:        post,
	}
}

func TestCreateComment(t *testing.T) {
	env := fixture(t)

	comment, err := env.comments.CreateComment(context.Background(), env.post.ID, nil, "bob", "hello")
	if err != nil {
		t.Fatalf("CreateComment: unexpected error %v", err)
	}
	if comment.PostID != env.post.ID {
		t.Errorf("PostID = %q, want %q", comment.PostID, env.post.ID)
	}
	if comment.ParentID != nil {
		t.Error("top-level comment must have nil ParentID")
	}
	if len(env.publisher.published) != 1 {
		t.Errorf("published %d comments, want 1", len(env.publisher.published))
	}

	if env.tx.calls != 1 {
		t.Errorf("WithinTx calls = %d, want 1", env.tx.calls)
	}
}

func TestCreateComment_Nested(t *testing.T) {
	env := fixture(t)

	parent, _ := env.comments.CreateComment(context.Background(), env.post.ID, nil, "bob", "parent")
	reply, err := env.comments.CreateComment(context.Background(), env.post.ID, &parent.ID, "carol", "reply")
	if err != nil {
		t.Fatalf("CreateComment reply: unexpected error %v", err)
	}
	if reply.ParentID == nil || *reply.ParentID != parent.ID {
		t.Error("reply must reference parent comment")
	}
}

func TestCreateComment_TooLong(t *testing.T) {
	env := fixture(t)

	long := strings.Repeat("я", model.MaxCommentLength+1)
	if _, err := env.comments.CreateComment(context.Background(), env.post.ID, nil, "bob", long); !errors.Is(err, model.ErrCommentTooLong) {
		t.Errorf("expected ErrCommentTooLong, got %v", err)
	}

	if env.tx.calls != 0 {
		t.Errorf("validation must happen before tx; WithinTx calls = %d, want 0", env.tx.calls)
	}

	exact := strings.Repeat("я", model.MaxCommentLength)
	if _, err := env.comments.CreateComment(context.Background(), env.post.ID, nil, "bob", exact); err != nil {
		t.Errorf("comment of exactly max length must be accepted, got %v", err)
	}
}

func TestCreateComment_Disabled(t *testing.T) {
	env := fixture(t)

	if _, err := env.posts.SetCommentsDisabled(context.Background(), env.post.ID, "alice", true); err != nil {
		t.Fatalf("disable comments: %v", err)
	}
	if _, err := env.comments.CreateComment(context.Background(), env.post.ID, nil, "bob", "hi"); !errors.Is(err, model.ErrCommentsDisabled) {
		t.Errorf("expected ErrCommentsDisabled, got %v", err)
	}
	if len(env.publisher.published) != 0 {
		t.Error("rejected comment must not be published")
	}
}

func TestCreateComment_PostNotFound(t *testing.T) {
	env := fixture(t)

	if _, err := env.comments.CreateComment(context.Background(), "missing", nil, "bob", "hi"); !errors.Is(err, model.ErrPostNotFound) {
		t.Errorf("expected ErrPostNotFound, got %v", err)
	}
}

func TestCreateComment_ParentNotFound(t *testing.T) {
	env := fixture(t)

	missing := "missing-parent"
	if _, err := env.comments.CreateComment(context.Background(), env.post.ID, &missing, "bob", "hi"); !errors.Is(err, model.ErrCommentNotFound) {
		t.Errorf("expected ErrCommentNotFound, got %v", err)
	}
}

func TestCreateComment_ParentFromAnotherPost(t *testing.T) {
	env := fixture(t)

	other, _ := env.posts.CreatePost(context.Background(), "Other", "Content", "alice", false)
	parent, _ := env.comments.CreateComment(context.Background(), other.ID, nil, "bob", "parent in other post")

	if _, err := env.comments.CreateComment(context.Background(), env.post.ID, &parent.ID, "bob", "hi"); !errors.Is(err, model.ErrParentFromAnotherPost) {
		t.Errorf("expected ErrParentFromAnotherPost, got %v", err)
	}
}

func TestCreateComment_EmptyFields(t *testing.T) {
	env := fixture(t)

	if _, err := env.comments.CreateComment(context.Background(), env.post.ID, nil, "", "hi"); !errors.Is(err, model.ErrEmptyField) {
		t.Errorf("expected ErrEmptyField for empty author, got %v", err)
	}
	if _, err := env.comments.CreateComment(context.Background(), env.post.ID, nil, "bob", "   "); !errors.Is(err, model.ErrEmptyField) {
		t.Errorf("expected ErrEmptyField for blank text, got %v", err)
	}
}

func TestCreateComment_NotPublishedOnStorageError(t *testing.T) {
	env := fixture(t)
	env.commentRepo.createErr = errors.New("db is down")

	_, err := env.comments.CreateComment(context.Background(), env.post.ID, nil, "bob", "hi")
	if err == nil {
		t.Fatal("expected error from failing storage")
	}
	if len(env.publisher.published) != 0 {
		t.Error("comment must not be published when the transaction fails")
	}
}

func TestListTopLevel_PostNotFound(t *testing.T) {
	env := fixture(t)

	if _, _, err := env.comments.ListTopLevel(context.Background(), "missing", model.Page{Limit: 10}); !errors.Is(err, model.ErrPostNotFound) {
		t.Errorf("expected ErrPostNotFound, got %v", err)
	}
}

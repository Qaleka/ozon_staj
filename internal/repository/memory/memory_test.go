package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"posts-service/internal/model"
)

func newPost(id string, at time.Time) *model.Post {
	return &model.Post{ID: id, Title: "t", Content: "c", Author: "a", CreatedAt: at}
}

func newComment(id, postID string, parentID *string, at time.Time) *model.Comment {
	return &model.Comment{ID: id, PostID: postID, ParentID: parentID, Author: "a", Text: "x", CreatedAt: at}
}

func TestPostRepo_ListPagination(t *testing.T) {
	repo := NewPostRepo()
	ctx := context.Background()
	base := time.Now().UTC()

	for i := 0; i < 5; i++ {
		if err := repo.Create(ctx, newPost(fmt.Sprintf("p%d", i), base.Add(time.Duration(i)*time.Second))); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	page1, hasNext, err := repo.List(ctx, model.Page{Limit: 2})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !hasNext {
		t.Error("page1: hasNext must be true")
	}
	if len(page1) != 2 || page1[0].ID != "p4" || page1[1].ID != "p3" {
		t.Fatalf("page1: got %v, want [p4 p3]", ids(page1))
	}

	cursor := model.Cursor{CreatedAt: page1[1].CreatedAt, ID: page1[1].ID}
	page2, hasNext, err := repo.List(ctx, model.Page{Limit: 2, After: &cursor})
	if err != nil {
		t.Fatalf("list page2: %v", err)
	}
	if !hasNext {
		t.Error("page2: hasNext must be true")
	}
	if len(page2) != 2 || page2[0].ID != "p2" || page2[1].ID != "p1" {
		t.Fatalf("page2: got %v, want [p2 p1]", ids(page2))
	}

	cursor = model.Cursor{CreatedAt: page2[1].CreatedAt, ID: page2[1].ID}
	page3, hasNext, err := repo.List(ctx, model.Page{Limit: 2, After: &cursor})
	if err != nil {
		t.Fatalf("list page3: %v", err)
	}
	if hasNext {
		t.Error("page3: hasNext must be false")
	}
	if len(page3) != 1 || page3[0].ID != "p0" {
		t.Fatalf("page3: got %v, want [p0]", ids(page3))
	}
}

func TestPostRepo_GetByID_NotFound(t *testing.T) {
	repo := NewPostRepo()
	if _, err := repo.GetByID(context.Background(), "missing"); !errors.Is(err, model.ErrPostNotFound) {
		t.Errorf("expected ErrPostNotFound, got %v", err)
	}
}

func TestPostRepo_SetCommentsDisabled(t *testing.T) {
	repo := NewPostRepo()
	ctx := context.Background()
	_ = repo.Create(ctx, newPost("p1", time.Now()))

	updated, err := repo.SetCommentsDisabled(ctx, "p1", true)
	if err != nil {
		t.Fatalf("SetCommentsDisabled: %v", err)
	}
	if !updated.CommentsDisabled {
		t.Error("comments must be disabled")
	}

	stored, _ := repo.GetByID(ctx, "p1")
	if !stored.CommentsDisabled {
		t.Error("change must be persisted")
	}
}

func TestCommentRepo_TopLevelPagination(t *testing.T) {
	repo := NewCommentRepo()
	ctx := context.Background()
	base := time.Now().UTC()

	for i := 0; i < 5; i++ {
		c := newComment(fmt.Sprintf("c%d", i), "post", nil, base.Add(time.Duration(i)*time.Second))
		if err := repo.Create(ctx, c); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	page1, hasNext, err := repo.ListTopLevel(ctx, "post", model.Page{Limit: 3})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !hasNext {
		t.Error("page1: hasNext must be true")
	}
	if len(page1) != 3 || page1[0].ID != "c0" || page1[2].ID != "c2" {
		t.Fatalf("page1: got %v, want [c0 c1 c2]", cids(page1))
	}

	cursor := model.Cursor{CreatedAt: page1[2].CreatedAt, ID: page1[2].ID}
	page2, hasNext, err := repo.ListTopLevel(ctx, "post", model.Page{Limit: 3, After: &cursor})
	if err != nil {
		t.Fatalf("list page2: %v", err)
	}
	if hasNext {
		t.Error("page2: hasNext must be false")
	}
	if len(page2) != 2 || page2[0].ID != "c3" || page2[1].ID != "c4" {
		t.Fatalf("page2: got %v, want [c3 c4]", cids(page2))
	}
}

func TestCommentRepo_RepliesSeparatedFromTopLevel(t *testing.T) {
	repo := NewCommentRepo()
	ctx := context.Background()
	now := time.Now().UTC()

	parent := newComment("parent", "post", nil, now)
	_ = repo.Create(ctx, parent)
	_ = repo.Create(ctx, newComment("reply1", "post", &parent.ID, now.Add(time.Second)))
	_ = repo.Create(ctx, newComment("reply2", "post", &parent.ID, now.Add(2*time.Second)))

	top, _, _ := repo.ListTopLevel(ctx, "post", model.Page{Limit: 10})
	if len(top) != 1 || top[0].ID != "parent" {
		t.Fatalf("top-level: got %v, want [parent]", cids(top))
	}

	replies, hasNext, _ := repo.ListReplies(ctx, "parent", model.Page{Limit: 10})
	if hasNext {
		t.Error("replies: hasNext must be false")
	}
	if len(replies) != 2 || replies[0].ID != "reply1" || replies[1].ID != "reply2" {
		t.Fatalf("replies: got %v, want [reply1 reply2]", cids(replies))
	}
}

func TestCommentRepo_ListRepliesBatch(t *testing.T) {
	repo := NewCommentRepo()
	ctx := context.Background()
	now := time.Now().UTC()

	p1, p2 := "parent1", "parent2"
	_ = repo.Create(ctx, newComment(p1, "post", nil, now))
	_ = repo.Create(ctx, newComment(p2, "post", nil, now))
	for i := 0; i < 3; i++ {
		_ = repo.Create(ctx, newComment(fmt.Sprintf("r1-%d", i), "post", &p1, now.Add(time.Duration(i)*time.Second)))
	}
	_ = repo.Create(ctx, newComment("r2-0", "post", &p2, now))

	items, hasNext, err := repo.ListRepliesBatch(ctx, []string{p1, p2, "empty"}, 2)
	if err != nil {
		t.Fatalf("batch: %v", err)
	}
	if len(items[p1]) != 2 || !hasNext[p1] {
		t.Errorf("parent1: got %d replies, hasNext=%v; want 2, true", len(items[p1]), hasNext[p1])
	}
	if len(items[p2]) != 1 || hasNext[p2] {
		t.Errorf("parent2: got %d replies, hasNext=%v; want 1, false", len(items[p2]), hasNext[p2])
	}
	if len(items["empty"]) != 0 || hasNext["empty"] {
		t.Errorf("empty parent must have no replies")
	}
}

func TestTxManager_NoOp(t *testing.T) {
	ctx := context.Background()
	called := false
	err := NewTxManager().WithinTx(ctx, func(context.Context) error {
		called = true
		return nil
	})
	if err != nil || !called {
		t.Errorf("WithinTx must run fn once; called=%v err=%v", called, err)
	}

	sentinel := errors.New("boom")
	if got := NewTxManager().WithinTx(ctx, func(context.Context) error { return sentinel }); got != sentinel {
		t.Errorf("WithinTx must propagate fn error, got %v", got)
	}
}

func TestPostRepo_GetByIDForShare(t *testing.T) {
	repo := NewPostRepo()
	ctx := context.Background()
	_ = repo.Create(ctx, newPost("p1", time.Now()))

	got, err := repo.GetByIDForShare(ctx, "p1")
	if err != nil || got.ID != "p1" {
		t.Errorf("GetByIDForShare: got %+v err %v", got, err)
	}
	if _, err := repo.GetByIDForShare(ctx, "missing"); !errors.Is(err, model.ErrPostNotFound) {
		t.Errorf("expected ErrPostNotFound, got %v", err)
	}
}

func TestConcurrentAccess(t *testing.T) {
	postRepo := NewPostRepo()
	commentRepo := NewCommentRepo()
	ctx := context.Background()
	_ = postRepo.Create(ctx, newPost("post", time.Now()))

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			_ = commentRepo.Create(ctx, newComment(fmt.Sprintf("c%d", i), "post", nil, time.Now().UTC()))
		}(i)
		go func() {
			defer wg.Done()
			_, _, _ = commentRepo.ListTopLevel(ctx, "post", model.Page{Limit: 10})
		}()
	}
	wg.Wait()

	all, _, err := commentRepo.ListTopLevel(ctx, "post", model.Page{Limit: 100})
	if err != nil {
		t.Fatalf("list after concurrent writes: %v", err)
	}
	if len(all) != 50 {
		t.Errorf("got %d comments, want 50", len(all))
	}
}

func ids(posts []*model.Post) []string {
	r := make([]string, len(posts))
	for i, p := range posts {
		r[i] = p.ID
	}
	return r
}

func cids(comments []*model.Comment) []string {
	r := make([]string, len(comments))
	for i, c := range comments {
		r[i] = c.ID
	}
	return r
}

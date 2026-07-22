package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"

	"posts-service/internal/model"
)

var commentCols = []string{"id", "post_id", "parent_id", "author", "text", "created_at"}

func TestCommentRepo_Create(t *testing.T) {
	mock := newMockPool(t)
	repo := NewCommentRepo(mock)
	c := &model.Comment{ID: "c1", PostID: "p1", Author: "bob", Text: "hi", CreatedAt: time.Now()}

	mock.ExpectExec("INSERT INTO comments").
		WithArgs(c.ID, c.PostID, c.ParentID, c.Author, c.Text, c.CreatedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	if err := repo.Create(context.Background(), c); err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestCommentRepo_GetByID_NotFound(t *testing.T) {
	mock := newMockPool(t)
	repo := NewCommentRepo(mock)

	mock.ExpectQuery("SELECT .+ FROM comments WHERE id = ").
		WithArgs("missing").
		WillReturnError(pgx.ErrNoRows)

	if _, err := repo.GetByID(context.Background(), "missing"); err != model.ErrCommentNotFound {
		t.Errorf("expected ErrCommentNotFound, got %v", err)
	}
}

func TestCommentRepo_ListTopLevel_HasNext(t *testing.T) {
	mock := newMockPool(t)
	repo := NewCommentRepo(mock)
	now := time.Now()

	rows := pgxmock.NewRows(commentCols).
		AddRow("c1", "p1", nil, "a", "t", now).
		AddRow("c2", "p1", nil, "a", "t", now.Add(time.Second)).
		AddRow("c3", "p1", nil, "a", "t", now.Add(2*time.Second))
	mock.ExpectQuery("FROM comments WHERE post_id = .+ parent_id IS NULL").
		WithArgs("p1").
		WillReturnRows(rows)

	list, hasNext, err := repo.ListTopLevel(context.Background(), "p1", model.Page{Limit: 2})
	if err != nil {
		t.Fatalf("ListTopLevel: %v", err)
	}
	if !hasNext {
		t.Error("expected hasNext=true")
	}
	if len(list) != 2 || list[0].ID != "c1" || list[1].ID != "c2" {
		t.Errorf("unexpected page: %v", commentIDs(list))
	}
}

func TestCommentRepo_ListReplies(t *testing.T) {
	mock := newMockPool(t)
	repo := NewCommentRepo(mock)
	now := time.Now()
	parent := "c1"

	rows := pgxmock.NewRows(commentCols).
		AddRow("r1", "p1", &parent, "a", "t", now)
	mock.ExpectQuery("FROM comments WHERE parent_id = ").
		WithArgs(parent).
		WillReturnRows(rows)

	list, hasNext, err := repo.ListReplies(context.Background(), parent, model.Page{Limit: 5})
	if err != nil {
		t.Fatalf("ListReplies: %v", err)
	}
	if hasNext {
		t.Error("expected hasNext=false")
	}
	if len(list) != 1 || list[0].ParentID == nil || *list[0].ParentID != parent {
		t.Errorf("unexpected replies: %v", commentIDs(list))
	}
}

func TestCommentRepo_ListRepliesBatch(t *testing.T) {
	mock := newMockPool(t)
	repo := NewCommentRepo(mock)
	now := time.Now()
	p1, p2 := "c1", "c2"

	rows := pgxmock.NewRows(commentCols).
		AddRow("r1", "p", &p1, "a", "t", now).
		AddRow("r2", "p", &p1, "a", "t", now.Add(time.Second)).
		AddRow("r3", "p", &p1, "a", "t", now.Add(2*time.Second)).
		AddRow("r4", "p", &p2, "a", "t", now)
	mock.ExpectQuery("ROW_NUMBER\\(\\) OVER \\(PARTITION BY parent_id").
		WithArgs([]string{p1, p2}).
		WillReturnRows(rows)

	items, hasNext, err := repo.ListRepliesBatch(context.Background(), []string{p1, p2}, 2)
	if err != nil {
		t.Fatalf("ListRepliesBatch: %v", err)
	}
	if len(items[p1]) != 2 || !hasNext[p1] {
		t.Errorf("p1: got %d hasNext=%v, want 2 true", len(items[p1]), hasNext[p1])
	}
	if len(items[p2]) != 1 || hasNext[p2] {
		t.Errorf("p2: got %d hasNext=%v, want 1 false", len(items[p2]), hasNext[p2])
	}
}

func TestCommentRepo_ListTopLevel_QueryError(t *testing.T) {
	mock := newMockPool(t)
	repo := NewCommentRepo(mock)

	mock.ExpectQuery("FROM comments WHERE post_id = ").
		WithArgs("p1").
		WillReturnError(pgx.ErrTxClosed)

	if _, _, err := repo.ListTopLevel(context.Background(), "p1", model.Page{Limit: 5}); err == nil {
		t.Error("expected error to be propagated")
	}
}

func commentIDs(comments []*model.Comment) []string {
	r := make([]string, len(comments))
	for i, c := range comments {
		r[i] = c.ID
	}
	return r
}

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"

	"posts-service/internal/model"
)

func newMockPool(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("new mock pool: %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet expectations: %v", err)
		}
		mock.Close()
	})
	return mock
}

var postCols = []string{"id", "title", "content", "author", "comments_disabled", "created_at"}

func TestPostRepo_Create(t *testing.T) {
	mock := newMockPool(t)
	repo := NewPostRepo(mock)
	post := &model.Post{ID: "p1", Title: "t", Content: "c", Author: "alice", CreatedAt: time.Now()}

	mock.ExpectExec("INSERT INTO posts").
		WithArgs(post.ID, post.Title, post.Content, post.Author, post.CommentsDisabled, post.CreatedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	if err := repo.Create(context.Background(), post); err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestPostRepo_GetByID(t *testing.T) {
	mock := newMockPool(t)
	repo := NewPostRepo(mock)
	now := time.Now()

	mock.ExpectQuery("SELECT .+ FROM posts WHERE id = ").
		WithArgs("p1").
		WillReturnRows(pgxmock.NewRows(postCols).AddRow("p1", "t", "c", "alice", false, now))

	got, err := repo.GetByID(context.Background(), "p1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != "p1" || got.Author != "alice" {
		t.Errorf("unexpected post: %+v", got)
	}
}

func TestPostRepo_GetByID_NotFound(t *testing.T) {
	mock := newMockPool(t)
	repo := NewPostRepo(mock)

	mock.ExpectQuery("SELECT .+ FROM posts WHERE id = ").
		WithArgs("missing").
		WillReturnError(pgx.ErrNoRows)

	if _, err := repo.GetByID(context.Background(), "missing"); err != model.ErrPostNotFound {
		t.Errorf("expected ErrPostNotFound, got %v", err)
	}
}

func TestPostRepo_GetByIDForShare(t *testing.T) {
	mock := newMockPool(t)
	repo := NewPostRepo(mock)
	now := time.Now()

	mock.ExpectQuery("FOR SHARE").
		WithArgs("p1").
		WillReturnRows(pgxmock.NewRows(postCols).AddRow("p1", "t", "c", "alice", false, now))

	if _, err := repo.GetByIDForShare(context.Background(), "p1"); err != nil {
		t.Fatalf("GetByIDForShare: %v", err)
	}
}

func TestPostRepo_List_FirstPage(t *testing.T) {
	mock := newMockPool(t)
	repo := NewPostRepo(mock)
	now := time.Now()

	rows := pgxmock.NewRows(postCols).
		AddRow("p3", "t", "c", "a", false, now.Add(2*time.Second)).
		AddRow("p2", "t", "c", "a", false, now.Add(time.Second)).
		AddRow("p1", "t", "c", "a", false, now)
	mock.ExpectQuery("SELECT .+ FROM posts").WillReturnRows(rows)

	posts, hasNext, err := repo.List(context.Background(), model.Page{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !hasNext {
		t.Error("expected hasNext=true when extra row returned")
	}
	if len(posts) != 2 || posts[0].ID != "p3" || posts[1].ID != "p2" {
		t.Errorf("unexpected page: %v", postIDs(posts))
	}
}

func TestPostRepo_List_LastPage(t *testing.T) {
	mock := newMockPool(t)
	repo := NewPostRepo(mock)
	cur := model.Cursor{CreatedAt: time.Now(), ID: "p9"}

	rows := pgxmock.NewRows(postCols).AddRow("p1", "t", "c", "a", false, time.Now())
	mock.ExpectQuery("SELECT .+ FROM posts").
		WithArgs(cur.CreatedAt, cur.ID).
		WillReturnRows(rows)

	posts, hasNext, err := repo.List(context.Background(), model.Page{Limit: 2, After: &cur})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if hasNext {
		t.Error("expected hasNext=false on last page")
	}
	if len(posts) != 1 {
		t.Errorf("expected 1 post, got %d", len(posts))
	}
}

func TestPostRepo_SetCommentsDisabled(t *testing.T) {
	mock := newMockPool(t)
	repo := NewPostRepo(mock)
	now := time.Now()

	mock.ExpectQuery("UPDATE posts SET comments_disabled").
		WithArgs("p1", true).
		WillReturnRows(pgxmock.NewRows(postCols).AddRow("p1", "t", "c", "alice", true, now))

	got, err := repo.SetCommentsDisabled(context.Background(), "p1", true)
	if err != nil {
		t.Fatalf("SetCommentsDisabled: %v", err)
	}
	if !got.CommentsDisabled {
		t.Error("expected comments disabled")
	}
}

func TestPostRepo_SetCommentsDisabled_NotFound(t *testing.T) {
	mock := newMockPool(t)
	repo := NewPostRepo(mock)

	mock.ExpectQuery("UPDATE posts SET comments_disabled").
		WithArgs("missing", true).
		WillReturnError(pgx.ErrNoRows)

	if _, err := repo.SetCommentsDisabled(context.Background(), "missing", true); err != model.ErrPostNotFound {
		t.Errorf("expected ErrPostNotFound, got %v", err)
	}
}

func postIDs(posts []*model.Post) []string {
	r := make([]string, len(posts))
	for i, p := range posts {
		r[i] = p.ID
	}
	return r
}

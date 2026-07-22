package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"

	"posts-service/internal/model"
)

func samplePost() *model.Post {
	return &model.Post{ID: "p1", Title: "t", Content: "c", Author: "a", CreatedAt: time.Now()}
}

func expectInsert(mock pgxmock.PgxPoolIface, p *model.Post) {
	mock.ExpectExec("INSERT INTO posts").
		WithArgs(p.ID, p.Title, p.Content, p.Author, p.CommentsDisabled, p.CreatedAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
}

func TestTxManager_Commit(t *testing.T) {
	mock := newMockPool(t)
	repo := NewPostRepo(mock)
	tx := NewTxManager(mock)
	post := samplePost()

	mock.ExpectBegin()
	expectInsert(mock, post)
	mock.ExpectCommit()

	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		return repo.Create(ctx, post)
	})
	if err != nil {
		t.Fatalf("WithinTx: %v", err)
	}
}

func TestTxManager_Rollback(t *testing.T) {
	mock := newMockPool(t)
	repo := NewPostRepo(mock)
	tx := NewTxManager(mock)
	post := samplePost()
	sentinel := errors.New("boom")

	mock.ExpectBegin()
	expectInsert(mock, post)
	mock.ExpectRollback()

	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		if err := repo.Create(ctx, post); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}

func TestTxManager_BeginError(t *testing.T) {
	mock := newMockPool(t)
	tx := NewTxManager(mock)

	mock.ExpectBegin().WillReturnError(errors.New("cannot begin"))

	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		t.Fatal("fn must not be called when Begin fails")
		return nil
	})
	if err == nil {
		t.Fatal("expected error from Begin")
	}
}

func TestTxManager_NestedReusesTx(t *testing.T) {
	mock := newMockPool(t)
	repo := NewPostRepo(mock)
	tx := NewTxManager(mock)
	post := samplePost()

	mock.ExpectBegin()
	expectInsert(mock, post)
	mock.ExpectCommit()

	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		return tx.WithinTx(ctx, func(ctx context.Context) error {
			return repo.Create(ctx, post)
		})
	})
	if err != nil {
		t.Fatalf("WithinTx: %v", err)
	}
}

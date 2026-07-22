package usecase

import (
	"context"

	"posts-service/internal/model"
)

type mockPostRepo struct {
	posts map[string]*model.Post
}

func newMockPostRepo() *mockPostRepo {
	return &mockPostRepo{posts: make(map[string]*model.Post)}
}

func (m *mockPostRepo) Create(_ context.Context, post *model.Post) error {
	m.posts[post.ID] = post
	return nil
}

func (m *mockPostRepo) GetByID(_ context.Context, id string) (*model.Post, error) {
	post, ok := m.posts[id]
	if !ok {
		return nil, model.ErrPostNotFound
	}
	return post, nil
}

func (m *mockPostRepo) GetByIDForShare(ctx context.Context, id string) (*model.Post, error) {
	return m.GetByID(ctx, id)
}

func (m *mockPostRepo) List(_ context.Context, _ model.Page) ([]*model.Post, bool, error) {
	result := make([]*model.Post, 0, len(m.posts))
	for _, p := range m.posts {
		result = append(result, p)
	}
	return result, false, nil
}

func (m *mockPostRepo) SetCommentsDisabled(_ context.Context, postID string, disabled bool) (*model.Post, error) {
	post, ok := m.posts[postID]
	if !ok {
		return nil, model.ErrPostNotFound
	}
	post.CommentsDisabled = disabled
	return post, nil
}

type mockCommentRepo struct {
	comments  map[string]*model.Comment
	createErr error
}

func newMockCommentRepo() *mockCommentRepo {
	return &mockCommentRepo{comments: make(map[string]*model.Comment)}
}

func (m *mockCommentRepo) Create(_ context.Context, comment *model.Comment) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.comments[comment.ID] = comment
	return nil
}

func (m *mockCommentRepo) GetByID(_ context.Context, id string) (*model.Comment, error) {
	comment, ok := m.comments[id]
	if !ok {
		return nil, model.ErrCommentNotFound
	}
	return comment, nil
}

func (m *mockCommentRepo) ListTopLevel(_ context.Context, postID string, _ model.Page) ([]*model.Comment, bool, error) {
	var result []*model.Comment
	for _, c := range m.comments {
		if c.PostID == postID && c.ParentID == nil {
			result = append(result, c)
		}
	}
	return result, false, nil
}

func (m *mockCommentRepo) ListReplies(_ context.Context, parentID string, _ model.Page) ([]*model.Comment, bool, error) {
	var result []*model.Comment
	for _, c := range m.comments {
		if c.ParentID != nil && *c.ParentID == parentID {
			result = append(result, c)
		}
	}
	return result, false, nil
}

func (m *mockCommentRepo) ListRepliesBatch(_ context.Context, _ []string, _ int) (map[string][]*model.Comment, map[string]bool, error) {
	return nil, nil, nil
}

type mockPublisher struct {
	published []*model.Comment
}

func (m *mockPublisher) Publish(comment *model.Comment) {
	m.published = append(m.published, comment)
}

type mockTxManager struct {
	calls int
}

func (m *mockTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	m.calls++
	return fn(ctx)
}

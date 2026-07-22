package memory

import (
	"context"
	"sync"

	"posts-service/internal/model"
)

type CommentRepo struct {
	mu       sync.RWMutex
	comments map[string]*model.Comment
	topLevel map[string][]entry
	replies  map[string][]entry
}

func NewCommentRepo() *CommentRepo {
	return &CommentRepo{
		comments: make(map[string]*model.Comment),
		topLevel: make(map[string][]entry),
		replies:  make(map[string][]entry),
	}
}

func (r *CommentRepo) Create(_ context.Context, comment *model.Comment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	c := *comment
	r.comments[c.ID] = &c
	e := entry{createdAt: c.CreatedAt, id: c.ID}
	if c.ParentID == nil {
		r.topLevel[c.PostID] = insertSorted(r.topLevel[c.PostID], e)
	} else {
		r.replies[*c.ParentID] = insertSorted(r.replies[*c.ParentID], e)
	}
	return nil
}

func (r *CommentRepo) GetByID(_ context.Context, id string) (*model.Comment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	comment, ok := r.comments[id]
	if !ok {
		return nil, model.ErrCommentNotFound
	}
	c := *comment
	return &c, nil
}

func (r *CommentRepo) ListTopLevel(_ context.Context, postID string, page model.Page) ([]*model.Comment, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries, hasNext := pageAfter(r.topLevel[postID], page)
	return r.collect(entries), hasNext, nil
}

func (r *CommentRepo) ListReplies(_ context.Context, parentID string, page model.Page) ([]*model.Comment, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries, hasNext := pageAfter(r.replies[parentID], page)
	return r.collect(entries), hasNext, nil
}

func (r *CommentRepo) ListRepliesBatch(_ context.Context, parentIDs []string, limit int) (map[string][]*model.Comment, map[string]bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make(map[string][]*model.Comment, len(parentIDs))
	hasNext := make(map[string]bool, len(parentIDs))
	for _, parentID := range parentIDs {
		entries, more := pageAfter(r.replies[parentID], model.Page{Limit: limit})
		items[parentID] = r.collect(entries)
		hasNext[parentID] = more
	}
	return items, hasNext, nil
}

func (r *CommentRepo) collect(entries []entry) []*model.Comment {
	result := make([]*model.Comment, 0, len(entries))
	for _, e := range entries {
		c := *r.comments[e.id]
		result = append(result, &c)
	}
	return result
}

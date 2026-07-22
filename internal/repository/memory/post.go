package memory

import (
	"context"
	"sort"
	"sync"

	"posts-service/internal/model"
)

type PostRepo struct {
	mu    sync.RWMutex
	posts map[string]*model.Post
	order []entry
}

func NewPostRepo() *PostRepo {
	return &PostRepo{posts: make(map[string]*model.Post)}
}

func (r *PostRepo) Create(_ context.Context, post *model.Post) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p := *post
	r.posts[p.ID] = &p
	r.order = insertSorted(r.order, entry{createdAt: p.CreatedAt, id: p.ID})
	return nil
}

func (r *PostRepo) GetByID(_ context.Context, id string) (*model.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	post, ok := r.posts[id]
	if !ok {
		return nil, model.ErrPostNotFound
	}
	p := *post
	return &p, nil
}

func (r *PostRepo) GetByIDForShare(ctx context.Context, id string) (*model.Post, error) {
	return r.GetByID(ctx, id)
}

func (r *PostRepo) List(_ context.Context, page model.Page) ([]*model.Post, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	end := len(r.order)
	if page.After != nil {
		c := entry{createdAt: page.After.CreatedAt, id: page.After.ID}
		end = sort.Search(len(r.order), func(i int) bool { return !r.order[i].less(c) })
	}
	start := end - page.Limit
	hasNext := start > 0
	if start < 0 {
		start = 0
	}

	result := make([]*model.Post, 0, end-start)
	for i := end - 1; i >= start; i-- {
		p := *r.posts[r.order[i].id]
		result = append(result, &p)
	}
	return result, hasNext, nil
}

func (r *PostRepo) SetCommentsDisabled(_ context.Context, postID string, disabled bool) (*model.Post, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	post, ok := r.posts[postID]
	if !ok {
		return nil, model.ErrPostNotFound
	}
	post.CommentsDisabled = disabled
	p := *post
	return &p, nil
}

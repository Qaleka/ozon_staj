package dataloader

import (
	"context"
	"net/http"
	"time"

	"github.com/vikstrous/dataloadgen"

	"posts-service/internal/model"
	"posts-service/internal/repository"
)

type ctxKey struct{}

type repliesKey struct {
	ParentID string
	First    int
}

type RepliesPage struct {
	Items   []*model.Comment
	HasNext bool
}

type Loaders struct {
	replies *dataloadgen.Loader[repliesKey, RepliesPage]
}

func NewLoaders(comments repository.CommentRepository) *Loaders {
	f := fetcher{comments: comments}
	return &Loaders{
		replies: dataloadgen.NewLoader(f.fetchReplies, dataloadgen.WithWait(time.Millisecond)),
	}
}

func (l *Loaders) LoadReplies(ctx context.Context, parentID string, first int) (RepliesPage, error) {
	return l.replies.Load(ctx, repliesKey{ParentID: parentID, First: first})
}

func Middleware(comments repository.CommentRepository, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), ctxKey{}, NewLoaders(comments))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func For(ctx context.Context) *Loaders {
	loaders, _ := ctx.Value(ctxKey{}).(*Loaders)
	return loaders
}

type fetcher struct {
	comments repository.CommentRepository
}

func (f fetcher) fetchReplies(ctx context.Context, keys []repliesKey) ([]RepliesPage, []error) {
	results := make([]RepliesPage, len(keys))
	errs := make([]error, len(keys))

	byFirst := make(map[int][]int)
	for i, k := range keys {
		byFirst[k.First] = append(byFirst[k.First], i)
	}

	for first, indexes := range byFirst {
		parentIDs := make([]string, 0, len(indexes))
		for _, i := range indexes {
			parentIDs = append(parentIDs, keys[i].ParentID)
		}
		items, hasNext, err := f.comments.ListRepliesBatch(ctx, parentIDs, first)
		for _, i := range indexes {
			if err != nil {
				errs[i] = err
				continue
			}
			results[i] = RepliesPage{Items: items[keys[i].ParentID], HasNext: hasNext[keys[i].ParentID]}
		}
	}
	return results, errs
}

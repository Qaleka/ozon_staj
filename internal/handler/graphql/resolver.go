package graphql

import (
	"posts-service/internal/handler/graphql/graphmodel"
	"posts-service/internal/model"
	"posts-service/internal/pubsub"
	"posts-service/internal/usecase"
)

type Resolver struct {
	posts    *usecase.PostUsecase
	comments *usecase.CommentUsecase
	bus      *pubsub.CommentBus
}

func NewResolver(posts *usecase.PostUsecase, comments *usecase.CommentUsecase, bus *pubsub.CommentBus) *Resolver {
	return &Resolver{posts: posts, comments: comments, bus: bus}
}

func commentConnection(items []*model.Comment, hasNext bool) *graphmodel.CommentConnection {
	edges := make([]*graphmodel.CommentEdge, 0, len(items))
	for _, c := range items {
		edges = append(edges, &graphmodel.CommentEdge{
			Cursor: model.Cursor{CreatedAt: c.CreatedAt, ID: c.ID}.Encode(),
			Node:   c,
		})
	}
	info := &graphmodel.PageInfo{HasNextPage: hasNext}
	if len(edges) > 0 {
		info.EndCursor = &edges[len(edges)-1].Cursor
	}
	return &graphmodel.CommentConnection{Edges: edges, PageInfo: info}
}

func postConnection(items []*model.Post, hasNext bool) *graphmodel.PostConnection {
	edges := make([]*graphmodel.PostEdge, 0, len(items))
	for _, p := range items {
		edges = append(edges, &graphmodel.PostEdge{
			Cursor: model.Cursor{CreatedAt: p.CreatedAt, ID: p.ID}.Encode(),
			Node:   p,
		})
	}
	info := &graphmodel.PageInfo{HasNextPage: hasNext}
	if len(edges) > 0 {
		info.EndCursor = &edges[len(edges)-1].Cursor
	}
	return &graphmodel.PostConnection{Edges: edges, PageInfo: info}
}

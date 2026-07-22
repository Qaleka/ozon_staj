package model

import "time"

type Post struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Content          string    `json:"content"`
	Author           string    `json:"author"`
	CommentsDisabled bool      `json:"commentsDisabled"`
	CreatedAt        time.Time `json:"createdAt"`
}

package model

import "errors"

var (
	ErrPostNotFound = errors.New("post not found")

	ErrCommentNotFound = errors.New("comment not found")

	ErrCommentsDisabled = errors.New("comments are disabled for this post")

	ErrCommentTooLong = errors.New("comment text exceeds maximum length")

	ErrEmptyField = errors.New("required field is empty")

	ErrForbidden = errors.New("operation is allowed only for the post author")

	ErrParentFromAnotherPost = errors.New("parent comment belongs to another post")

	ErrInvalidCursor = errors.New("invalid pagination cursor")

	ErrInvalidLimit = errors.New("invalid page size: must be between 1 and 100")
)

package graphql

import (
	"context"
	"errors"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"posts-service/internal/model"
)

var errorCodes = map[error]string{
	model.ErrPostNotFound:          "NOT_FOUND",
	model.ErrCommentNotFound:       "NOT_FOUND",
	model.ErrCommentsDisabled:      "COMMENTS_DISABLED",
	model.ErrCommentTooLong:        "BAD_REQUEST",
	model.ErrEmptyField:            "BAD_REQUEST",
	model.ErrParentFromAnotherPost: "BAD_REQUEST",
	model.ErrInvalidCursor:         "BAD_REQUEST",
	model.ErrInvalidLimit:          "BAD_REQUEST",
	model.ErrForbidden:             "FORBIDDEN",
}

func ErrorPresenter(ctx context.Context, err error) *gqlerror.Error {
	gqlErr := graphql.DefaultErrorPresenter(ctx, err)
	if gqlErr.Extensions == nil {
		gqlErr.Extensions = map[string]any{}
	}

	for domainErr, code := range errorCodes {
		if errors.Is(err, domainErr) {
			gqlErr.Message = domainErr.Error()
			gqlErr.Extensions["code"] = code
			return gqlErr
		}
	}

	var validationErr *gqlerror.Error
	if errors.As(err, &validationErr) && validationErr.Extensions["code"] != nil {
		return gqlErr
	}

	gqlErr.Message = "internal server error"
	gqlErr.Extensions["code"] = "INTERNAL_ERROR"
	return gqlErr
}

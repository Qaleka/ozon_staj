package model

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultLimit = 10
	MaxLimit     = 100
)

type Cursor struct {
	CreatedAt time.Time
	ID        string
}

func (c Cursor) Encode() string {
	raw := fmt.Sprintf("%d|%s", c.CreatedAt.UTC().UnixNano(), c.ID)
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(s string) (Cursor, error) {
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 || parts[1] == "" {
		return Cursor{}, ErrInvalidCursor
	}
	nanos, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	return Cursor{CreatedAt: time.Unix(0, nanos).UTC(), ID: parts[1]}, nil
}

type Page struct {
	Limit int
	After *Cursor
}

func NewPage(first *int32, after *string) (Page, error) {
	limit := DefaultLimit
	if first != nil {
		limit = int(*first)
	}
	if limit < 1 || limit > MaxLimit {
		return Page{}, ErrInvalidLimit
	}
	p := Page{Limit: limit}
	if after != nil && *after != "" {
		c, err := DecodeCursor(*after)
		if err != nil {
			return Page{}, err
		}
		p.After = &c
	}
	return p, nil
}

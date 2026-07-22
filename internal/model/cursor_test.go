package model

import (
	"errors"
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	orig := Cursor{CreatedAt: time.Date(2026, 7, 22, 10, 30, 0, 123456789, time.UTC), ID: "some-id"}

	decoded, err := DecodeCursor(orig.Encode())
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !decoded.CreatedAt.Equal(orig.CreatedAt) || decoded.ID != orig.ID {
		t.Errorf("round trip mismatch: got %+v, want %+v", decoded, orig)
	}
}

func TestDecodeCursor_Invalid(t *testing.T) {
	for _, s := range []string{"", "not-base64!!!", "aGVsbG8=", "fHw="} {
		if _, err := DecodeCursor(s); !errors.Is(err, ErrInvalidCursor) {
			t.Errorf("DecodeCursor(%q): expected ErrInvalidCursor, got %v", s, err)
		}
	}
}

func TestNewPage(t *testing.T) {

	page, err := NewPage(nil, nil)
	if err != nil {
		t.Fatalf("NewPage: %v", err)
	}
	if page.Limit != DefaultLimit || page.After != nil {
		t.Errorf("defaults: got %+v", page)
	}

	for _, bad := range []int32{0, -1, MaxLimit + 1} {
		v := bad
		if _, err := NewPage(&v, nil); !errors.Is(err, ErrInvalidLimit) {
			t.Errorf("first=%d: expected ErrInvalidLimit, got %v", bad, err)
		}
	}

	c := Cursor{CreatedAt: time.Now().UTC(), ID: "id"}
	encoded := c.Encode()
	page, err = NewPage(nil, &encoded)
	if err != nil {
		t.Fatalf("NewPage with cursor: %v", err)
	}
	if page.After == nil || page.After.ID != "id" {
		t.Errorf("cursor not parsed: %+v", page)
	}

	junk := "junk"
	if _, err := NewPage(nil, &junk); !errors.Is(err, ErrInvalidCursor) {
		t.Errorf("expected ErrInvalidCursor, got %v", err)
	}
}

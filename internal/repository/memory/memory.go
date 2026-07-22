package memory

import (
	"sort"
	"time"

	"posts-service/internal/model"
)

type entry struct {
	createdAt time.Time
	id        string
}

func (e entry) less(other entry) bool {
	if !e.createdAt.Equal(other.createdAt) {
		return e.createdAt.Before(other.createdAt)
	}
	return e.id < other.id
}

func insertSorted(list []entry, e entry) []entry {
	pos := sort.Search(len(list), func(i int) bool { return e.less(list[i]) })
	list = append(list, entry{})
	copy(list[pos+1:], list[pos:])
	list[pos] = e
	return list
}

func pageAfter(list []entry, page model.Page) ([]entry, bool) {
	start := 0
	if page.After != nil {
		c := entry{createdAt: page.After.CreatedAt, id: page.After.ID}
		start = sort.Search(len(list), func(i int) bool { return c.less(list[i]) })
	}
	end := start + page.Limit
	if end >= len(list) {
		return list[start:], false
	}
	return list[start:end], true
}

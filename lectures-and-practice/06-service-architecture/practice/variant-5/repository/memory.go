package repository

import (
	"context"
	"sync"
	"time"

	"github.com/course-go-autumn-2026/practice6-variant5-comments/domain"
)

type commentRecord struct {
	ID        int64
	AuthorID  int64
	Text      string
	Status    string
	CreatedAt time.Time
}

func toRecord(id int64, c *domain.Comment) commentRecord {
	return commentRecord{id, c.AuthorID(), c.Text(), string(c.Status()), c.CreatedAt()}
}

func (r commentRecord) toDomain() *domain.Comment {
	return domain.RestoreComment(r.ID, r.AuthorID, r.Text, domain.Status(r.Status), r.CreatedAt)
}

type MemoryCommentRepository struct {
	mu     sync.Mutex
	rows   map[int64]commentRecord
	nextID int64
}

func NewMemoryCommentRepository() *MemoryCommentRepository {
	return &MemoryCommentRepository{rows: map[int64]commentRecord{}}
}

func (r *MemoryCommentRepository) Create(_ context.Context, c *domain.Comment) (*domain.Comment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	row := toRecord(r.nextID, c)
	r.rows[row.ID] = row
	return row.toDomain(), nil
}

func (r *MemoryCommentRepository) Get(_ context.Context, id int64) (*domain.Comment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return row.toDomain(), nil
}

func (r *MemoryCommentRepository) Update(_ context.Context, c *domain.Comment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rows[c.ID()]; !ok {
		return domain.ErrNotFound
	}
	r.rows[c.ID()] = toRecord(c.ID(), c)
	return nil
}

type authorRecord struct {
	ID         int64
	Reputation string
}

type MemoryAuthorRepository struct {
	rows map[int64]authorRecord
}

func NewMemoryAuthorRepository() *MemoryAuthorRepository {
	return &MemoryAuthorRepository{rows: map[int64]authorRecord{
		1: {ID: 1, Reputation: "new"},
		2: {ID: 2, Reputation: "trusted"},
	}}
}

func (r *MemoryAuthorRepository) Get(_ context.Context, id int64) (domain.Author, error) {
	row, ok := r.rows[id]
	if !ok {
		return domain.Author{}, domain.ErrNotFound
	}
	return domain.RestoreAuthor(row.ID, domain.Reputation(row.Reputation)), nil
}

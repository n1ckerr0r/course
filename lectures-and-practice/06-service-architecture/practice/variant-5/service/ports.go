package service

import (
	"context"

	"github.com/course-go-autumn-2026/practice6-variant5-comments/domain"
)

type CommentRepository interface {
	Create(ctx context.Context, c *domain.Comment) error
	Get(ctx context.Context, id int64) (*domain.Comment, error)
	Update(ctx context.Context, c *domain.Comment) error
}

type AuthorRepository interface {
	Get(ctx context.Context, id int64) (domain.Author, error)
}

type SpamFilter interface {
	IsSpam(ctx context.Context, text string) (bool, error)
}

type ToxicityChecker interface {
	IsToxic(ctx context.Context, text string) (bool, error)
}

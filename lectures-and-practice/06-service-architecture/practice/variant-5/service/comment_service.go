package service

import (
	"context"
	"errors"

	"github.com/course-go-autumn-2026/practice6-variant5-comments/domain"
)

type CommentService struct {
	comments CommentRepository
	authors  AuthorRepository
	spam     SpamFilter
	toxicity ToxicityChecker
}

func NewCommentService(c CommentRepository, a AuthorRepository, s SpamFilter, t ToxicityChecker) *CommentService {
	return &CommentService{comments: c, authors: a, spam: s, toxicity: t}
}

func (s *CommentService) Create(ctx context.Context, authorID int64, text string) (*domain.Comment, error) {
	comment, err := domain.NewComment(authorID, text)
	if err != nil {
		return nil, err
	}

	if isSpam, err := s.spam.IsSpam(ctx, text); err == nil && isSpam {
		return nil, domain.ErrLooksLikeSpam
	}

	author, err := s.authors.Get(ctx, authorID)
	if errors.Is(err, domain.ErrNotFound) {
		author = domain.NewAuthor(authorID)
	} else if err != nil {
		return nil, err
	}

	if err := comment.ApplyModeration(author, s.checkToxicity(ctx, text)); err != nil {
		return nil, err
	}

	return s.comments.Create(ctx, comment)
}

func (s *CommentService) Get(ctx context.Context, id int64) (*domain.Comment, error) {
	return s.comments.Get(ctx, id)
}

func (s *CommentService) EditText(ctx context.Context, id int64, text string) (*domain.Comment, error) {
	if err := domain.ValidateText(text); err != nil {
		return nil, err
	}
	comment, err := s.comments.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	needsRecheck, err := comment.ChangeText(text)
	if err != nil {
		return nil, err
	}
	if needsRecheck {
		if err := comment.ApplyEditCheck(s.checkToxicity(ctx, text)); err != nil {
			return nil, err
		}
	}

	if err := s.comments.Update(ctx, comment); err != nil {
		return nil, err
	}
	return comment, nil
}

func (s *CommentService) checkToxicity(ctx context.Context, text string) domain.Toxicity {
	toxic, err := s.toxicity.IsToxic(ctx, text)
	switch {
	case err != nil:
		return domain.ToxicityUnknown
	case toxic:
		return domain.ToxicityToxic
	default:
		return domain.ToxicityClean
	}
}

package domain

import (
	"strings"
	"time"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusPublished Status = "published"
	StatusHidden    Status = "hidden"
)

type Toxicity int

const (
	ToxicityClean Toxicity = iota
	ToxicityToxic
	ToxicityUnknown
)

type Comment struct {
	ID        int64
	AuthorID  int64
	Text      string
	Status    Status
	CreatedAt time.Time
}

func ValidateText(text string) error {
	if strings.TrimSpace(text) == "" {
		return ErrTextRequired
	}
	return nil
}

func NewComment(authorID int64, text string) (*Comment, error) {
	if err := ValidateText(text); err != nil {
		return nil, err
	}
	return &Comment{
		AuthorID:  authorID,
		Text:      text,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}, nil
}

func (c *Comment) ApplyModeration(author Author, t Toxicity) {
	switch t {
	case ToxicityToxic:
		c.Status = StatusHidden
	case ToxicityClean:
		if author.CanSkipManualModeration() {
			c.Status = StatusPublished
		}
	}
}

func (c *Comment) ChangeText(text string) (needsRecheck bool, err error) {
	if err := ValidateText(text); err != nil {
		return false, err
	}
	if c.Status == StatusHidden {
		return false, ErrHiddenComment
	}
	c.Text = text
	return c.Status == StatusPublished, nil
}

func (c *Comment) ApplyEditCheck(t Toxicity) {
	if c.Status == StatusPublished && t != ToxicityClean {
		c.Status = StatusPending
	}
}

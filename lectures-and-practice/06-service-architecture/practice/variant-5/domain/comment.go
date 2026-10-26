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
	id        int64
	authorID  int64
	text      string
	status    Status
	createdAt time.Time
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
		authorID:  authorID,
		text:      text,
		status:    StatusPending,
		createdAt: time.Now(),
	}, nil
}

func RestoreComment(id, authorID int64, text string, status Status, createdAt time.Time) *Comment {
	return &Comment{id: id, authorID: authorID, text: text, status: status, createdAt: createdAt}
}

func (c *Comment) ID() int64            { return c.id }
func (c *Comment) AuthorID() int64      { return c.authorID }
func (c *Comment) Text() string         { return c.text }
func (c *Comment) Status() Status       { return c.status }
func (c *Comment) CreatedAt() time.Time { return c.createdAt }

func (c *Comment) ApplyModeration(author Author, t Toxicity) error {
	if c.status != StatusPending {
		return ErrInvalidTransition
	}
	switch t {
	case ToxicityToxic:
		c.status = StatusHidden
	case ToxicityClean:
		if author.CanSkipManualModeration() {
			c.status = StatusPublished
		}
	}
	return nil
}

func (c *Comment) ChangeText(text string) (needsRecheck bool, err error) {
	if err := ValidateText(text); err != nil {
		return false, err
	}
	if c.status == StatusHidden {
		return false, ErrHiddenComment
	}
	c.text = text
	return c.status == StatusPublished, nil
}

func (c *Comment) ApplyEditCheck(t Toxicity) error {
	if c.status != StatusPublished {
		return ErrInvalidTransition
	}
	if t != ToxicityClean {
		c.status = StatusPending
	}
	return nil
}

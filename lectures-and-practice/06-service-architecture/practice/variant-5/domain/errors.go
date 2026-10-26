package domain

import "errors"

var (
	ErrTextRequired  = errors.New("text is required")
	ErrLooksLikeSpam = errors.New("comment looks like spam")
	ErrHiddenComment = errors.New("cannot edit a hidden comment")
	ErrNotFound      = errors.New("comment not found")
)

package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/course-go-autumn-2026/practice6-variant5-comments/domain"
)

func restored(s domain.Status) *domain.Comment {
	return domain.RestoreComment(1, 1, "t", s, time.Now())
}

func TestApplyModerationOnlyFromPending(t *testing.T) {
	trusted := domain.RestoreAuthor(2, domain.ReputationTrusted)
	for _, s := range []domain.Status{domain.StatusPublished, domain.StatusHidden} {
		if err := restored(s).ApplyModeration(trusted, domain.ToxicityClean); !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("%s: err = %v", s, err)
		}
	}
	c := restored(domain.StatusPending)
	if err := c.ApplyModeration(trusted, domain.ToxicityClean); err != nil || c.Status() != domain.StatusPublished {
		t.Fatalf("got %v, %v", c.Status(), err)
	}
}

func TestApplyEditCheckOnlyFromPublished(t *testing.T) {
	for _, s := range []domain.Status{domain.StatusPending, domain.StatusHidden} {
		if err := restored(s).ApplyEditCheck(domain.ToxicityToxic); !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("%s: err = %v", s, err)
		}
	}
	c := restored(domain.StatusPublished)
	if err := c.ApplyEditCheck(domain.ToxicityUnknown); err != nil || c.Status() != domain.StatusPending {
		t.Fatalf("got %v, %v", c.Status(), err)
	}
}

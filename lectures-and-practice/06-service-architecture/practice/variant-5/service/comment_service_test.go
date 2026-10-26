package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/course-go-autumn-2026/practice6-variant5-comments/domain"
	"github.com/course-go-autumn-2026/practice6-variant5-comments/repository"
	"github.com/course-go-autumn-2026/practice6-variant5-comments/service"
)

type fakeSpam struct {
	spam bool
	err  error
}

func (f fakeSpam) IsSpam(context.Context, string) (bool, error) { return f.spam, f.err }

type fakeToxicity struct {
	toxic bool
	err   error
}

func (f *fakeToxicity) IsToxic(context.Context, string) (bool, error) { return f.toxic, f.err }

var errDown = errors.New("down")

func newSvc(sp fakeSpam, tx *fakeToxicity) *service.CommentService {
	return service.NewCommentService(
		repository.NewMemoryCommentRepository(),
		repository.NewMemoryAuthorRepository(), sp, tx)
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name    string
		author  int64
		text    string
		spam    fakeSpam
		tox     fakeToxicity
		want    domain.Status
		wantErr error
	}{
		{"empty text", 2, "  ", fakeSpam{}, fakeToxicity{}, "", domain.ErrTextRequired},
		{"spam", 2, "x", fakeSpam{spam: true}, fakeToxicity{}, "", domain.ErrLooksLikeSpam},
		{"spam filter down is fail-open", 2, "x", fakeSpam{err: errDown}, fakeToxicity{}, domain.StatusPublished, nil},
		{"trusted clean", 2, "x", fakeSpam{}, fakeToxicity{}, domain.StatusPublished, nil},
		{"new clean stays pending", 1, "x", fakeSpam{}, fakeToxicity{}, domain.StatusPending, nil},
		{"unknown author is new", 99, "x", fakeSpam{}, fakeToxicity{}, domain.StatusPending, nil},
		{"toxic hidden", 2, "x", fakeSpam{}, fakeToxicity{toxic: true}, domain.StatusHidden, nil},
		{"toxicity down is fail-safe", 2, "x", fakeSpam{}, fakeToxicity{err: errDown}, domain.StatusPending, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tox := tt.tox
			c, err := newSvc(tt.spam, &tox).Create(context.Background(), tt.author, tt.text)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err == nil && c.Status() != tt.want {
				t.Fatalf("status = %s, want %s", c.Status(), tt.want)
			}
		})
	}
}

func TestEditText(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		author  int64
		create  fakeToxicity
		edit    fakeToxicity
		text    string
		want    domain.Status
		wantErr error
	}{
		{"published stays", 2, fakeToxicity{}, fakeToxicity{}, "new", domain.StatusPublished, nil},
		{"published toxic -> pending", 2, fakeToxicity{}, fakeToxicity{toxic: true}, "new", domain.StatusPending, nil},
		{"published down -> pending", 2, fakeToxicity{}, fakeToxicity{err: errDown}, "new", domain.StatusPending, nil},
		{"pending unchanged, no recheck", 1, fakeToxicity{}, fakeToxicity{toxic: true}, "new", domain.StatusPending, nil},
		{"hidden forbidden", 2, fakeToxicity{toxic: true}, fakeToxicity{}, "new", "", domain.ErrHiddenComment},
		{"empty text", 2, fakeToxicity{}, fakeToxicity{}, "", "", domain.ErrTextRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tox := tt.create
			svc := newSvc(fakeSpam{}, &tox)
			c, err := svc.Create(ctx, tt.author, "old")
			if err != nil {
				t.Fatal(err)
			}
			tox.toxic, tox.err = tt.edit.toxic, tt.edit.err
			got, err := svc.EditText(ctx, c.ID(), tt.text)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err == nil && (got.Status() != tt.want || got.Text() != tt.text) {
				t.Fatalf("got %+v", got)
			}
		})
	}
}

func TestNotFound(t *testing.T) {
	svc := newSvc(fakeSpam{}, &fakeToxicity{})
	if _, err := svc.Get(context.Background(), 1); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("get: %v", err)
	}
	if _, err := svc.EditText(context.Background(), 1, "x"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("edit: %v", err)
	}
}

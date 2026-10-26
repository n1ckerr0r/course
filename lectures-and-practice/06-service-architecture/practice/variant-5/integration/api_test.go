//go:build integration

package integration

import (
	"net/http"
	"testing"
)

const (
	newAuthor     = 1
	trustedAuthor = 2
	unknownAuthor = 99
)

func TestCreate_Status(t *testing.T) {
	cases := []struct {
		name     string
		authorID int64
		toxicity mode
		want     string
	}{
		{"trusted, not toxic", trustedAuthor, modeOK, "published"},
		{"new, not toxic", newAuthor, modeOK, "pending"},
		{"unknown author is new", unknownAuthor, modeOK, "pending"},
		{"trusted, toxic", trustedAuthor, modeReject, "hidden"},
		{"new, toxic", newAuthor, modeReject, "hidden"},
		{"toxicity 500 counts as toxic", trustedAuthor, modeError, "hidden"},
		{"toxicity unavailable", trustedAuthor, modeDown, "pending"},
	}
	forEachTarget(t, func(t *testing.T, bin string) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				s := start(t, bin)
				s.ext.set(modeOK, c.toxicity)

				expect(t, s.create(c.authorID, "hello"), http.StatusCreated, `{"id":1,"status":"`+c.want+`"}`+"\n")
				expectComment(t, s.get("1"), comment{ID: 1, AuthorID: c.authorID, Text: "hello", Status: c.want})
			})
		}
	})
}

func TestCreate_ChecksSpamThenToxicity(t *testing.T) {
	forEachTarget(t, func(t *testing.T, bin string) {
		s := start(t, bin)

		s.mustCreate(trustedAuthor, "hello", "published")

		expectCalls(t, s.ext.takeCalls(),
			call{host: spamHost, text: "hello"},
			call{host: toxicityHost, text: "hello"},
		)
	})
}

func TestCreate_Spam(t *testing.T) {
	forEachTarget(t, func(t *testing.T, bin string) {
		s := start(t, bin)
		s.ext.set(modeReject, modeOK)

		expect(t, s.create(trustedAuthor, "buy now"), http.StatusBadRequest, "comment looks like spam\n")
		expectCalls(t, s.ext.takeCalls(), call{host: spamHost, text: "buy now"})
		expect(t, s.get("1"), http.StatusNotFound, "")

		s.ext.set(modeOK, modeOK)
		expect(t, s.create(trustedAuthor, "hello"), http.StatusCreated, `{"id":1,"status":"published"}`+"\n")
	})
}

func TestCreate_SpamFilterFailOpen(t *testing.T) {
	cases := []struct {
		name string
		spam mode
	}{
		{"spam filter unavailable", modeDown},
		{"spam filter 500", modeError},
	}
	forEachTarget(t, func(t *testing.T, bin string) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				s := start(t, bin)
				s.ext.set(c.spam, modeOK)

				expect(t, s.create(trustedAuthor, "hello"), http.StatusCreated, `{"id":1,"status":"published"}`+"\n")
			})
		}
	})
}

func TestCreate_BadRequest(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantBody string
	}{
		{"empty text", `{"author_id":1,"text":""}`, "text is required\n"},
		{"blank text", `{"author_id":1,"text":"   "}`, "text is required\n"},
		{"no text", `{"author_id":1}`, "text is required\n"},
		{"broken json", `{bad json`, ""},
	}
	forEachTarget(t, func(t *testing.T, bin string) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				s := start(t, bin)

				expect(t, s.do(http.MethodPost, "/comments", c.body), http.StatusBadRequest, c.wantBody)
				expectCalls(t, s.ext.takeCalls())
				expect(t, s.get("1"), http.StatusNotFound, "")
			})
		}
	})
}

func TestGet(t *testing.T) {
	forEachTarget(t, func(t *testing.T, bin string) {
		s := start(t, bin)
		s.mustCreate(trustedAuthor, "hello", "published")

		expectComment(t, s.get("1"), comment{ID: 1, AuthorID: trustedAuthor, Text: "hello", Status: "published"})
		expect(t, s.get("999"), http.StatusNotFound, "")
		expect(t, s.get("abc"), http.StatusBadRequest, "")
	})
}

func TestUpdateText_PublishedIsRechecked(t *testing.T) {
	cases := []struct {
		name     string
		toxicity mode
		want     string
	}{
		{"still not toxic", modeOK, "published"},
		{"became toxic", modeReject, "pending"},
		{"toxicity 500", modeError, "pending"},
		{"toxicity unavailable", modeDown, "pending"},
	}
	forEachTarget(t, func(t *testing.T, bin string) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				s := start(t, bin)
				s.mustCreate(trustedAuthor, "hello", "published")
				s.ext.takeCalls()

				s.ext.set(modeOK, c.toxicity)
				want := comment{ID: 1, AuthorID: trustedAuthor, Text: "edited", Status: c.want}
				expectComment(t, s.patchText("1", "edited"), want)
				expectCalls(t, s.ext.takeCalls(), call{host: toxicityHost, text: "edited"})
				expectComment(t, s.get("1"), want)
			})
		}
	})
}

func TestUpdateText_PendingIsNotRechecked(t *testing.T) {
	forEachTarget(t, func(t *testing.T, bin string) {
		s := start(t, bin)
		s.mustCreate(newAuthor, "hello", "pending")
		s.ext.takeCalls()

		s.ext.set(modeOK, modeReject)
		want := comment{ID: 1, AuthorID: newAuthor, Text: "edited", Status: "pending"}
		expectComment(t, s.patchText("1", "edited"), want)
		expectCalls(t, s.ext.takeCalls())
		expectComment(t, s.get("1"), want)
	})
}

func TestUpdateText_HiddenIsConflict(t *testing.T) {
	forEachTarget(t, func(t *testing.T, bin string) {
		s := start(t, bin)
		s.ext.set(modeOK, modeReject)
		s.mustCreate(trustedAuthor, "hello", "hidden")
		s.ext.takeCalls()

		expect(t, s.patchText("1", "edited"), http.StatusConflict, "cannot edit a hidden comment\n")
		expectCalls(t, s.ext.takeCalls())
		expectComment(t, s.get("1"), comment{ID: 1, AuthorID: trustedAuthor, Text: "hello", Status: "hidden"})
	})
}

func TestUpdateText_BadRequest(t *testing.T) {
	cases := []struct {
		name       string
		path       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{"empty text", "/comments/1/text", `{"text":""}`, http.StatusBadRequest, "text is required\n"},
		{"blank text", "/comments/1/text", `{"text":"   "}`, http.StatusBadRequest, "text is required\n"},
		{"broken json", "/comments/1/text", `{bad`, http.StatusBadRequest, ""},
		{"id is not a number", "/comments/abc/text", `{"text":"x"}`, http.StatusBadRequest, ""},
		{"not found", "/comments/999/text", `{"text":"x"}`, http.StatusNotFound, ""},
		{"empty text and not found", "/comments/999/text", `{"text":""}`, http.StatusBadRequest, "text is required\n"},
	}
	forEachTarget(t, func(t *testing.T, bin string) {
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				s := start(t, bin)
				s.mustCreate(trustedAuthor, "hello", "published")
				s.ext.takeCalls()

				expect(t, s.do(http.MethodPatch, c.path, c.body), c.wantStatus, c.wantBody)
				expectCalls(t, s.ext.takeCalls())
				expectComment(t, s.get("1"), comment{ID: 1, AuthorID: trustedAuthor, Text: "hello", Status: "published"})
			})
		}
	})
}

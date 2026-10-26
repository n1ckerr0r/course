//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	baseURL      = "http://localhost:8080"
	spamHost     = "spam-filter.internal"
	toxicityHost = "toxicity-check.internal"
)

type target struct {
	name   string
	source string
	bin    string
}

var targets = []target{
	{name: "old", source: "../main_old.go"},
	{name: "new", source: ".."},
}

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "comments-integration")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for i := range targets {
		targets[i].bin = filepath.Join(dir, targets[i].name)
		out, err := exec.Command("go", "build", "-o", targets[i].bin, targets[i].source).CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "build %s: %v\n%s", targets[i].name, err, out)
			os.RemoveAll(dir)
			os.Exit(1)
		}
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func forEachTarget(t *testing.T, fn func(t *testing.T, bin string)) {
	for _, tg := range targets {
		t.Run(tg.name, func(t *testing.T) { fn(t, tg.bin) })
	}
}

type mode int

const (
	modeOK mode = iota
	modeReject
	modeError
	modeDown
)

type call struct {
	host string
	text string
}

type fakeExternal struct {
	mu       sync.Mutex
	spam     mode
	toxicity mode
	calls    []call
}

func (f *fakeExternal) set(spam, toxicity mode) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.spam, f.toxicity = spam, toxicity
}

func (f *fakeExternal) takeCalls() []call {
	f.mu.Lock()
	defer f.mu.Unlock()
	calls := f.calls
	f.calls = nil
	return calls
}

func (f *fakeExternal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Text string `json:"text"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	f.mu.Lock()
	var m mode
	switch r.Host {
	case spamHost:
		m = f.spam
	case toxicityHost:
		m = f.toxicity
	default:
		f.mu.Unlock()
		http.Error(w, "unexpected host "+r.Host, http.StatusBadGateway)
		return
	}
	f.calls = append(f.calls, call{host: r.Host, text: body.Text})
	f.mu.Unlock()

	switch m {
	case modeOK:
		w.WriteHeader(http.StatusOK)
	case modeReject:
		w.WriteHeader(http.StatusForbidden)
	case modeError:
		w.WriteHeader(http.StatusInternalServerError)
	case modeDown:
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			conn.Close()
		}
	}
}

var client = &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{}}

type server struct {
	t   *testing.T
	ext *fakeExternal
}

func start(t *testing.T, bin string) *server {
	t.Helper()

	if conn, err := net.Dial("tcp", "localhost:8080"); err == nil {
		conn.Close()
		t.Fatal("port 8080 is busy: stop the running service first")
	}

	ext := &fakeExternal{}
	proxy := httptest.NewServer(ext)
	t.Cleanup(proxy.Close)

	var logs bytes.Buffer
	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(), "HTTP_PROXY="+proxy.URL, "NO_PROXY=", "no_proxy=")
	cmd.Stdout, cmd.Stderr = &logs, &logs
	if err := cmd.Start(); err != nil {
		t.Fatalf("start service: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		if t.Failed() {
			t.Logf("service log:\n%s", logs.String())
		}
	})

	for range 50 {
		resp, err := client.Get(baseURL + "/comments/0")
		if err == nil {
			resp.Body.Close()
			return &server{t: t, ext: ext}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("service did not start on :8080")
	return nil
}

type response struct {
	status int
	body   string
}

func (s *server) do(method, path, body string) response {
	s.t.Helper()

	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, baseURL+path, r)
	if err != nil {
		s.t.Fatalf("new request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		s.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		s.t.Fatalf("read body: %v", err)
	}
	return response{status: resp.StatusCode, body: string(b)}
}

func (s *server) create(authorID int64, text string) response {
	s.t.Helper()
	return s.do(http.MethodPost, "/comments", toJSON(s.t, map[string]any{"author_id": authorID, "text": text}))
}

func (s *server) get(id string) response {
	s.t.Helper()
	return s.do(http.MethodGet, "/comments/"+id, "")
}

func (s *server) patchText(id, text string) response {
	s.t.Helper()
	return s.do(http.MethodPatch, "/comments/"+id+"/text", toJSON(s.t, map[string]any{"text": text}))
}

func (s *server) mustCreate(authorID int64, text, wantStatus string) {
	s.t.Helper()
	resp := s.create(authorID, text)
	if resp.status != http.StatusCreated || !strings.Contains(resp.body, `"status":"`+wantStatus+`"`) {
		s.t.Fatalf("create: got %d %q, want 201 with status %q", resp.status, resp.body, wantStatus)
	}
}

func toJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func expect(t *testing.T, got response, status int, body string) {
	t.Helper()
	if got.status != status || got.body != body {
		t.Fatalf("got %d %q, want %d %q", got.status, got.body, status, body)
	}
}

type comment struct {
	ID        int64
	AuthorID  int64
	Text      string
	Status    string
	CreatedAt time.Time
}

func expectComment(t *testing.T, got response, want comment) {
	t.Helper()
	if got.status != http.StatusOK {
		t.Fatalf("got %d %q, want 200", got.status, got.body)
	}

	var keys map[string]json.RawMessage
	if err := json.Unmarshal([]byte(got.body), &keys); err != nil {
		t.Fatalf("decode %q: %v", got.body, err)
	}
	for _, k := range []string{"ID", "AuthorID", "Text", "Status", "CreatedAt"} {
		if _, ok := keys[k]; !ok {
			t.Fatalf("body %q has no key %q", got.body, k)
		}
	}
	if len(keys) != 5 {
		t.Fatalf("body %q has extra keys", got.body)
	}

	var c comment
	if err := json.Unmarshal([]byte(got.body), &c); err != nil {
		t.Fatalf("decode %q: %v", got.body, err)
	}
	if time.Since(c.CreatedAt) > time.Minute || c.CreatedAt.After(time.Now()) {
		t.Fatalf("CreatedAt = %v, want about now", c.CreatedAt)
	}
	c.CreatedAt = time.Time{}
	if c != want {
		t.Fatalf("got %+v, want %+v", c, want)
	}
}

func expectCalls(t *testing.T, got []call, want ...call) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("external calls = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("external calls = %v, want %v", got, want)
		}
	}
}

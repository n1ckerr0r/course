package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/course-go-autumn-2026/practice6-variant5-comments/domain"
)

type CommentService interface {
	Create(ctx context.Context, authorID int64, text string) (*domain.Comment, error)
	Get(ctx context.Context, id int64) (*domain.Comment, error)
	EditText(ctx context.Context, id int64, text string) (*domain.Comment, error)
}

type CommentHandler struct{ svc CommentService }

func NewCommentHandler(svc CommentService) *CommentHandler { return &CommentHandler{svc: svc} }

func (h *CommentHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /comments", h.create)
	mux.HandleFunc("GET /comments/{id}", h.get)
	mux.HandleFunc("PATCH /comments/{id}/text", h.updateText)
}

type createRequest struct {
	AuthorID int64  `json:"author_id"`
	Text     string `json:"text"`
}

type updateTextRequest struct {
	Text string `json:"text"`
}

type commentResponse struct {
	ID        int64     `json:"ID"`
	AuthorID  int64     `json:"AuthorID"`
	Text      string    `json:"Text"`
	Status    string    `json:"Status"`
	CreatedAt time.Time `json:"CreatedAt"`
}

func toResponse(c *domain.Comment) commentResponse {
	return commentResponse{c.ID, c.AuthorID, c.Text, string(c.Status), c.CreatedAt}
}

func (h *CommentHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	c, err := h.svc.Create(r.Context(), req.AuthorID, req.Text)
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": c.ID, "status": c.Status})
}

func (h *CommentHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	c, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toResponse(c))
}

func (h *CommentHandler) updateText(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var req updateTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	c, err := h.svc.EditText(r.Context(), id, req.Text)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toResponse(c))
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrTextRequired), errors.Is(err, domain.ErrLooksLikeSpam):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, domain.ErrHiddenComment):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, domain.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"posts/internal/core/comment"
	"posts/pkg/utils"
)

// CommentHandler wires comment repository
type CommentHandler struct {
	repo *comment.Repository
}

func NewCommentHandler(r *comment.Repository) *CommentHandler {
	return &CommentHandler{repo: r}
}

// POST /comments
func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	var c comment.Comment
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	if c.UserId == "" || c.RelatedId == "" || c.Message == "" {
		utils.Error(w, http.StatusBadRequest, "userId, relatedId and message required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	id, err := h.repo.Create(ctx, &c)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.JSON(w, http.StatusCreated, map[string]string{"id": id})
}

// GET /comments?relatedId=xxx&limit=10
func (h *CommentHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	related := strings.TrimSpace(q.Get("relatedId"))
	limit := int64(0)
	skip := int64(0)

	if l := q.Get("limit"); l != "" {
		if v, err := strconv.ParseInt(l, 10, 64); err == nil {
			limit = v
		}
	}
	if s := q.Get("skip"); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			skip = v
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	comments, total, err := h.repo.ListAllOrByRelated(ctx, related, limit, skip)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"comments": comments,
		"total":    total,
	})
}

// GET /comments/{id}
func (h *CommentHandler) GetComment(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/comments/")
	if id == "" {
		utils.Error(w, http.StatusBadRequest, "missing id")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	cmt, err := h.repo.GetByID(ctx, id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if cmt == nil {
		utils.Error(w, http.StatusNotFound, "comment not found")
		return
	}
	utils.JSON(w, http.StatusOK, cmt)
}

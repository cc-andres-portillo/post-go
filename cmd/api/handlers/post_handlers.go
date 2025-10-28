package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"posts/internal/core/comment"
	"posts/internal/core/post"
	"posts/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PostHandler wires repository
type PostHandler struct {
	repo    *post.Repository
	cmtRepo *comment.Repository
}

func NewPostHandler(r *post.Repository, c *comment.Repository) *PostHandler {
	return &PostHandler{repo: r, cmtRepo: c}
}

// GET /posts
// supports filters: title, description, mediaCategory, postGroupId, limit, skip, sortFixed=true
func (h *PostHandler) ListPosts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	q := r.URL.Query()
	filter := bson.M{}

	// Filtros básicos
	if t := strings.TrimSpace(q.Get("title")); t != "" {
		filter["title"] = primitive.Regex{Pattern: t, Options: "i"}
	}
	if d := strings.TrimSpace(q.Get("description")); d != "" {
		filter["description"] = primitive.Regex{Pattern: d, Options: "i"}
	}
	if mc := strings.TrimSpace(q.Get("mediaCategory")); mc != "" {
		filter["mediaCategory"] = mc
	}
	if pg := strings.TrimSpace(q.Get("postGroupId")); pg != "" {
		filter["postGroupId"] = pg
	}

	// Paginación
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

	// Ordenamiento
	sort := bson.D{{Key: "createdAt", Value: -1}}
	if q.Get("sortFixed") == "true" {
		sort = bson.D{{Key: "fixed", Value: -1}, {Key: "createdAt", Value: -1}}
	}

	// Llamada al nuevo método que hace el lookup con comentarios
	posts, total, err := h.repo.ListWithComments(ctx, post.ListOptions{
		Limit:  limit,
		Skip:   skip,
		Sort:   sort,
		Filter: filter,
	})
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Añadir totalComments calculado
	for i := range posts {
		cnt, _ := h.cmtRepo.CountByRelated(ctx, posts[i].ID)
		posts[i].TotalComments = int64(cnt)
	}

	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"posts": posts,
		"total": total,
	})
}

// GET /posts/{id}
func (h *PostHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/posts/")
	if id == "" {
		utils.Error(w, http.StatusBadRequest, "missing id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Buscar el post
	post, err := h.repo.GetByID(ctx, id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if post == nil {
		utils.Error(w, http.StatusNotFound, "post not found")
		return
	}

	// Obtener los comentarios relacionados
	comments, totalComments, err := h.cmtRepo.ListByRelated(ctx, id, 0)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Agregar la información dentro del post
	postMap := utils.StructToMap(post)
	postMap["totalComments"] = totalComments
	postMap["comments"] = comments

	utils.JSON(w, http.StatusOK, postMap)
}

// POST /posts
func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	var p post.Post
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	if p.ImgHeight == "" {
		p.ImgHeight = "600px"
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	id, err := h.repo.Create(ctx, &p)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.JSON(w, http.StatusCreated, map[string]string{"id": id})
}

// PUT /posts/{id}
func (h *PostHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/posts/")
	if id == "" {
		utils.Error(w, http.StatusBadRequest, "missing id")
		return
	}
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid body")
		return
	}

	delete(body, "isRemove")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := h.repo.Update(ctx, id, body); err != nil {
		utils.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.JSON(w, http.StatusOK, map[string]string{"id": id})
}

// DELETE /posts/{id}
func (h *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/posts/")
	if id == "" {
		utils.Error(w, http.StatusBadRequest, "missing id")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := h.repo.SoftDelete(ctx, id); err != nil {
		utils.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.JSON(w, http.StatusOK, map[string]string{"result": "ok"})
}

// POST /posts/removeSome  { "postIds": ["id1","id2"] }
func (h *PostHandler) RemoveSome(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PostIds []string `json:"postIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(body.PostIds) == 0 {
		utils.Error(w, http.StatusBadRequest, "postIds required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err := h.repo.RemoveSome(ctx, body.PostIds); err != nil {
		utils.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.JSON(w, http.StatusOK, map[string]string{"result": "ok"})
}

// POST /posts/like { "postId":"...", "userId":"..." }
func (h *PostHandler) ToggleLike(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PostId string `json:"postId"`
		UserId string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.PostId == "" || body.UserId == "" {
		utils.Error(w, http.StatusBadRequest, "postId and userId required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	liked, total, err := h.repo.ToggleLike(ctx, body.PostId, body.UserId)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.JSON(w, http.StatusOK, map[string]interface{}{"liked": liked, "totalLikes": total})
}

// GET /posts/{id}/likes -> we return the list of userIds and total
func (h *PostHandler) GetLikes(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/posts/")
	if !strings.HasSuffix(r.URL.Path, "/likes") {
		utils.Error(w, http.StatusBadRequest, "invalid path")
		return
	}
	postId := strings.TrimSuffix(id, "/likes")
	postId = strings.TrimSuffix(postId, "/")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	p, err := h.repo.GetByID(ctx, postId)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if p == nil {
		utils.Error(w, http.StatusNotFound, "post not found")
		return
	}
	utils.JSON(w, http.StatusOK, map[string]interface{}{
		"likes": p.Like,
		"total": len(p.Like),
	})
}

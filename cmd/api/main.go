package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"posts/cmd/api/handlers"
	"posts/internal/core/comment"
	"posts/internal/core/post"
	"posts/pkg/db"

	"github.com/go-chi/chi/v5"
)

func main() {
	// Mongo URI from env or fallback
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://root:12345abc@localhost:27017/futurapps?directConnection=true&authSource=admin&authMechanism=SCRAM-SHA-1"
	}

	ctx := context.Background()
	client, err := db.NewMongoClient(ctx, mongoURI)
	if err != nil {
		log.Fatalf("cannot connect to mongo: %v", err)
	}
	defer func() {
		_ = client.Disconnect(ctx)
	}()

	mdb := client.Database("futurapps")

	// repositories
	postRepo := post.NewRepository(mdb)
	commentRepo := comment.NewRepository(mdb)

	// handlers
	postHandler := handlers.NewPostHandler(postRepo, commentRepo)
	commentHandler := handlers.NewCommentHandler(commentRepo)

	r := chi.NewRouter()

	// posts collection endpoints
	r.Get("/posts", postHandler.ListPosts)
	r.Post("/posts", postHandler.CreatePost)
	r.Get("/posts/{id}", postHandler.GetPost) // chi provides id but we used path trimming above; keep for clarity
	r.Put("/posts/{id}", postHandler.UpdatePost)
	r.Delete("/posts/{id}", postHandler.DeletePost)

	r.Post("/posts/removeSome", postHandler.RemoveSome)
	r.Post("/posts/like", postHandler.ToggleLike)
	r.Get("/posts/{id}/likes", postHandler.GetLikes)
	// posts/{id}/comments -> we want GET /posts/{id}/comments to list comments
	r.Get("/posts/{id}/comments", func(w http.ResponseWriter, r *http.Request) {
		// rewrite query for comment handler
		id := chi.URLParam(r, "id")
		q := r.URL.Query()
		q.Set("relatedId", id)
		r.URL.RawQuery = q.Encode()
		commentHandler.ListComments(w, r)
	})

	// comments
	r.Get("/comments", commentHandler.ListComments)
	r.Post("/comments", commentHandler.CreateComment)
	r.Get("/comments/{id}", commentHandler.GetComment)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		log.Printf("listening on %s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server err: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("shutting down server...")
	ctxSh, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctxSh); err != nil {
		log.Fatalf("failed to shutdown: %v", err)
	}
	log.Println("server stopped")
}

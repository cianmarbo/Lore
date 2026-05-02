package server

import (
	"io/fs"
	"net/http"
	"strings"

	"lore/internal/docgen"
	"lore/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func New(st *store.Store, staticFS fs.FS, pipeline *docgen.Pipeline) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors)

	h := &handlers{store: st, pipeline: pipeline}

	r.Route("/api", func(r chi.Router) {
		r.Get("/conversations", h.listConversations)
		r.Post("/conversations/upload", h.uploadConversation)
		r.Get("/conversations/{id}", h.getConversation)
		r.Delete("/conversations/{id}", h.deleteConversation)
		r.Get("/conversations/{id}/messages", h.getMessages)
		r.Get("/search", h.searchTopics)

		r.Get("/documents", h.listDocuments)
		r.Get("/documents/{id}", h.getDocument)
		r.Delete("/documents/{id}", h.deleteDocument)
		r.Post("/documents/search", h.searchDocuments)
		r.Post("/documents/{id}/regenerate", h.regenerateDocument)
	})

	// Serve Vue static files with SPA fallback
	if staticFS != nil {
		fileServer := http.FileServer(http.FS(staticFS))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			// Try serving the file directly
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}

			// Check if file exists
			if f, err := staticFS.Open(path); err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}

			// SPA fallback: serve index.html
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
		})
	}

	return r
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

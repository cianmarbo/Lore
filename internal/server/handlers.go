package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"lore/internal/docgen"
	"lore/internal/parser"
	"lore/internal/store"

	"github.com/go-chi/chi/v5"
)

type handlers struct {
	store    *store.Store
	pipeline *docgen.Pipeline
}

func (h *handlers) listConversations(w http.ResponseWriter, r *http.Request) {
	convos, err := h.store.ListConversations()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, convos)
}

func (h *handlers) getConversation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}

	convo, err := h.store.GetConversation(id)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	jsonOK(w, convo)
}

func (h *handlers) getMessages(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}

	var topicID *int64
	if tid := r.URL.Query().Get("topic_id"); tid != "" {
		parsed, err := strconv.ParseInt(tid, 10, 64)
		if err != nil {
			jsonError(w, "invalid topic_id", http.StatusBadRequest)
			return
		}
		topicID = &parsed
	}

	msgs, err := h.store.GetMessages(id, topicID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, msgs)
}

type uploadRequest struct {
	SessionID string `json:"session_id"`
	Path      string `json:"path"`
}

func (h *handlers) uploadConversation(w http.ResponseWriter, r *http.Request) {
	var req uploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	path := req.Path
	if path == "" && req.SessionID != "" {
		path = findSessionFile(req.SessionID)
		if path == "" {
			jsonError(w, "could not find JSONL file for session "+req.SessionID, http.StatusNotFound)
			return
		}
	}
	if path == "" {
		jsonError(w, "provide session_id or path", http.StatusBadRequest)
		return
	}

	parsed, err := parser.ParseConversation(path)
	if err != nil {
		jsonError(w, "parse error: "+err.Error(), http.StatusBadRequest)
		return
	}

	convo, err := h.store.SaveConversation(parsed, path)
	if err != nil {
		jsonError(w, "store error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Trigger doc generation (non-fatal — conversation is always saved)
	if h.pipeline != nil {
		if err := h.pipeline.ProcessConversation(r.Context(), convo); err != nil {
			log.Printf("docgen: %v", err)
		}
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, convo)
}

func (h *handlers) deleteConversation(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteConversation(id); err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonOK(w, map[string]bool{"deleted": true})
}

func (h *handlers) searchTopics(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		jsonError(w, "query parameter q is required", http.StatusBadRequest)
		return
	}

	topics, err := h.store.SearchTopics(q)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, topics)
}

// Document handlers

func (h *handlers) listDocuments(w http.ResponseWriter, r *http.Request) {
	docs, err := h.store.ListDocuments()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, docs)
}

func (h *handlers) getDocument(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}

	doc, err := h.store.GetDocument(id)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	jsonOK(w, doc)
}

func (h *handlers) deleteDocument(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteDocument(id); err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonOK(w, map[string]bool{"deleted": true})
}

type searchDocumentsRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

func (h *handlers) searchDocuments(w http.ResponseWriter, r *http.Request) {
	var req searchDocumentsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Query == "" {
		jsonError(w, "query is required", http.StatusBadRequest)
		return
	}
	if req.Limit <= 0 {
		req.Limit = 5
	}

	docs, err := h.store.SearchDocuments(req.Query, req.Limit)
	if err != nil {
		jsonError(w, "search error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, docs)
}

func (h *handlers) regenerateDocument(w http.ResponseWriter, r *http.Request) {
	if h.pipeline == nil {
		jsonError(w, "document generation not available", http.StatusServiceUnavailable)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}

	doc, err := h.store.GetDocument(id)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	if err := h.pipeline.RegenerateDocument(r.Context(), doc); err != nil {
		jsonError(w, "regenerate error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated document
	doc, err = h.store.GetDocument(id)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, doc)
}

// helpers

func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}


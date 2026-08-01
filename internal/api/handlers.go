package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"calebsons_inc/calebsons_go_cloud_native_distributed_task_queue/internal/demo"
	"calebsons_inc/calebsons_go_cloud_native_distributed_task_queue/internal/queue"
	"calebsons_inc/calebsons_go_cloud_native_distributed_task_queue/internal/task"
)

type Server struct {
	Queue *queue.Queue
}

func (s *Server) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /api/demos", s.handleDemos)
	mux.HandleFunc("GET /api/demos/{kind}", s.handleDemo)
	mux.HandleFunc("GET /api/stats", s.handleStats)
	mux.HandleFunc("GET /api/tasks", s.handleListTasks)
	mux.HandleFunc("POST /api/tasks", s.handleEnqueue)
	mux.HandleFunc("GET /api/tasks/{id}", s.handleGetTask)
	mux.HandleFunc("POST /api/tasks/{id}/cancel", s.handleCancel)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDemos(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"demos": demo.Catalog()})
}

func (s *Server) handleDemo(w http.ResponseWriter, r *http.Request) {
	info, ok := demo.Get(r.PathValue("kind"))
	if !ok {
		writeErr(w, http.StatusNotFound, "unknown demo")
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	if kind != "" && !demo.ValidKind(kind) {
		writeErr(w, http.StatusBadRequest, "invalid kind")
		return
	}
	writeJSON(w, http.StatusOK, s.Queue.Stats(kind))
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	opts := queue.ListOptions{
		Page:  queryInt(r, "page", 1),
		Limit: queryInt(r, "limit", 20),
	}
	if raw := r.URL.Query().Get("status"); raw != "" {
		st, ok := task.ParseStatus(raw)
		if !ok {
			writeErr(w, http.StatusBadRequest, "invalid status filter")
			return
		}
		opts.Status = st
	}
	if raw := r.URL.Query().Get("kind"); raw != "" {
		if !demo.ValidKind(raw) {
			writeErr(w, http.StatusBadRequest, "invalid kind filter")
			return
		}
		opts.Kind = demo.NormalizeKind(raw)
	}
	writeJSON(w, http.StatusOK, s.Queue.List(opts))
}

func (s *Server) handleEnqueue(w http.ResponseWriter, r *http.Request) {
	var req task.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	t, err := s.Queue.Enqueue(req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, ok := s.Queue.Get(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := s.Queue.Cancel(id)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeErr(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func queryInt(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// Package httpapi exposes the REST endpoints and the WebSocket upgrade.
package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"quizlive/internal/quiz"
	"quizlive/internal/session"
	"quizlive/internal/store"
)

const MaxUpload = 1 << 20 // 1 MB

type Config struct {
	PublicBaseURL  string // used to build links, e.g. http://localhost:3000
	AllowedOrigins []string
}

type Server struct {
	cfg   Config
	store store.Store
}

func New(cfg Config, st store.Store) http.Handler {
	s := &Server{cfg, st}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("POST /sessions/preview", s.preview)
	mux.HandleFunc("POST /sessions", s.create)
	mux.HandleFunc("GET /sessions/{id}", s.exists)
	mux.HandleFunc("GET /ws/{id}", s.ws)
	return s.cors(mux)
}

func (s *Server) originAllowed(origin string) bool {
	if origin == "" {
		return true
	}
	for _, o := range s.cfg.AllowedOrigins {
		if o == "*" || strings.EqualFold(o, origin) {
			return true
		}
	}
	return false
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" && s.originAllowed(o) {
			w.Header().Set("Access-Control-Allow-Origin", o)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// readSet extracts and validates the uploaded file; on failure it has already written the response.
func readSet(w http.ResponseWriter, r *http.Request) *quiz.Set {
	r.Body = http.MaxBytesReader(w, r.Body, MaxUpload+4096)
	if err := r.ParseMultipartForm(MaxUpload); err != nil {
		code := http.StatusBadRequest
		if strings.Contains(err.Error(), "too large") {
			code = http.StatusRequestEntityTooLarge
		}
		writeJSON(w, code, map[string]any{"errors": []quiz.FieldError{{Path: "file", Message: "upload failed or larger than 1 MB"}}})
		return nil
	}
	f, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"errors": []quiz.FieldError{{Path: "file", Message: "multipart field \"file\" is required"}}})
		return nil
	}
	defer f.Close()
	set, errs := quiz.Parse(io.LimitReader(f, MaxUpload+1))
	if errs != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": errs})
		return nil
	}
	return set
}

type qSummary struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

func summaries(set *quiz.Set) []qSummary {
	out := make([]qSummary, len(set.Questions))
	for i, q := range set.Questions {
		out[i] = qSummary{q.ID, q.Text}
	}
	return out
}

func (s *Server) preview(w http.ResponseWriter, r *http.Request) {
	set := readSet(w, r)
	if set == nil {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"questionCount": len(set.Questions), "questions": summaries(set)})
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	set := readSet(w, r)
	if set == nil {
		return
	}
	id, key := session.NewSecret()[:16], session.NewSecret()
	if err := s.store.Add(session.NewHub(id, key, set, nil)); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"errors": []quiz.FieldError{{Path: "$", Message: "server is at capacity, try again later"}}})
		return
	}
	base := strings.TrimRight(s.cfg.PublicBaseURL, "/")
	writeJSON(w, http.StatusCreated, map[string]any{
		"sessionId": id, "hostKey": key,
		"questionCount": len(set.Questions), "questions": summaries(set),
		"hostLink": base + "/admin/" + id + "?key=" + key, "shareLink": base + "/join/" + id,
	})
}

func (s *Server) exists(w http.ResponseWriter, r *http.Request) {
	h, ok := s.store.Get(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusOK, map[string]bool{"exists": false, "ended": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"exists": true, "ended": h.Ended()})
}

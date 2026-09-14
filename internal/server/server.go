package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/ai"
	"github.com/CodeMeAPixel/Mellow/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	http     *http.Server
	store    *db.Store
	ai       *ai.Client
	token    string
	statusFn func() any
	countsFn func() (guilds, users int)
}

func New(port int, token string, store *db.Store, aiClient *ai.Client, statusFn func() any, countsFn func() (guilds, users int)) *Server {
	s := &Server{store: store, ai: aiClient, token: token, statusFn: statusFn, countsFn: countsFn}

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(rateLimit(60, time.Minute))

	r.Get("/healthz", s.handleHealth)
	r.Get("/openapi.json", s.handleOpenAPI)
	r.Get("/docs", s.handleDocs)
	r.Route("/v1", func(r chi.Router) {
		r.Get("/stats", s.handleStats)
		r.Get("/status", s.handleStatus)
		r.Get("/testimonials", s.handleTestimonials)
		r.Group(func(r chi.Router) {
			r.Use(s.requireToken)
			r.With(rateLimit(20, time.Minute)).Post("/chat", s.handleChat)
			r.With(rateLimit(5, time.Minute)).Post("/feedback", s.handleFeedback)
		})
	})

	s.http = &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return s
}

func (s *Server) Start() error {
	if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error { return s.http.Shutdown(ctx) }

func (s *Server) requireToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.token != "" && r.Header.Get("Authorization") != "Bearer "+s.token {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if s.statusFn == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "status unavailable"})
		return
	}
	ch := make(chan any, 1)
	go func() { ch <- s.statusFn() }()
	select {
	case v := <-ch:
		writeJSON(w, http.StatusOK, v)
	case <-time.After(5 * time.Second):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "status computation timed out"})
	}
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	st, err := s.store.CommunityStats(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stats unavailable"})
		return
	}
	out := map[string]any{
		"Users":         st.Users,
		"Guilds":        st.Guilds,
		"Conversations": st.Conversations,
		"CrisisEvents":  st.CrisisEvents,
		"MoodCheckIns":  st.MoodCheckIns,
	}
	if s.countsFn != nil {
		g, u := s.countsFn()
		out["Guilds"] = g
		out["Users"] = u
	}
	writeJSON(w, http.StatusOK, out)
}

type chatRequest struct {
	Message string `json:"message"`
}

type chatResponse struct {
	Reply string `json:"reply"`
}

type testimonial struct {
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
	Featured  bool   `json:"featured"`
}

func (s *Server) handleTestimonials(w http.ResponseWriter, r *http.Request) {
	approved := true
	rows, err := s.store.ListFeedback(r.Context(), 50, &approved)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unavailable"})
		return
	}
	out := make([]testimonial, 0, len(rows))
	for _, f := range rows {
		if !f.Public {
			continue
		}
		out = append(out, testimonial{Message: f.Message, CreatedAt: f.CreatedAt.UTC().Format(time.RFC3339), Featured: f.Featured})
	}
	writeJSON(w, http.StatusOK, out)
}

type feedbackRequest struct {
	UserID  string `json:"userId"`
	Message string `json:"message"`
}

func (s *Server) handleFeedback(w http.ResponseWriter, r *http.Request) {
	var req feedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Message) < 3 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message is required"})
		return
	}
	var uid int64
	if req.UserID != "" {
		if n, err := strconv.ParseInt(req.UserID, 10, 64); err == nil {
			uid = n
		}
	}
	if uid != 0 {
		if _, err := s.store.UpsertUser(r.Context(), uid, "web"); err != nil {
			uid = 0
		}
	}
	if uid == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "a valid userId is required"})
		return
	}
	if err := s.store.CreateFeedback(r.Context(), uid, req.Message); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not save feedback"})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "received"})
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message is required"})
		return
	}
	if !s.ai.Live() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "ai is not configured"})
		return
	}
	reply, err := s.ai.Generate(r.Context(), 0, req.Message, ai.GenOpts{IsDM: true})
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "ai request failed"})
		return
	}
	writeJSON(w, http.StatusOK, chatResponse{Reply: reply})
}

package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/PedroEvaldt/shortener/internal/shortener"
	"github.com/PedroEvaldt/shortener/internal/store"
)

type Server struct {
	pg      *store.PostgresStore
	rc      *store.RedisStore
	baseURL string
	logger  *slog.Logger
}

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}

type statsResponse struct {
	Code   string `json:"code"`
	URL    string `json:"url"`
	Clicks int64  `json:"clicks"`
}

type healthResponse struct {
	Status   string `json:"status"`
	Postgres string `json:"postgres"`
	Redis    string `json:"redis"`
}

func New(pg *store.PostgresStore, rc *store.RedisStore, baseURL string, logger *slog.Logger) *Server {
	return &Server{
		pg:      pg,
		rc:      rc,
		baseURL: baseURL,
		logger:  logger,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/shorten", s.handleShorten)
	mux.HandleFunc("GET /api/stats/{code}", s.handleStats)
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /{code}", s.handleRedirect)
	return mux
}

func (s *Server) handleShorten(w http.ResponseWriter, r *http.Request) {
	var req shortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if !isValidURL(req.URL) {
		writeError(w, http.StatusBadRequest, "invalid url")
		return
	}
	code, err := shortener.Generate(7)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed generating short code")
		return
	}
	if err := s.pg.SaveLink(r.Context(), code, req.URL); err != nil {
		writeError(w, http.StatusInternalServerError, "failed saving link")
		return
	}
	writeJSON(w, http.StatusCreated, shortenResponse{
		Code:     code,
		ShortURL: s.baseURL + "/" + code,
	})
}

func (s *Server) handleRedirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		writeError(w, http.StatusNotFound, "invalid code")
		return
	}
	target, _, err := s.pg.GetLink(r.Context(), code)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "code not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed getting link")
		return
	}
	if err := s.rc.IncrementClick(r.Context(), code); err != nil {
		s.logger.Error("failed incrementing click counter", "code", code, "error", err)
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		writeError(w, http.StatusNotFound, "code not found")
		return
	}
	target, persisted, err := s.pg.GetLink(r.Context(), code)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "code not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed getting link")
		return
	}
	pending, err := s.rc.PendingClicks(r.Context(), code)
	if err != nil {
		s.logger.Error("failed getting pending clicks", "code", code, "error", err)
	}
	writeJSON(w, http.StatusOK, statsResponse{
		Code:   code,
		URL:    target,
		Clicks: persisted + pending,
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	pgErr := s.pg.Ping(r.Context())
	rcErr := s.rc.Ping(r.Context())
	if pgErr != nil || rcErr != nil {
		writeJSON(w, http.StatusServiceUnavailable, healthResponse{
			Status:   "unhealthy",
			Postgres: statusStr(pgErr),
			Redis:    statusStr(rcErr),
		})
		return
	}
	writeJSON(w, http.StatusOK, healthResponse{
		Status:   "ok",
		Postgres: "ok",
		Redis:    "ok",
	})
}

func isValidURL(raw string) bool {
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if u.Host == "" {
		return false
	}
	return true
}

func statusStr(err error) string {
	if err == nil {
		return "ok"
	}
	return "error"
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

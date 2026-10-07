package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/facinect/ai/internal/analyze"
	"github.com/facinect/ai/internal/config"
)

type Server struct {
	Cfg config.Config
	AI  *analyze.Service
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/ai/health", s.handleHealth)
	mux.HandleFunc("GET /v1/ai/openapi.json", s.handleOpenAPI)
	mux.HandleFunc("POST /v1/ai/enquiry/analyze", s.withAuth(s.handleAnalyze))
	mux.HandleFunc("POST /v1/ai/reply/suggest", s.withAuth(s.handleSuggest))
	return s.withCORS(mux)
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := s.corsOrigin(); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Service-Key")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) corsOrigin() string {
	o := strings.TrimSpace(s.Cfg.CORSOrigins)
	if o == "" {
		if s.Cfg.IsLocal() {
			return "*"
		}
		return ""
	}
	if o == "*" && !s.Cfg.IsLocal() {
		return ""
	}
	return o
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.authorize(r); err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}
		next(w, r)
	}
}

func (s *Server) authorize(r *http.Request) error {
	key := strings.TrimSpace(r.Header.Get("X-Service-Key"))
	if s.Cfg.ServiceKey != "" && key != "" && key == s.Cfg.ServiceKey {
		return nil
	}
	if s.Cfg.ServiceKey == "" && s.Cfg.IsLocal() {
		return nil
	}
	if s.Cfg.ServiceKey != "" && key == s.Cfg.ServiceKey {
		return nil
	}
	return errUnauthorized
}

var errUnauthorized = &authError{"unauthorized"}

type authError struct{ s string }

func (e *authError) Error() string { return e.s }

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"service":         "ai",
		"status":          "ok",
		"gemini_ready":    s.Cfg.GeminiAPIKey != "",
		"service_key_set": s.Cfg.ServiceKey != "",
		"model":           s.Cfg.GeminiModel,
		"runtime":         "go",
	})
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Transcript string `json:"transcript"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	out, err := s.AI.AnalyzeEnquiry(r.Context(), in.Transcript)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"ai_data": out,
	})
}

func (s *Server) handleSuggest(w http.ResponseWriter, r *http.Request) {
	var in struct {
		FacilityName string                 `json:"facility_name"`
		CustomerName string                 `json:"customer_name"`
		AIData       map[string]interface{} `json:"ai_data"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	text, err := s.AI.SuggestReply(r.Context(), in.FacilityName, in.CustomerName, in.AIData)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"reply":   text,
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

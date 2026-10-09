package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/facinect/tournaments/internal/authn"
	"github.com/facinect/tournaments/internal/config"
	"github.com/facinect/tournaments/internal/tournaments"
)

type Server struct {
	Cfg         config.Config
	Tournaments *tournaments.Service
	JWT         *authn.Verifier
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/tournaments/health", s.handleHealth)
	mux.HandleFunc("GET /v1/tournaments/openapi.json", s.handleOpenAPI)

	mux.HandleFunc("GET /v1/tournaments/facilities/{facilityId}/tournaments/summary", s.withAuth(s.handleSummary))
	mux.HandleFunc("GET /v1/tournaments/facilities/{facilityId}/tournaments", s.withAuth(s.handleList))
	mux.HandleFunc("POST /v1/tournaments/facilities/{facilityId}/tournaments", s.withAuth(s.handleCreate))

	mux.HandleFunc("GET /v1/tournaments/tournaments/{tournamentId}", s.withAuth(s.handleGet))
	mux.HandleFunc("POST /v1/tournaments/tournaments/{tournamentId}", s.withAuth(s.handleUpdate))
	mux.HandleFunc("POST /v1/tournaments/tournaments/{tournamentId}/status", s.withAuth(s.handleStatus))

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

type authCtx struct {
	Principal *authn.Principal
	ViaKey    bool
}

func (s *Server) withAuth(next func(http.ResponseWriter, *http.Request, *authCtx)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ac, err := s.authorize(r)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}
		next(w, r, ac)
	}
}

func (s *Server) authorize(r *http.Request) (*authCtx, error) {
	key := strings.TrimSpace(r.Header.Get("X-Service-Key"))
	if s.Cfg.ServiceKey != "" && key != "" && key == s.Cfg.ServiceKey {
		return &authCtx{ViaKey: true}, nil
	}
	authz := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authz), "bearer ") && s.JWT != nil {
		p, err := s.JWT.ParseAccess(strings.TrimSpace(authz[7:]))
		if err != nil {
			return nil, err
		}
		return &authCtx{Principal: p}, nil
	}
	if s.Cfg.ServiceKey == "" && s.Cfg.IsLocal() {
		return &authCtx{ViaKey: true}, nil
	}
	return nil, errUnauthorized
}

var errUnauthorized = &authError{"unauthorized"}

type authError struct{ s string }

func (e *authError) Error() string { return e.s }

func (s *Server) requireFacility(ac *authCtx, facilityID int64) error {
	if ac.ViaKey || ac.Principal == nil {
		return nil
	}
	if !ac.Principal.CanAccessFacility(facilityID) {
		return &authError{"forbidden_facility"}
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ok := true
	if err := s.Tournaments.DB.Ping(); err != nil {
		ok = false
	}
	status := http.StatusOK
	st := "ok"
	if !ok {
		status = http.StatusServiceUnavailable
		st = "degraded"
	}
	writeJSON(w, status, map[string]interface{}{
		"service":         "tournaments",
		"status":          st,
		"postgres":        ok,
		"service_key_set": s.Cfg.ServiceKey != "",
		"jwt_ready":       s.JWT != nil,
		"runtime":         "go",
	})
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request, ac *authCtx) {
	fid, err := pathInt(r, "facilityId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_facility_id"})
		return
	}
	if err := s.requireFacility(ac, fid); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	list, err := s.Tournaments.List(r.Context(), fid, status)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"tournaments": list})
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request, ac *authCtx) {
	fid, err := pathInt(r, "facilityId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_facility_id"})
		return
	}
	if err := s.requireFacility(ac, fid); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}
	n, err := s.Tournaments.CountActive(r.Context(), fid)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"active_count": n})
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request, ac *authCtx) {
	fid, err := pathInt(r, "facilityId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_facility_id"})
		return
	}
	if err := s.requireFacility(ac, fid); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}
	var in tournaments.UpsertInput
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	row, err := s.Tournaments.Create(r.Context(), fid, in)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request, ac *authCtx) {
	id, err := pathInt(r, "tournamentId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_tournament_id"})
		return
	}
	row, err := s.Tournaments.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	if err := s.requireFacility(ac, row.FacilityID); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request, ac *authCtx) {
	id, err := pathInt(r, "tournamentId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_tournament_id"})
		return
	}
	cur, err := s.Tournaments.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	if err := s.requireFacility(ac, cur.FacilityID); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}
	var in tournaments.UpsertInput
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	row, err := s.Tournaments.Update(r.Context(), cur.FacilityID, id, in)
	if err != nil {
		code := http.StatusBadRequest
		if err.Error() == "facility_mismatch" {
			code = http.StatusForbidden
		}
		writeJSON(w, code, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request, ac *authCtx) {
	id, err := pathInt(r, "tournamentId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_tournament_id"})
		return
	}
	cur, err := s.Tournaments.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	if err := s.requireFacility(ac, cur.FacilityID); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	row, err := s.Tournaments.UpdateStatus(r.Context(), cur.FacilityID, id, body.Status)
	if err != nil {
		code := http.StatusBadRequest
		if err.Error() == "not_found" {
			code = http.StatusNotFound
		}
		writeJSON(w, code, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func pathInt(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

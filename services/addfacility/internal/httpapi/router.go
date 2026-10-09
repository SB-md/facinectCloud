package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/facinect/addfacility/internal/addfacility"
	"github.com/facinect/addfacility/internal/authn"
	"github.com/facinect/addfacility/internal/config"
)

type Server struct {
	Cfg config.Config
	Svc *addfacility.Service
	JWT *authn.Verifier
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/add-facility/health", s.handleHealth)
	mux.HandleFunc("GET /v1/add-facility/openapi.json", s.handleOpenAPI)

	mux.HandleFunc("GET /v1/add-facility/requests/mine", s.withAuth(s.handleListMine))
	mux.HandleFunc("GET /v1/add-facility/requests/pending", s.withAuth(s.handleListPending))
	mux.HandleFunc("GET /v1/add-facility/requests/{requestId}", s.withAuth(s.handleGet))
	mux.HandleFunc("POST /v1/add-facility/requests", s.withAuth(s.handleSubmit))
	mux.HandleFunc("POST /v1/add-facility/requests/{requestId}/approve", s.withAuth(s.handleApprove))
	mux.HandleFunc("POST /v1/add-facility/requests/{requestId}/reject", s.withAuth(s.handleReject))
	mux.HandleFunc("GET /v1/add-facility/summary", s.withAuth(s.handleSummary))

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

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ok := true
	if err := s.Svc.DB.Ping(); err != nil {
		ok = false
	}
	status := http.StatusOK
	st := "ok"
	if !ok {
		status = http.StatusServiceUnavailable
		st = "degraded"
	}
	writeJSON(w, status, map[string]interface{}{
		"service":         "add-facility",
		"status":          st,
		"postgres":        ok,
		"service_key_set": s.Cfg.ServiceKey != "",
		"jwt_ready":       s.JWT != nil,
		"runtime":         "go",
	})
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request, ac *authCtx) {
	var in addfacility.SubmitInput
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	userID, email := s.actor(ac)
	row, err := s.Svc.Submit(r.Context(), userID, email, in)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleListMine(w http.ResponseWriter, r *http.Request, ac *authCtx) {
	userID, _ := s.actor(ac)
	if userID <= 0 {
		if v := strings.TrimSpace(r.URL.Query().Get("user_id")); v != "" {
			n, _ := strconv.ParseInt(v, 10, 64)
			userID = n
		}
	}
	list, err := s.Svc.ListMine(r.Context(), userID, r.URL.Query().Get("status"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"requests": list, "count": len(list)})
}

func (s *Server) handleListPending(w http.ResponseWriter, r *http.Request, ac *authCtx) {
	if !ac.ViaKey && (ac.Principal == nil || !isReviewer(ac.Principal.Role)) {
		// Local: allow any JWT to list pending for demo; prod with service key preferred.
		if !s.Cfg.IsLocal() {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
	}
	list, err := s.Svc.ListPending(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"requests": list, "count": len(list)})
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request, ac *authCtx) {
	id, err := pathInt(r, "requestId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request_id"})
		return
	}
	row, err := s.Svc.Get(r.Context(), id)
	if err != nil {
		code := http.StatusBadRequest
		if err.Error() == "not_found" {
			code = http.StatusNotFound
		}
		writeJSON(w, code, map[string]string{"error": err.Error()})
		return
	}
	userID, _ := s.actor(ac)
	role := ""
	if ac.Principal != nil {
		role = ac.Principal.Role
	}
	if !ac.ViaKey && userID > 0 && row.RequesterUserID != userID && !isReviewer(role) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request, ac *authCtx) {
	if !s.canReview(ac) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	id, err := pathInt(r, "requestId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request_id"})
		return
	}
	var in addfacility.ReviewInput
	_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in)
	reviewer, _ := s.actor(ac)
	row, err := s.Svc.Approve(r.Context(), id, reviewer, in.Note)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleReject(w http.ResponseWriter, r *http.Request, ac *authCtx) {
	if !s.canReview(ac) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	id, err := pathInt(r, "requestId")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request_id"})
		return
	}
	var in addfacility.ReviewInput
	_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in)
	reviewer, _ := s.actor(ac)
	row, err := s.Svc.Reject(r.Context(), id, reviewer, in.Note)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request, ac *authCtx) {
	userID, _ := s.actor(ac)
	if userID <= 0 {
		if v := strings.TrimSpace(r.URL.Query().Get("user_id")); v != "" {
			n, _ := strconv.ParseInt(v, 10, 64)
			userID = n
		}
	}
	sum, err := s.Svc.Summary(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

func (s *Server) actor(ac *authCtx) (int64, string) {
	if ac.Principal != nil {
		return ac.Principal.UserID, ac.Principal.Email
	}
	return 0, ""
}

func (s *Server) canReview(ac *authCtx) bool {
	if ac.ViaKey {
		return true
	}
	if s.Cfg.IsLocal() {
		return true
	}
	return ac.Principal != nil && isReviewer(ac.Principal.Role)
}

func isReviewer(role string) bool {
	r := strings.ToLower(strings.TrimSpace(role))
	return r == "platform" || r == "super_admin" || r == "management"
}

func pathInt(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

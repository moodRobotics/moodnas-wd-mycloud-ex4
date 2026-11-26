// Package api is the MoodNAS HTTP API and static web UI server.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/ai/assistant"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/audit"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/auth"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/monitor"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/state"
)

const (
	cookieName = "moodnas_session"
	csrfHeader = "X-CSRF-Token"
	maxBody    = 64 << 10
)

// Simulator is implemented by the simulated platform only.
type Simulator interface {
	SetScenario(string) error
	Scenario() string
}

type Server struct {
	Store         *state.Store
	Sessions      *auth.Sessions
	Limiter       *auth.Limiter
	Audit         *audit.Log
	Monitor       *monitor.Monitor
	Assistant     *assistant.Assistant
	Platform      platform.Platform
	Sim           Simulator // nil on real hardware
	Log           *slog.Logger
	UIDir         string
	Version       string
	SecureCookies bool
	// PanelSetupCode: the front display shows the setup code (false when WD's
	// firmware owns the display, e.g. sidecar mode): the code is then only in the log.
	PanelSetupCode bool

	setupMu   sync.Mutex
	setupCode string
}

func (s *Server) SetupCode() string {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	return s.setupCode
}

// EnsureSetupCode creates a one-time setup code if no user exists yet.
func (s *Server) EnsureSetupCode() string {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	if len(s.Store.Read().Users) == 0 && s.setupCode == "" {
		s.setupCode = auth.SetupCode()
	}
	s.Monitor.SetSetupCode(s.setupCode)
	return s.setupCode
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	pub := func(pattern string, h http.HandlerFunc) { mux.HandleFunc(pattern, h) }
	usr := func(pattern string, h func(http.ResponseWriter, *http.Request, *auth.Session)) {
		mux.HandleFunc(pattern, s.authed(false, h))
	}
	adm := func(pattern string, h func(http.ResponseWriter, *http.Request, *auth.Session)) {
		mux.HandleFunc(pattern, s.authed(true, h))
	}

	pub("GET /api/v1/bootstrap", s.bootstrap)
	pub("POST /api/v1/setup", s.setup)
	pub("POST /api/v1/login", s.login)
	usr("POST /api/v1/logout", s.logout)
	usr("GET /api/v1/me", s.me)
	usr("POST /api/v1/me/password", s.changePassword)
	usr("POST /api/v1/me/totp/begin", s.totpBegin)
	usr("POST /api/v1/me/totp/confirm", s.totpConfirm)
	usr("POST /api/v1/me/totp/disable", s.totpDisable)

	usr("GET /api/v1/status", s.status)
	usr("GET /api/v1/history", s.history)
	usr("GET /api/v1/shares", s.listShares)
	adm("POST /api/v1/shares", s.createShare)
	adm("PUT /api/v1/shares/{name}", s.updateShare)
	adm("DELETE /api/v1/shares/{name}", s.deleteShare)
	adm("GET /api/v1/users", s.listUsers)
	adm("POST /api/v1/users", s.createUser)
	adm("DELETE /api/v1/users/{name}", s.deleteUser)
	adm("GET /api/v1/settings", s.getSettings)
	adm("PUT /api/v1/settings", s.putSettings)
	adm("POST /api/v1/incidents/{share}/resolve", s.resolveIncident)
	adm("GET /api/v1/audit", s.auditLog)

	adm("GET /api/v1/ai/settings", s.getAISettings)
	adm("PUT /api/v1/ai/settings", s.putAISettings)
	usr("POST /api/v1/ai/chat", s.chat)
	usr("POST /api/v1/ai/actions/{id}", s.decideAction)
	usr("GET /api/v1/ai/ledger", s.ledger)

	adm("POST /api/v1/sim/scenario", s.setScenario)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeErr(w, http.StatusNotFound, "not_found")
	})
	mux.Handle("/", s.static())
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if r.TLS != nil {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			h.Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

// authed checks the session cookie, the CSRF header on state-changing
// requests, and the admin role when required.
func (s *Server) authed(admin bool, h func(http.ResponseWriter, *http.Request, *auth.Session)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "auth.required")
			return
		}
		sess := s.Sessions.Get(c.Value)
		if sess == nil {
			writeErr(w, http.StatusUnauthorized, "auth.required")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if !constantEq(r.Header.Get(csrfHeader), sess.CSRF) {
				writeErr(w, http.StatusForbidden, "auth.csrf")
				return
			}
		}
		if admin && sess.Role != state.RoleAdmin {
			writeErr(w, http.StatusForbidden, "auth.adminOnly")
			return
		}
		h(w, r, sess)
	}
}

func (s *Server) static() http.Handler {
	fs := http.FileServer(http.Dir(s.UIDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Join(s.UIDir, filepath.Clean("/"+r.URL.Path))
		if st, err := os.Stat(p); err != nil || st.IsDir() {
			// SPA fallback
			http.ServeFile(w, r, filepath.Join(s.UIDir, "index.html"))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		fs.ServeHTTP(w, r)
	})
}

// ---- helpers ----

type apiError struct {
	Error  string         `json:"error"` // i18n key
	Params map[string]any `json:"params,omitempty"`
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, key string) {
	writeJSON(w, code, apiError{Error: key})
}

func writeErrP(w http.ResponseWriter, code int, key string, params map[string]any) {
	writeJSON(w, code, apiError{Error: key, Params: params})
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, "request.invalid")
		return false
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		writeErr(w, http.StatusBadRequest, "request.invalid")
		return false
	}
	return true
}

// clientIP deliberately ignores X-Forwarded-For: MoodNAS is not meant to sit
// behind a proxy, and trusting that header would let anyone dodge throttling.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

var errStop = errors.New("stop")

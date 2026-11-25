package api

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/auth"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/state"
)

var (
	reUser     = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,31}$`)
	reHostname = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,30}[a-z0-9])?$`)
	languages  = map[string]bool{"es": true, "en": true, "pt": true, "fr": true, "de": true, "it": true, "zh": true, "ja": true}
)

func constantEq(a, b string) bool {
	return a != "" && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (s *Server) bootstrap(w http.ResponseWriter, r *http.Request) {
	d := s.Store.Read()
	writeJSON(w, http.StatusOK, map[string]any{
		"setupRequired": len(d.Users) == 0,
		"hostname":      d.Hostname,
		"language":      d.Language,
		"version":       s.Version,
		"simulator":     s.Sim != nil,
	})
}

func (s *Server) startSession(w http.ResponseWriter, u state.User) *auth.Session {
	sess := s.Sessions.Create(u.Name, u.Role)
	http.SetCookie(w, &http.Cookie{
		Name: cookieName, Value: sess.Token, Path: "/", HttpOnly: true,
		Secure: s.SecureCookies, SameSite: http.SameSiteStrictMode,
		MaxAge: int(auth.AbsoluteTimeout.Seconds()),
	})
	return sess
}

func sessionJSON(sess *auth.Session, totp bool) map[string]any {
	return map[string]any{"user": sess.User, "role": sess.Role, "csrf": sess.CSRF, "totp": totp}
}

func (s *Server) setup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SetupCode string `json:"setupCode"`
		User      string `json:"user"`
		Password  string `json:"password"`
		Hostname  string `json:"hostname"`
		Language  string `json:"language"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	ip := clientIP(r)
	if ok, wait := s.Limiter.Allowed("setup:"+ip, time.Now()); !ok {
		writeErrP(w, http.StatusTooManyRequests, "auth.locked", map[string]any{"seconds": int(wait.Seconds()) + 1})
		return
	}
	code := strings.ToUpper(strings.TrimSpace(req.SetupCode))
	if !constantEq(code, s.SetupCode()) {
		s.Limiter.Fail("setup:"+ip, time.Now())
		s.Audit.Add("-", "setup.badCode", ip, nil)
		writeErr(w, http.StatusForbidden, "setup.badCode")
		return
	}
	if !reUser.MatchString(req.User) {
		writeErr(w, http.StatusBadRequest, "user.invalidName")
		return
	}
	if err := auth.ValidatePassword(req.Password); err != nil {
		writeErr(w, http.StatusBadRequest, "user.weakPassword")
		return
	}
	if req.Hostname != "" && !reHostname.MatchString(req.Hostname) {
		writeErr(w, http.StatusBadRequest, "settings.invalidHostname")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	u := state.User{Name: req.User, Hash: hash, Role: state.RoleAdmin, Created: time.Now().UTC()}
	err = s.Store.Update(func(d *state.Data) error {
		if len(d.Users) > 0 {
			return fmt.Errorf("already set up")
		}
		d.Users = append(d.Users, u)
		if req.Hostname != "" {
			d.Hostname = req.Hostname
		}
		if languages[req.Language] {
			d.Language = req.Language
		}
		if len(d.Shares) == 0 {
			d.Shares = []state.Share{
				{Name: "Public", Description: "", Users: []string{"*"}, Sentinel: true},
				{Name: req.User, Description: "", Users: []string{req.User}, Sentinel: true},
			}
		}
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusConflict, "setup.done")
		return
	}
	s.setupMu.Lock()
	s.setupCode = ""
	s.setupMu.Unlock()
	s.Monitor.SetSetupCode("")
	s.Monitor.Tick() // take the setup code off the front panel right away
	s.Audit.Add(u.Name, "setup.completed", ip, nil)
	s.Log.Info("first-time setup completed", "admin", u.Name)
	writeJSON(w, http.StatusOK, sessionJSON(s.startSession(w, u), false))
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		User     string `json:"user"`
		Password string `json:"password"`
		TOTP     string `json:"totp"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	ip := clientIP(r)
	now := time.Now()
	if ok, wait := s.Limiter.Allowed(ip, now); !ok {
		writeErrP(w, http.StatusTooManyRequests, "auth.locked", map[string]any{"seconds": int(wait.Seconds()) + 1})
		return
	}
	d := s.Store.Read()
	u := d.FindUser(req.User)
	hash := ""
	if u != nil {
		hash = u.Hash
	}
	if !auth.VerifyOrDummy(req.Password, hash) {
		s.Limiter.Fail(ip, now)
		s.Audit.Add(req.User, "login.failed", ip, nil)
		writeErr(w, http.StatusUnauthorized, "auth.invalid")
		return
	}
	if u.TOTPSecret != "" {
		if req.TOTP == "" {
			writeErr(w, http.StatusUnauthorized, "auth.totpRequired")
			return
		}
		if !auth.VerifyTOTP(u.TOTPSecret, req.TOTP, now) {
			s.Limiter.Fail(ip, now)
			s.Audit.Add(u.Name, "login.badTotp", ip, nil)
			writeErr(w, http.StatusUnauthorized, "auth.badTotp")
			return
		}
	}
	s.Limiter.Success(ip)
	s.Audit.Add(u.Name, "login", ip, nil)
	writeJSON(w, http.StatusOK, sessionJSON(s.startSession(w, *u), u.TOTPSecret != ""))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	s.Sessions.Delete(sess.Token)
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.SecureCookies, SameSite: http.SameSiteStrictMode})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	d := s.Store.Read()
	u := d.FindUser(sess.User)
	writeJSON(w, http.StatusOK, sessionJSON(sess, u != nil && u.TOTPSecret != ""))
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	var req struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	d := s.Store.Read()
	u := d.FindUser(sess.User)
	if u == nil || !auth.VerifyPassword(req.Current, u.Hash) {
		s.Limiter.Fail(clientIP(r), time.Now())
		writeErr(w, http.StatusForbidden, "auth.invalid")
		return
	}
	if auth.ValidatePassword(req.New) != nil {
		writeErr(w, http.StatusBadRequest, "user.weakPassword")
		return
	}
	hash, _ := auth.HashPassword(req.New)
	if err := s.Store.Update(func(d *state.Data) error {
		d.FindUser(sess.User).Hash = hash
		return nil
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	s.Sessions.DeleteUser(sess.User)
	s.Audit.Add(sess.User, "password.changed", clientIP(r), nil)
	d2 := s.Store.Read()
	u2 := *d2.FindUser(sess.User)
	writeJSON(w, http.StatusOK, sessionJSON(s.startSession(w, u2), u2.TOTPSecret != ""))
}

func (s *Server) totpBegin(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	secret := auth.NewTOTPSecret()
	writeJSON(w, http.StatusOK, map[string]any{
		"secret": secret, "url": auth.TOTPURL(secret, sess.User, "MoodNAS "+s.Store.Read().Hostname),
	})
}

func (s *Server) totpConfirm(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	var req struct {
		Secret string `json:"secret"`
		Code   string `json:"code"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if !auth.VerifyTOTP(req.Secret, req.Code, time.Now()) {
		writeErr(w, http.StatusBadRequest, "auth.badTotp")
		return
	}
	_ = s.Store.Update(func(d *state.Data) error {
		d.FindUser(sess.User).TOTPSecret = req.Secret
		return nil
	})
	s.Audit.Add(sess.User, "totp.enabled", clientIP(r), nil)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) totpDisable(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	var req struct {
		Password string `json:"password"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	d := s.Store.Read()
	u := d.FindUser(sess.User)
	if u == nil || !auth.VerifyPassword(req.Password, u.Hash) {
		writeErr(w, http.StatusForbidden, "auth.invalid")
		return
	}
	_ = s.Store.Update(func(d *state.Data) error {
		d.FindUser(sess.User).TOTPSecret = ""
		return nil
	})
	s.Audit.Add(sess.User, "totp.disabled", clientIP(r), nil)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

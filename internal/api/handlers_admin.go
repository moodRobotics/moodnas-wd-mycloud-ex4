package api

import (
	"errors"
	"net/http"
	"regexp"
	"slices"
	"time"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/auth"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/state"
)

var reShare = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 _.-]{0,31}$`)

var (
	errShareExists  = errors.New("share.exists")
	errShareInvalid = errors.New("share.invalidName")
	errShareMissing = errors.New("share.notFound")
	errUnknownUser  = errors.New("user.notFound")
)

func (s *Server) status(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	snap := s.Monitor.Snapshot()
	resp := map[string]any{"snapshot": snap}
	if s.Sim != nil {
		resp["scenario"] = s.Sim.Scenario()
	}
	writeJSON(w, http.StatusOK, resp)
}

// visibleShares: admins see everything, users see shares they may access.
func visibleShares(d state.Data, sess *auth.Session) []state.Share {
	out := []state.Share{}
	for _, sh := range d.Shares {
		if sess.Role == state.RoleAdmin || slices.Contains(sh.Users, "*") || slices.Contains(sh.Users, sess.User) {
			out = append(out, sh)
		}
	}
	return out
}

func (s *Server) listShares(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	writeJSON(w, http.StatusOK, visibleShares(s.Store.Read(), sess))
}

func validateShare(d *state.Data, sh state.Share) error {
	if !reShare.MatchString(sh.Name) {
		return errShareInvalid
	}
	for _, u := range sh.Users {
		if u != "*" && d.FindUser(u) == nil {
			return errUnknownUser
		}
	}
	return nil
}

// CreateShare is shared by the API and the assistant's confirmed actions.
func (s *Server) CreateShare(actor, ip string, sh state.Share) error {
	err := s.Store.Update(func(d *state.Data) error {
		if d.FindShare(sh.Name) != nil {
			return errShareExists
		}
		if err := validateShare(d, sh); err != nil {
			return err
		}
		if sh.Users == nil {
			sh.Users = []string{}
		}
		d.Shares = append(d.Shares, sh)
		return nil
	})
	if err == nil {
		s.Audit.Add(actor, "share.created", ip, map[string]string{"share": sh.Name})
	}
	return err
}

func (s *Server) createShare(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	var sh state.Share
	if !readJSON(w, r, &sh) {
		return
	}
	if err := s.CreateShare(sess.User, clientIP(r), sh); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, sh)
}

func (s *Server) updateShare(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	name := r.PathValue("name")
	var sh state.Share
	if !readJSON(w, r, &sh) {
		return
	}
	sh.Name = name
	err := s.Store.Update(func(d *state.Data) error {
		cur := d.FindShare(name)
		if cur == nil {
			return errShareMissing
		}
		if err := validateShare(d, sh); err != nil {
			return err
		}
		*cur = sh
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.Audit.Add(sess.User, "share.updated", clientIP(r), map[string]string{"share": name})
	writeJSON(w, http.StatusOK, sh)
}

func (s *Server) deleteShare(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	name := r.PathValue("name")
	err := s.Store.Update(func(d *state.Data) error {
		i := slices.IndexFunc(d.Shares, func(x state.Share) bool { return x.Name == name })
		if i < 0 {
			return errShareMissing
		}
		d.Shares = slices.Delete(d.Shares, i, i+1)
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	// Only the share definition is removed; the data stays on disk.
	s.Audit.Add(sess.User, "share.deleted", clientIP(r), map[string]string{"share": name})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type userView struct {
	Name    string     `json:"name"`
	Role    state.Role `json:"role"`
	TOTP    bool       `json:"totp"`
	Created time.Time  `json:"created"`
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	out := []userView{}
	for _, u := range s.Store.Read().Users {
		out = append(out, userView{u.Name, u.Role, u.TOTPSecret != "", u.Created})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	var req struct {
		Name     string     `json:"name"`
		Password string     `json:"password"`
		Role     state.Role `json:"role"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if !reUser.MatchString(req.Name) {
		writeErr(w, http.StatusBadRequest, "user.invalidName")
		return
	}
	if req.Role != state.RoleAdmin && req.Role != state.RoleUser {
		writeErr(w, http.StatusBadRequest, "request.invalid")
		return
	}
	if auth.ValidatePassword(req.Password) != nil {
		writeErr(w, http.StatusBadRequest, "user.weakPassword")
		return
	}
	hash, _ := auth.HashPassword(req.Password)
	err := s.Store.Update(func(d *state.Data) error {
		if d.FindUser(req.Name) != nil {
			return errors.New("user.exists")
		}
		d.Users = append(d.Users, state.User{Name: req.Name, Hash: hash, Role: req.Role, Created: time.Now().UTC()})
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	s.Audit.Add(sess.User, "user.created", clientIP(r), map[string]string{"user": req.Name, "role": string(req.Role)})
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	name := r.PathValue("name")
	if name == sess.User {
		writeErr(w, http.StatusBadRequest, "user.cannotDeleteSelf")
		return
	}
	err := s.Store.Update(func(d *state.Data) error {
		i := slices.IndexFunc(d.Users, func(u state.User) bool { return u.Name == name })
		if i < 0 {
			return errUnknownUser
		}
		d.Users = slices.Delete(d.Users, i, i+1)
		for j := range d.Shares {
			d.Shares[j].Users = slices.DeleteFunc(d.Shares[j].Users, func(u string) bool { return u == name })
		}
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	s.Sessions.DeleteUser(name)
	s.Audit.Add(sess.User, "user.deleted", clientIP(r), map[string]string{"user": name})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	d := s.Store.Read()
	writeJSON(w, http.StatusOK, map[string]any{"hostname": d.Hostname, "language": d.Language, "fan": d.Fan})
}

func (s *Server) putSettings(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	var req struct {
		Hostname string            `json:"hostname"`
		Language string            `json:"language"`
		Fan      state.FanSettings `json:"fan"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if !reHostname.MatchString(req.Hostname) {
		writeErr(w, http.StatusBadRequest, "settings.invalidHostname")
		return
	}
	if err := s.applyFan(req.Fan); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = s.Store.Update(func(d *state.Data) error {
		d.Hostname = req.Hostname
		if languages[req.Language] {
			d.Language = req.Language
		}
		return nil
	})
	s.Audit.Add(sess.User, "settings.updated", clientIP(r), nil)
	s.Monitor.Tick()
	s.getSettings(w, r, sess)
}

// applyFan validates and stores fan settings (also used by the assistant).
func (s *Server) applyFan(f state.FanSettings) error {
	if (f.Mode != "ai" && f.Mode != "manual") || f.ManualPct < 20 || f.ManualPct > 100 ||
		f.QuietStart < 0 || f.QuietStart > 23 || f.QuietEnd < 0 || f.QuietEnd > 23 {
		return errors.New("settings.invalidFan")
	}
	return s.Store.Update(func(d *state.Data) error { d.Fan = f; return nil })
}

func (s *Server) resolveIncident(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	share := r.PathValue("share")
	if err := s.Monitor.ResolveIncident(share); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	s.Audit.Add(sess.User, "incident.resolved", clientIP(r), map[string]string{"share": share})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) auditLog(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	writeJSON(w, http.StatusOK, s.Audit.Recent(200))
}

func (s *Server) setScenario(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	if s.Sim == nil {
		writeErr(w, http.StatusNotFound, "not_found")
		return
	}
	var req struct {
		Scenario string `json:"scenario"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.Sim.SetScenario(req.Scenario); err != nil {
		writeErr(w, http.StatusBadRequest, "request.invalid")
		return
	}
	if req.Scenario != "ransomware" {
		for _, inc := range s.Monitor.Sentinel.Active() {
			_ = s.Monitor.ResolveIncident(inc.Share)
		}
	}
	s.Monitor.Tick()
	writeJSON(w, http.StatusOK, map[string]any{"scenario": req.Scenario})
}

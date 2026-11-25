package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/ai/assistant"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/auth"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/state"
)

var aiFeatures = []string{"diskrisk", "sentinel", "fan", "forecast", "assistant", "explain"}

func (s *Server) getAISettings(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	ai := s.Store.Read().AI
	writeJSON(w, http.StatusOK, map[string]any{
		"provider": ai.Provider, "model": ai.Model, "endpoint": ai.Endpoint,
		"hasKey": ai.APIKey != "", "features": ai.Features,
	})
}

func (s *Server) putAISettings(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	var req struct {
		Provider string          `json:"provider"`
		Model    string          `json:"model"`
		Endpoint string          `json:"endpoint"`
		APIKey   *string         `json:"apiKey"` // nil = keep, "" = remove
		Features map[string]bool `json:"features"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	switch req.Provider {
	case "none", "anthropic":
	case "ollama":
		if _, err := assistant.NewOllama(req.Endpoint, req.Model); err != nil {
			writeErr(w, http.StatusBadRequest, "ai.invalidEndpoint")
			return
		}
	default:
		writeErr(w, http.StatusBadRequest, "request.invalid")
		return
	}
	if len(req.Model) > 100 {
		writeErr(w, http.StatusBadRequest, "request.invalid")
		return
	}
	_ = s.Store.Update(func(d *state.Data) error {
		d.AI.Provider, d.AI.Model, d.AI.Endpoint = req.Provider, req.Model, req.Endpoint
		if req.APIKey != nil {
			d.AI.APIKey = *req.APIKey
		}
		for _, f := range aiFeatures {
			if v, ok := req.Features[f]; ok {
				d.AI.Features[f] = v
			}
		}
		return nil
	})
	s.Audit.Add(sess.User, "ai.settings", clientIP(r), map[string]string{"provider": req.Provider, "model": req.Model})
	s.Monitor.Tick()
	s.getAISettings(w, r, sess)
}

func (s *Server) provider() (assistant.Provider, error) {
	ai := s.Store.Read().AI
	switch ai.Provider {
	case "anthropic":
		if ai.APIKey == "" {
			return nil, errors.New("ai.noKey")
		}
		return assistant.NewClaude(ai.APIKey, ai.Model), nil
	case "ollama":
		return assistant.NewOllama(ai.Endpoint, ai.Model)
	}
	return nil, errors.New("ai.disabled")
}

func (s *Server) chat(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	var req struct {
		History []assistant.Turn `json:"history"`
		Message string           `json:"message"`
		Lang    string           `json:"lang"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	d := s.Store.Read()
	if !d.AI.Features["assistant"] {
		writeErr(w, http.StatusConflict, "ai.disabled")
		return
	}
	if req.Message == "" || len(req.Message) > 4000 {
		writeErr(w, http.StatusBadRequest, "request.invalid")
		return
	}
	p, err := s.provider()
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	res, err := s.Assistant.Chat(ctx, p, sess.User, req.Lang, d.Hostname, req.History, req.Message, s.tools(sess))
	if err != nil {
		s.Log.Warn("assistant", "err", err)
		writeErrP(w, http.StatusBadGateway, "ai.providerError", map[string]any{"detail": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) decideAction(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	var req struct {
		Confirm bool `json:"confirm"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	id := r.PathValue("id")
	if !req.Confirm {
		s.Assistant.Discard(id, sess.User)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	pa, ok := s.Assistant.Take(id, sess.User)
	if !ok {
		writeErr(w, http.StatusNotFound, "ai.actionExpired")
		return
	}
	for _, t := range s.tools(sess) {
		if t.Name == pa.Tool && t.Mutating {
			if _, err := t.Run(r.Context(), pa.Input); err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			s.Audit.Add(sess.User, "ai.actionConfirmed", clientIP(r), map[string]string{"tool": pa.Tool, "summary": pa.Summary})
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
	}
	writeErr(w, http.StatusForbidden, "auth.adminOnly")
}

func (s *Server) ledger(w http.ResponseWriter, r *http.Request, sess *auth.Session) {
	writeJSON(w, http.StatusOK, s.Assistant.Ledger())
}

// tools are what the assistant may use; mutating ones are only offered to
// admins and always go through a confirmation in the UI.
func (s *Server) tools(sess *auth.Session) []assistant.Tool {
	js := func(v any) (string, error) {
		b, err := json.Marshal(v)
		return string(b), err
	}
	ts := []assistant.Tool{
		{
			Name:        "get_status",
			Description: "Current mood of the NAS with its reasons, CPU/board temperatures, fan decision, load, memory, uptime and network links.",
			Schema:      map[string]any{},
			Run: func(ctx context.Context, _ json.RawMessage) (string, error) {
				sn := s.Monitor.Snapshot()
				return js(map[string]any{"mood": sn.Mood, "sensors": sn.Sensors, "fan": sn.Fan, "system": sn.System})
			},
		},
		{
			Name:        "get_storage",
			Description: "Disks per bay with SMART attributes and the local AI failure-risk estimate, RAID arrays and their state, capacity use and the AI forecast of when the volume fills up.",
			Schema:      map[string]any{},
			Run: func(ctx context.Context, _ json.RawMessage) (string, error) {
				sn := s.Monitor.Snapshot()
				return js(map[string]any{"disks": sn.Disks, "risks": sn.Risks, "arrays": sn.Arrays, "fill": sn.Fill})
			},
		},
		{
			Name:        "get_security",
			Description: "Ransomware sentinel incidents (shares frozen read-only) and the most recent security audit events.",
			Schema:      map[string]any{},
			Run: func(ctx context.Context, _ json.RawMessage) (string, error) {
				out := map[string]any{"incidents": s.Monitor.Sentinel.Incidents()}
				if sess.Role == state.RoleAdmin {
					out["recentAudit"] = s.Audit.Recent(30)
				}
				return js(out)
			},
		},
		{
			Name:        "list_shares",
			Description: "Shared folders visible to the current user, with their access lists and options.",
			Schema:      map[string]any{},
			Run: func(ctx context.Context, _ json.RawMessage) (string, error) {
				return js(visibleShares(s.Store.Read(), sess))
			},
		},
	}
	if sess.Role != state.RoleAdmin {
		return ts
	}
	type shareIn struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Users       []string `json:"users"`
		ReadOnly    bool     `json:"readOnly"`
	}
	type fanIn struct {
		Mode string `json:"mode"`
		Pct  int    `json:"pct"`
	}
	type shareRef struct {
		Share string `json:"share"`
	}
	ts = append(ts,
		assistant.Tool{
			Name:        "list_users",
			Description: "User accounts (names, roles, whether two-factor authentication is on).",
			Schema:      map[string]any{},
			Run: func(ctx context.Context, _ json.RawMessage) (string, error) {
				var out []map[string]any
				for _, u := range s.Store.Read().Users {
					out = append(out, map[string]any{"name": u.Name, "role": u.Role, "twoFactor": u.TOTPSecret != ""})
				}
				return js(out)
			},
		},
		assistant.Tool{
			Name:        "create_share",
			Description: `Create a shared folder. users lists the account names allowed to access it; ["*"] means every user. Requires the user's confirmation in the UI.`,
			Schema: map[string]any{
				"name":        map[string]any{"type": "string", "description": "1-32 chars: letters, digits, space, _ . -"},
				"description": map[string]any{"type": "string"},
				"users":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"readOnly":    map[string]any{"type": "boolean"},
			},
			Required: []string{"name", "users"},
			Mutating: true,
			Summary: func(in json.RawMessage) string {
				var v shareIn
				_ = json.Unmarshal(in, &v)
				return fmt.Sprintf("Create share %q for %v", v.Name, v.Users)
			},
			Run: func(ctx context.Context, in json.RawMessage) (string, error) {
				var v shareIn
				if err := json.Unmarshal(in, &v); err != nil {
					return "", errors.New("request.invalid")
				}
				err := s.CreateShare(sess.User+" (via assistant)", "", state.Share{Name: v.Name, Description: v.Description, Users: v.Users, ReadOnly: v.ReadOnly, Sentinel: true})
				if err != nil {
					return "", err
				}
				return "created", nil
			},
		},
		assistant.Tool{
			Name:        "set_fan_mode",
			Description: `Set the fan to "ai" (predictive automatic control) or "manual" with pct between 20 and 100. Critical temperatures always override to 100%. Requires confirmation.`,
			Schema: map[string]any{
				"mode": map[string]any{"type": "string", "enum": []string{"ai", "manual"}},
				"pct":  map[string]any{"type": "integer", "minimum": 20, "maximum": 100},
			},
			Required: []string{"mode"},
			Mutating: true,
			Summary: func(in json.RawMessage) string {
				var v fanIn
				_ = json.Unmarshal(in, &v)
				if v.Mode == "manual" {
					return fmt.Sprintf("Set fan to manual %d%%", v.Pct)
				}
				return "Set fan to AI predictive mode"
			},
			Run: func(ctx context.Context, in json.RawMessage) (string, error) {
				var v fanIn
				if err := json.Unmarshal(in, &v); err != nil {
					return "", errors.New("request.invalid")
				}
				f := s.Store.Read().Fan
				f.Mode = v.Mode
				if v.Mode == "manual" {
					f.ManualPct = v.Pct
				}
				if err := s.applyFan(f); err != nil {
					return "", err
				}
				s.Audit.Add(sess.User+" (via assistant)", "fan.mode", "", map[string]string{"mode": v.Mode})
				s.Monitor.Tick()
				return "done", nil
			},
		},
		assistant.Tool{
			Name:        "unfreeze_share",
			Description: "Lift the sentinel's read-only freeze on a share after a ransomware alert, once the user has checked their devices. Requires confirmation.",
			Schema:      map[string]any{"share": map[string]any{"type": "string"}},
			Required:    []string{"share"},
			Mutating:    true,
			Summary: func(in json.RawMessage) string {
				var v shareRef
				_ = json.Unmarshal(in, &v)
				return fmt.Sprintf("Unfreeze share %q (mark the incident resolved)", v.Share)
			},
			Run: func(ctx context.Context, in json.RawMessage) (string, error) {
				var v shareRef
				if err := json.Unmarshal(in, &v); err != nil {
					return "", errors.New("request.invalid")
				}
				if err := s.Monitor.ResolveIncident(v.Share); err != nil {
					return "", err
				}
				s.Audit.Add(sess.User+" (via assistant)", "incident.resolved", "", map[string]string{"share": v.Share})
				return "done", nil
			},
		},
	)
	return ts
}

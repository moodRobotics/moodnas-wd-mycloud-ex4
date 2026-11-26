package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/ai/assistant"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/ai/local"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/audit"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/auth"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/history"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/monitor"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform/sim"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/state"
)

type client struct {
	t    *testing.T
	base string
	hc   *http.Client
	csrf string
}

func (c *client) do(method, path string, body any) (int, map[string]any) {
	c.t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.base+path, r)
	if c.csrf != "" {
		req.Header.Set(csrfHeader, c.csrf)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	raw, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		out = map[string]any{"_raw": string(raw)}
	}
	return resp.StatusCode, out
}

func newTestServer(t *testing.T) (*Server, *httptest.Server) {
	dir := t.TempDir()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	store, err := state.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	al, _ := audit.Open(dir)
	p := sim.New(log)
	mon := &monitor.Monitor{P: p, Store: store, Audit: al, Log: log, Interval: time.Hour,
		DiskModel: local.DefaultDiskModel, Sentinel: local.NewSentinel(),
		Forecast: local.NewForecast(time.Second, time.Minute), Fan: local.NewFanController(23, 7),
		History: history.New("", time.Second)}
	mon.Tick()
	s := &Server{Store: store, Sessions: auth.NewSessions(), Limiter: auth.NewLimiter(), Audit: al,
		Monitor: mon, Assistant: assistant.New(), Platform: p, Sim: p, Log: log, UIDir: dir, Version: "test"}
	s.EnsureSetupCode()
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return s, ts
}

func newClient(t *testing.T, base string) *client {
	jar, _ := cookiejar.New(nil)
	return &client{t: t, base: base, hc: &http.Client{Jar: jar}}
}

func TestSetupLoginAndPermissions(t *testing.T) {
	s, ts := newTestServer(t)
	c := newClient(t, ts.URL)

	if code, b := c.do("GET", "/api/v1/bootstrap", nil); code != 200 || b["setupRequired"] != true {
		t.Fatalf("bootstrap %d %v", code, b)
	}
	if code, _ := c.do("GET", "/api/v1/status", nil); code != 401 {
		t.Fatalf("status without session should be 401, got %d", code)
	}
	// wrong setup code
	if code, b := c.do("POST", "/api/v1/setup", map[string]any{"setupCode": "AAAA-AAAA", "user": "ana", "password": "a-long-password"}); code != 403 {
		t.Fatalf("bad setup code accepted: %d %v", code, b)
	}
	// weak password
	if code, _ := c.do("POST", "/api/v1/setup", map[string]any{"setupCode": s.SetupCode(), "user": "ana", "password": "short"}); code != 400 {
		t.Fatalf("weak password accepted: %d", code)
	}
	code, b := c.do("POST", "/api/v1/setup", map[string]any{"setupCode": strings.ToLower(s.SetupCode()), "user": "ana", "password": "a-long-password", "hostname": "casa", "language": "es"})
	if code != 200 || b["role"] != "admin" {
		t.Fatalf("setup failed %d %v", code, b)
	}
	c.csrf = b["csrf"].(string)
	if s.SetupCode() != "" {
		t.Fatal("setup code must be cleared after setup")
	}
	// setup cannot run twice
	if code, _ := c.do("POST", "/api/v1/setup", map[string]any{"setupCode": "X", "user": "eve", "password": "a-long-password"}); code == 200 {
		t.Fatal("second setup accepted")
	}

	// CSRF: a state-changing request without the header is refused
	noCSRF := &client{t: t, base: ts.URL, hc: c.hc}
	if code, _ := noCSRF.do("POST", "/api/v1/shares", map[string]any{"name": "Docs", "users": []string{"*"}}); code != 403 {
		t.Fatalf("missing CSRF accepted: %d", code)
	}
	if code, b := c.do("POST", "/api/v1/shares", map[string]any{"name": "Docs", "users": []string{"*"}}); code != 201 {
		t.Fatalf("create share %d %v", code, b)
	}
	if code, _ := c.do("POST", "/api/v1/shares", map[string]any{"name": "../etc", "users": []string{}}); code != 400 {
		t.Fatal("path-like share name accepted")
	}
	if code, _ := c.do("POST", "/api/v1/users", map[string]any{"name": "bob", "password": "another-long-pw", "role": "user"}); code != 201 {
		t.Fatal("create user failed")
	}
	// unknown fields are rejected
	if code, _ := c.do("POST", "/api/v1/users", map[string]any{"name": "eve", "password": "another-long-pw", "role": "user", "isRoot": true}); code != 400 {
		t.Fatal("unknown field accepted")
	}

	// AI key is write-only
	key := "sk-test-secret"
	c.do("PUT", "/api/v1/ai/settings", map[string]any{"provider": "anthropic", "model": "claude-opus-5-5", "endpoint": "", "apiKey": key, "features": map[string]bool{"assistant": true}})
	_, ai := c.do("GET", "/api/v1/ai/settings", nil)
	if ai["hasKey"] != true || strings.Contains(toJSON(ai), key) {
		t.Fatalf("API key leaked or not stored: %v", ai)
	}

	// regular user: no admin endpoints, sees only permitted shares
	u := newClient(t, ts.URL)
	code, b = u.do("POST", "/api/v1/login", map[string]any{"user": "bob", "password": "another-long-pw"})
	if code != 200 || b["role"] != "user" {
		t.Fatalf("bob login %d %v", code, b)
	}
	u.csrf = b["csrf"].(string)
	if code, _ := u.do("GET", "/api/v1/users", nil); code != 403 {
		t.Fatal("user reached admin endpoint")
	}
	if code, _ := u.do("POST", "/api/v1/shares", map[string]any{"name": "Mine", "users": []string{}}); code != 403 {
		t.Fatal("user created share")
	}
	if code, b := u.do("GET", "/api/v1/status", nil); code != 200 || b["snapshot"] == nil {
		t.Fatal("user cannot read status")
	}
	s.Monitor.Tick()
	code, b = u.do("GET", "/api/v1/history?range=6h", nil)
	pts, _ := b["points"].([]any)
	if code != 200 || len(pts) != 240 {
		t.Fatalf("history: %d, %d points", code, len(pts))
	}
	last := pts[len(pts)-1].(map[string]any)
	if last["cpu"] == nil || last["mem"] == nil {
		t.Fatalf("latest history point empty: %v", last)
	}
}

func TestLoginThrottle(t *testing.T) {
	s, ts := newTestServer(t)
	c := newClient(t, ts.URL)
	_, b := c.do("POST", "/api/v1/setup", map[string]any{"setupCode": s.SetupCode(), "user": "ana", "password": "a-long-password"})
	if b["csrf"] == nil {
		t.Fatal("setup failed")
	}
	x := newClient(t, ts.URL)
	for i := 0; i < 5; i++ {
		if code, _ := x.do("POST", "/api/v1/login", map[string]any{"user": "ana", "password": "wrong-password"}); code != 401 {
			t.Fatalf("attempt %d: %d", i, code)
		}
	}
	if code, b := x.do("POST", "/api/v1/login", map[string]any{"user": "ana", "password": "a-long-password"}); code != 429 || b["error"] != "auth.locked" {
		t.Fatalf("expected lock-out, got %d %v", code, b)
	}
}

func TestSecurityHeaders(t *testing.T) {
	_, ts := newTestServer(t)
	resp, err := http.Get(ts.URL + "/api/v1/bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	for _, h := range []string{"Content-Security-Policy", "X-Frame-Options", "X-Content-Type-Options", "Referrer-Policy"} {
		if resp.Header.Get(h) == "" {
			t.Errorf("missing %s", h)
		}
	}
}

func toJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

package auth

import (
	"sync"
	"time"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/state"
)

const (
	IdleTimeout     = 30 * time.Minute
	AbsoluteTimeout = 12 * time.Hour
)

type Session struct {
	Token    string
	User     string
	Role     state.Role
	CSRF     string
	Created  time.Time
	LastSeen time.Time
}

type Sessions struct {
	mu sync.Mutex
	m  map[string]*Session
}

func NewSessions() *Sessions { return &Sessions{m: map[string]*Session{}} }

func (s *Sessions) Create(user string, role state.Role) *Session {
	now := time.Now()
	sess := &Session{Token: RandomToken(32), User: user, Role: role, CSRF: RandomToken(24), Created: now, LastSeen: now}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[sess.Token] = sess
	return sess
}

func (s *Sessions) Get(token string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.m[token]
	if !ok {
		return nil
	}
	now := time.Now()
	if now.Sub(sess.LastSeen) > IdleTimeout || now.Sub(sess.Created) > AbsoluteTimeout {
		delete(s.m, token)
		return nil
	}
	sess.LastSeen = now
	c := *sess
	return &c
}

func (s *Sessions) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, token)
}

// DeleteUser ends every session of a user (password change, deletion).
func (s *Sessions) DeleteUser(user string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.m {
		if v.User == user {
			delete(s.m, k)
		}
	}
}

// Limiter throttles failed logins per client address with exponential
// lock-out: 5 free attempts, then 1 min, 2 min, ... capped at 15 min.
type Limiter struct {
	mu sync.Mutex
	m  map[string]*attempts
}

type attempts struct {
	fails int
	until time.Time
}

func NewLimiter() *Limiter { return &Limiter{m: map[string]*attempts{}} }

func (l *Limiter) Allowed(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	a := l.m[key]
	if a == nil || now.After(a.until) {
		return true, 0
	}
	return false, a.until.Sub(now)
}

func (l *Limiter) Fail(key string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	a := l.m[key]
	if a == nil {
		a = &attempts{}
		l.m[key] = a
	}
	a.fails++
	if a.fails >= 5 {
		d := time.Minute << min(a.fails-5, 4)
		a.until = now.Add(min(d, 15*time.Minute))
	}
}

func (l *Limiter) Success(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.m, key)
}
